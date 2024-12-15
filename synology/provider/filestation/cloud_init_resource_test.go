package filestation_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type CloudInitResource struct{}

func TestAccCloudInitResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"file name is set",
			`
			resource "synology_filestation_cloud_init" "foo" {
				path = "/data/foo/bar/test.iso"
				create_parents = true
				overwrite = true
				user_data = "#cloud-config\n\nusers:\n  - name: test\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh_authorized_keys:\n      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDf7"
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
							r.TestCheckResourceAttrWith("synology_filestation_cloud_init.foo", "path", func(attr string) error {
								if attr != "/data/foo/bar/test.iso" {
									return fmt.Errorf("expected file path to be '/data/foo/bar/test.iso', got %s", attr)
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
