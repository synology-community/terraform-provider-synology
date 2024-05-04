package provider_test

import (
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type GuestDataSource struct{}

func TestAccGuestDataSource_basic(t *testing.T) {
	testCases := []struct {
		Name            string
		DataSourceBlock string
	}{
		{
			"id exists for known guest",
			`
			data "synology_guests" "foo" {}

			data "synology_guest" "foo" {
				name = data.synology_guests.foo.guest.0.name
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			r.UnitTest(t, r.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []r.TestStep{
					{
						Config: tt.DataSourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrSet("data.synology_guest.foo", "id"),
						),
					},
				},
			})
		})
	}
}
