package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig,
				Check: resource.ComposeTestCheckFunc(
					// Just verify provider configures successfully
					func(s *terraform.State) error {
						return nil
					},
				),
			},
		},
	})
}

const testAccProviderConfig = `
provider "synology" {}
`
