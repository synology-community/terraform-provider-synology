package core_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccGroupResource_basic(t *testing.T) {
	name := "tfacc_group_basic"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: fmt.Sprintf(`
resource "synology_core_group" "test" {
  name        = %q
  description = "tfacc group"
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_core_group.test", "name", name),
					r.TestCheckResourceAttr("synology_core_group.test", "description", "tfacc group"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "synology_core_group" "test" {
  name        = %q
  description = "tfacc group updated"
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_core_group.test", "description", "tfacc group updated"),
				),
			},
			{
				ResourceName:                         "synology_core_group.test",
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"description", "gid"},
			},
		},
	})
}
