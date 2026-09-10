package virtualization_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type ImageResource struct{}

func TestAccImageResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"image name is set",
			`
			resource "synology_virtualization_image" "foo" {
				name       = "testiso"
				path       = "/data/cluster_storage/commoninit.iso"
				image_type = "iso"
				auto_clean = true
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			// r.UnitTest is really r.Test with the TF_ACC opt-in check
			// removed, not a unit test — this uploaded a real virtualization
			// image to the NAS with no gate. r.Test puts that gate back.
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.ResourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_virtualization_image.foo",
								"name",
								func(attr string) error {
									if attr != "testiso" {
										return fmt.Errorf(
											"expected image name to be 'testiso', got %s",
											attr,
										)
									}
									return nil
								},
							),
						),
					},
				},
			})
		})
	}
}
