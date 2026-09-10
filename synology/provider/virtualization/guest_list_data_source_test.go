package virtualization_test

import (
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type GuestListDataSource struct{}

func TestAccGuestListDataSource_basic(t *testing.T) {
	testCases := []struct {
		Name            string
		DataSourceBlock string
	}{
		{
			"contains a guest",
			`data "synology_virtualization_guest_list" "all" {}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			// r.UnitTest's only effect versus r.Test is skipping the TF_ACC
			// check, so this listed real VM guests off the live NAS on any
			// `go test ./...`, unit-test name notwithstanding.
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.DataSourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrSet(
								"data.synology_virtualization_guest_list.all",
								"guest.0.%",
							),
						),
					},
				},
			})
		})
	}
}
