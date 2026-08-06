package provider_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type ImageResource struct{}

func TestAccApiResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"System returns info",
			`
			resource "synology_api" "foo" {
				api        = "SYNO.Core.System"
				method     = "info"
				version    = 1
				parameters = {
					"query" = "all"
				}
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			// r.UnitTest sets IsUnitTest but still calls r.Test underneath, and
			// its only real effect is bypassing the TF_ACC environment check —
			// so despite the name this ran real API calls against live hardware
			// with no opt-in. Use r.Test so TF_ACC gates it like every other
			// acceptance test.
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.ResourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_api.foo",
								"result.cpu_vendor",
								func(attr string) error {
									if attr == "\"AMD\"" {
										return nil
									}
									if len(attr) == 0 {
										return fmt.Errorf("expected result to be populated")
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
