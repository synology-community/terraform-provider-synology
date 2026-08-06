package core_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
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
				name   = "Test"
				script = "echo 'Hello, World!'"
				user   = "root"
				run    = true
				when   = ["apply"]
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			// r.UnitTest only skips the TF_ACC gate that r.Test enforces; it
			// does not make this a unit test. Left as-is, this would schedule
			// and run a real DSM event/script against live hardware on every
			// `go test ./...`. Switch to r.Test to require explicit opt-in.
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.ResourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_core_event.test",
								"id",
								func(attr string) error {
									if attr == "" {
										return fmt.Errorf("expected event id to be set")
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
