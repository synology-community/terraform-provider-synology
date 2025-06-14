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
			r.UnitTest(t, r.TestCase{
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
