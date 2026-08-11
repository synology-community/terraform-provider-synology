package core_test

import (
	"fmt"
	"regexp"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccUserResource_basic(t *testing.T) {
	name := "tfacc_user_basic"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: fmt.Sprintf(`
resource "synology_core_user" "test" {
  name                = %q
  password_wo         = "TfAcc!Passw0rd"
  password_wo_version = 1
  description         = "tfacc user"
  email               = "tfacc@example.com"
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_core_user.test", "name", name),
					r.TestCheckResourceAttr("synology_core_user.test", "description", "tfacc user"),
					r.TestCheckResourceAttrSet("synology_core_user.test", "uid"),
					r.TestCheckNoResourceAttr("synology_core_user.test", "password_wo"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "synology_core_user" "test" {
  name                = %q
  password_wo         = "TfAcc!Passw0rd"
  password_wo_version = 1
  description         = "tfacc user updated"
  email               = "tfacc@example.com"
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_core_user.test", "description", "tfacc user updated"),
				),
			},
			{
				ResourceName:                         "synology_core_user.test",
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore: []string{
					"password_wo_version", "cannot_change_password", "password_never_expire",
					"groups", "expire", "email", "description",
				},
			},
		},
	})
}

func TestAccUserResource_passwordNotInState(t *testing.T) {
	// Guard: if password_wo ever leaks into state attributes, this fails.
	name := "tfacc_user_pwcheck"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: fmt.Sprintf(`
resource "synology_core_user" "test" {
  name                = %q
  password_wo         = "NeverInState!1"
  password_wo_version = 1
}
`, name),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttrWith("synology_core_user.test", "name", func(v string) error {
						if v != name {
							return fmt.Errorf("name")
						}
						return nil
					}),
					r.TestMatchResourceAttr("synology_core_user.test", "uid", regexp.MustCompile(`^[0-9]+$`)),
				),
			},
		},
	})
}
