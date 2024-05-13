package virtualization_test

import (
	"fmt"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type GuestResource struct{}

func TestAccGuestResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"guest name is set",
			`
			resource "synology_virtualization_guest" "foo" {
				name         = "testvm"
				storage_name = "default"
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			r.UnitTest(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.ResourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith("synology_virtualization_guest.foo", "name", func(attr string) error {
								if attr != "testiso" {
									return fmt.Errorf("expected guest name to be 'testiso', got %s", attr)
								}
								return nil
							}),
						),
					},
				},
			})
		})
	}
}
