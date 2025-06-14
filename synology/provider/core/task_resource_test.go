package core_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type TaskResource struct{}

func TestAccTaskResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"test",
			`
			resource "synology_core_task" "test" {
				name = "Test Test"

				script = "echo 'Hello, World!'"
				user = "terraform"

				schedule = "0 0 1 * *"

				run = true
				when = "apply"
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
								"synology_core_task.test",
								"id",
								func(attr string) error {
									if attr == "" {
										return fmt.Errorf("expected task id to be set")
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
