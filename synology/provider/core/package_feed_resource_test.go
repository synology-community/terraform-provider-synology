package core_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type PackageFeedResource struct{}

func TestAccPackageFeedResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		// {
		// 	"package name is set",
		// 	`
		// 	resource "synology_core_package" "foo" {
		// 		name = "exFAT-Free"
		// 	}`,
		// },
		{
			"package name is set",
			`
			resource "synology_core_package_feed" "foo" {
				name = "Homebridge"
				url  = "https://synology.homebridge.io"
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
								"synology_core_package_feed.foo",
								"url",
								func(attr string) error {
									if attr == "" {
										return fmt.Errorf("expected package version to be set")
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

func TestAccPackageFeedResource_url(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"package url is set",
			`
			resource "synology_core_package" "noble" {
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
							r.TestCheckResourceAttrWith(
								"synology_core_package.noble",
								"url",
								func(attr string) error {
									if attr != "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img" {
										return fmt.Errorf(
											"expected package url to be 'https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img', got %s",
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
