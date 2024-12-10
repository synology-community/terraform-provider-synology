package core_test

import (
	"fmt"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type EventResource struct{}

func TestAccEventResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"test",
			`
			resource "synology_core_event" "test" {
				name = "Test"

				script = "echo 'Hello, World!'"
				user = "root"

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
							r.TestCheckResourceAttrWith("synology_core_event.test", "id", func(attr string) error {
								if attr == "" {
									return fmt.Errorf("expected event id to be set")
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
