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

				vcpu_num  = 4
				vram_size = 4096

				network {
					name = "default"
				}

				disk {
					create_type = 0
					size        = 20000
				}

				iso_image {
					id = "65caaef4-6622-4643-9feb-1c3b5a915eb8"
				}

				iso_image {
					id = "unmounted"
				}
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
								if attr != "testvm" {
									return fmt.Errorf("expected guest name to be 'testvm', got %s", attr)
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
