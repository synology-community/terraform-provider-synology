package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	synologyapi "github.com/synology-community/go-synology/pkg/api"
	"github.com/testcontainers/testcontainers-go"
	testcompose "github.com/testcontainers/testcontainers-go/modules/compose"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"synology": providerserver.NewProtocol6WithError(New()()),
}

var testAccProtoV6ProviderFactories = providerFactories

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

type logConsumer struct {
	StdOut bool

	ctx context.Context
}

func (l *logConsumer) Accept(logEntry testcontainers.Log) {
	switch logEntry.LogType {
	case testcontainers.StdoutLog:
		tflog.Info(l.ctx, string(logEntry.Content))
	case testcontainers.StderrLog:
		tflog.Error(l.ctx, string(logEntry.Content))
	}
}

func runAcceptanceTests(m *testing.M) int {
	// Disable Ryuk reaper to avoid connection issues in local development
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := NewLogger(ctx)

	_ = testcompose.WithLogger(logger)

	dc, err := testcompose.NewDockerCompose("../../docker-compose.yaml")
	if err != nil {
		panic(err)
	}

	// Don't wait for health check in compose.Up - we'll do our own waiting with waitForDSMAPI
	// The health check may have a long start_period which can cause timeouts in testcontainers
	if err = dc.WithOsEnv().
		Up(ctx, testcompose.Wait(true), testcompose.WithRecreate(api.RecreateDiverged)); err != nil {
		panic(err)
	}

	defer func() {
		logger.Printf("RUNNING TEAR DOWN")
		if err := dc.Down(
			context.Background(),
			testcompose.RemoveOrphans(true),
			testcompose.RemoveImagesLocal,
		); err != nil {
			panic(err)
		}
	}()

	container, err := dc.ServiceContainer(ctx, "dsm")
	if err != nil {
		panic(err)
	}

	lc := &logConsumer{StdOut: os.Getenv("DSM_STDOUT") != "", ctx: ctx}

	testcontainers.WithLogConsumers(lc)

	// Get the host that the container is accessible from
	host, err := container.Host(ctx)
	if err != nil {
		panic(err)
	}

	// Get the mapped port for 5000
	mappedPort, err := container.MappedPort(ctx, "5000/tcp")
	if err != nil {
		panic(err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
	logger.Printf("DSM endpoint: %s", endpoint)

	// Get the mapped port for 5000
	mappedPortHttps, err := container.MappedPort(ctx, "5001/tcp")
	if err != nil {
		panic(err)
	}

	// Convert to HTTPS endpoint
	httpsEndpoint := fmt.Sprintf("https://%s:%s", host, mappedPortHttps.Port())
	logger.Printf("DSM Https Endpoint: %s", httpsEndpoint)

	const user = "admin"
	const password = "synology"

	logger.Printf("DSM endpoint: %s", httpsEndpoint)

	if err = os.Setenv("SYNOLOGY_HOST", httpsEndpoint); err != nil {
		panic(err)
	}

	if err = os.Setenv("SYNOLOGY_USER", user); err != nil {
		panic(err)
	}

	if err = os.Setenv("SYNOLOGY_PASSWORD", password); err != nil {
		panic(err)
	}

	if err = os.Setenv("SYNOLOGY_SKIP_CERT_CHECK", "true"); err != nil {
		panic(err)
	}

	// Initialize test client
	cli, err := client.New(synologyapi.Options{
		Host:       httpsEndpoint,
		VerifyCert: false,
		RetryLimit: 5,
	})
	if err != nil {
		panic(err)
	}

	testClient, ok := cli.(*client.Client)
	if !ok {
		panic("failed to cast client")
	}

	// Wait for DSM API to be ready
	if err = waitForDSMAPI(ctx, logger, testClient, user, password); err != nil {
		panic(err)
	}

	return m.Run()
}

func preCheck(t *testing.T) {
	variables := []string{
		"SYNOLOGY_HOST",
		"SYNOLOGY_USER",
		"SYNOLOGY_PASSWORD",
	}

	for _, variable := range variables {
		value := os.Getenv(variable)
		if value == "" {
			t.Fatalf("`%s` must be set for acceptance tests!", variable)
		}
	}
}

// waitForDSMAPI waits for the DSM API to be ready and accepting requests
// This is necessary because the container may report as healthy before the API is fully initialized.
func waitForDSMAPI(
	ctx context.Context,
	logger *SynoLogger,
	client *client.Client,
	user, password string,
) error {
	maxRetries := 120
	retryDelay := 5 * time.Second

	logger.Printf(
		"Waiting for DSM API to be ready (max %d attempts, %v between attempts)...",
		maxRetries,
		retryDelay,
	)

	for i := range maxRetries {
		_, err := client.Login(ctx, synologyapi.LoginOptions{
			Username: user,
			Password: password,
		})
		if err == nil {
			logger.Printf("✓ DSM API is ready after %d attempts", i+1)
			return nil
		}

		// Check if it's a login error (expected during setup) vs connection error
		if i < maxRetries-1 {
			if (i+1)%10 == 0 {
				logger.Printf(
					"Still waiting... (attempt %d/%d): %v",
					i+1,
					maxRetries,
					err.Error(),
				)
			}
			time.Sleep(retryDelay)
			continue
		}

		return fmt.Errorf(
			"DSM API did not become ready after %d attempts (waited %v): %w",
			maxRetries,
			time.Duration(maxRetries)*retryDelay,
			err,
		)
	}

	return fmt.Errorf("DSM API did not become ready after %d attempts", maxRetries)
}
