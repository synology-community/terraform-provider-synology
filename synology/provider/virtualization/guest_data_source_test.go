package virtualization_test

import (
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
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
			data "synology_virtualization_guest_list" "all" {}

			data "synology_virtualization_guest" "foo" {
				name = data.synology_virtualization_guest_list.all.guest[0].name
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			// r.UnitTest only skips the TF_ACC check that r.Test performs; it
			// is not a lightweight unit test. This queries a real VM guest on
			// the NAS, so it needs the same opt-in as any acceptance test.
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.DataSourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrSet(
								"data.synology_virtualization_guest.foo",
								"id",
							),
						),
					},
				},
			})
		})
	}
}
