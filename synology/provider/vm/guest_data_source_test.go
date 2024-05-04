package vm_test

import (
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
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
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
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
