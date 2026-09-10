package acctest

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/synology-community/terraform-provider-synology/synology/provider"
)

// ProtoV6ProviderFactories is used to set ProtoV6ProviderFactories in a
// resource.TestCase. It skips when the NAS credentials are unset: community CI
// always sets TF_ACC=1, but fork PRs do not receive repository secrets, so
// every TestAcc* otherwise fails with "host information is not provided".
func ProtoV6ProviderFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	TestAccPreCheck(t)
	return map[string]func() (tfprotov6.ProviderServer, error){
		"synology": providerserver.NewProtocol6WithError(provider.New()()),
	}
}

func TestAccPreCheck(t *testing.T) {
	t.Helper()
	for _, variable := range []string{"SYNOLOGY_HOST", "SYNOLOGY_USER", "SYNOLOGY_PASSWORD"} {
		if os.Getenv(variable) == "" {
			t.Skipf("%s is not set; skipping acceptance tests", variable)
		}
	}
}
