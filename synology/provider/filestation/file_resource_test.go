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

func TestAccFileResource_url(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"file url is set",
			`
			resource "synology_filestation_file" "noble" {
				path = "/data/cluster_storage/noble-server-cloudimg-amd64.img"
				url = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
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
							r.TestCheckResourceAttrWith("synology_filestation_file.noble", "url", func(attr string) error {
								if attr != "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img" {
									return fmt.Errorf("expected file url to be 'https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img', got %s", attr)
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
