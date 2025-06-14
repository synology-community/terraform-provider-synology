package filestation_test

import (
	"fmt"
	"regexp"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type FolderResource struct{}

func TestAccFolderResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"folder path is set",
			`
			resource "synology_filestation_folder" "default" {
				path = "/docker/foo/bar"
				create_parents = true
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
								"synology_filestation_folder.default",
								"path",
								func(attr string) error {
									if attr != "/docker/foo/bar" {
										return fmt.Errorf(
											"expected folder path to be '/docker/foo/bar', got %s",
											attr,
										)
									}
									return nil
								},
							),
						),
					},
					{
						Config: tt.ResourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_filestation_folder.default",
								"real_path",
								func(attr string) error {
									exp := regexp.MustCompile(`^/volume[0-9]{1,9}/docker/foo/bar$`)
									if !exp.MatchString(attr) {
										return fmt.Errorf(
											"expected real path to contain the /volume prefix, got %s",
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
