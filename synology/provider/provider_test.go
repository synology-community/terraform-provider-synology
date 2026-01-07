package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/testcontainers/testcontainers-go"
	testcompose "github.com/testcontainers/testcontainers-go/modules/compose"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"synology": providerserver.NewProtocol6WithError(New()()),
}

var testAccProtoV6ProviderFactories = providerFactories

var testClient *client.Client

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

type logConsumer struct {
	StdOut bool
}

func (l *logConsumer) Accept(logEntry testcontainers.Log) {
	if logEntry.LogType == testcontainers.StdoutLog && l.StdOut {
		fmt.Printf("[DSM] %s", logEntry.Content)
	}
	if logEntry.LogType == testcontainers.StderrLog {
		fmt.Printf("[DSM ERROR] %s", logEntry.Content)
	}
}

func runAcceptanceTests(m *testing.M) int {
	// Disable Ryuk reaper to avoid connection issues in local development
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		panic(err)
	}

	dc, err := testcompose.NewDockerCompose("../../docker-compose.yaml")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = dc.WithOsEnv().Up(ctx, testcompose.Wait(true)); err != nil {
		panic(err)
	}

	defer func() {
		if err := dc.Down(context.Background(), testcompose.RemoveOrphans(true), testcompose.RemoveImagesLocal); err != nil {
			panic(err)
		}
	}()

	container, err := dc.ServiceContainer(ctx, "dsm")
	if err != nil {
		panic(err)
	}

	lc := &logConsumer{StdOut: os.Getenv("DSM_STDOUT") != ""}
	container.FollowOutput(lc)
	if err := container.StartLogProducer(ctx); err != nil {
		fmt.Printf("Warning: Could not start log producer: %v\n", err)
	}
	defer container.StopLogProducer()

	endpoint, err := container.PortEndpoint(ctx, "5000/tcp", "http")
	if err != nil {
		panic(err)
	}

	// Convert to HTTPS endpoint
	httpsEndpoint, err := container.PortEndpoint(ctx, "5001/tcp", "https")
	if err != nil {
		// Fallback to HTTP if HTTPS not available
		httpsEndpoint = endpoint
		fmt.Printf("Warning: HTTPS port not available, using HTTP: %s\n", endpoint)
	}

	const user = "admin"
	const password = "synology"

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
	cli, err := client.New(api.Options{
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
	if err = waitForDSMAPI(ctx, testClient, user, password); err != nil {
		panic(err)
	}

	return m.Run()
}

func importStep(name string, ignore ...string) resource.TestStep {
	step := resource.TestStep{
		ResourceName:      name,
		ImportState:       true,
		ImportStateVerify: true,
	}

	if len(ignore) > 0 {
		step.ImportStateVerifyIgnore = ignore
	}

	return step
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
func waitForDSMAPI(ctx context.Context, client *client.Client, user, password string) error {
	maxRetries := 120
	retryDelay := 5 * time.Second

	fmt.Printf(
		"Waiting for DSM API to be ready (max %d attempts, %v between attempts)...\n",
		maxRetries,
		retryDelay,
	)

	for i := range maxRetries {
		_, err := client.Login(ctx, api.LoginOptions{
			Username: user,
			Password: password,
		})
		if err == nil {
			fmt.Printf("✓ DSM API is ready after %d attempts\n", i+1)
			return nil
		}

		// Check if it's a login error (expected during setup) vs connection error
		if i < maxRetries-1 {
			if (i+1)%10 == 0 {
				fmt.Printf(
					"Still waiting... (attempt %d/%d): %v\n",
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
