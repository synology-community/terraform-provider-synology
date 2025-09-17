package container_test

import (
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

const (
	basicNetwork = `
resource "synology_container_network" "test" {
	name    = "test-network"
	subnet  = "172.90.0.1/20"
	gateway = "172.90.0.1"
}
`
)

func TestAccNetworkResource_basic(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: basicNetwork,
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr(
						"synology_container_network.test",
						"name",
						"test-network",
					),
					r.TestCheckResourceAttr("synology_container_network.test", "driver", "bridge"),
					r.TestCheckResourceAttrSet("synology_container_network.test", "id"),
				),
				ResourceName:  "synology_container_network.test",
				ImportStateId: "test-network",
				ImportState:   false,
			},
		},
	})
}
