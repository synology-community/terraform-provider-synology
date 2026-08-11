package core_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccShareResource_basic(t *testing.T) {
	name := "tfacc_share_basic"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: fmt.Sprintf(`
resource "synology_core_share" "test" {
  name     = %q
  vol_path = "/volume1"
  desc     = "tfacc share"
  hidden   = false
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_core_share.test", "name", name),
					r.TestCheckResourceAttr("synology_core_share.test", "desc", "tfacc share"),
					r.TestCheckResourceAttrSet("synology_core_share.test", "uuid"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "synology_core_share" "test" {
  name     = %q
  vol_path = "/volume1"
  desc     = "tfacc share updated"
  hidden   = true
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_core_share.test", "desc", "tfacc share updated"),
					r.TestCheckResourceAttr("synology_core_share.test", "hidden", "true"),
				),
			},
			{
				ResourceName:                         "synology_core_share.test",
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore: []string{
					"enable_recycle_bin", "recycle_bin_admin_only", "enable_share_compress",
				},
			},
		},
	})
}
