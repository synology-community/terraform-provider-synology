package container_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccRegistryResource_basic(t *testing.T) {
	name := "tfacc_registry_basic"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: fmt.Sprintf(`
resource "synology_container_registry" "test" {
  name = %q
  url  = "https://example.com/tfacc"
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_container_registry.test", "name", name),
					r.TestCheckResourceAttr("synology_container_registry.test", "url", "https://example.com/tfacc"),
					r.TestCheckResourceAttr("synology_container_registry.test", "syno", "false"),
				),
			},
			{
				ResourceName:                         "synology_container_registry.test",
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}
