package filestation_test

import (
	"fmt"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type FileResource struct{}

func TestAccFileResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"file name is set",
			`
			resource "synology_filestation_file" "foo" {
				path = "/data/foo/bar/file.txt"
				create_parents = true
				overwrite = true
				content = "Hello, World!"
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
							r.TestCheckResourceAttrWith("synology_filestation_file.foo", "path", func(attr string) error {
								if attr != "/data/foo/bar/file.txt" {
									return fmt.Errorf("expected file path to be '/data/foo/bar/file.txt', got %s", attr)
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
