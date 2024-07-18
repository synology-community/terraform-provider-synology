package core_test

import (
	"fmt"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type PackageResource struct{}

func TestAccPackageResource_basic(t *testing.T) {
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
		// {
		// 	"package name is set",
		// 	`
		// 	resource "synology_core_package" "foo" {
		// 		name = "transmission"
		// 	}`,
		// },
		{
			"mariadb",
			`
			resource "synology_core_package" "mariadb" {
				name = "MariaDB10"

				wizard = {
					pkgwizard_port              = 3306
					pkgwizard_new_root_password = "T3stP@ssw0rd"
				}
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
							r.TestCheckResourceAttrWith("synology_core_package.foo", "version", func(attr string) error {
								if attr == "" {
									return fmt.Errorf("expected package version to be set")
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

func TestAccPackageResource_url(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"package url is set",
			`
			resource "synology_core_package" "noble" {
				name = "vault"
				url = "https://synology-community.github.io/spksrc/packages/vault_1.17.2_linux_amd64.spk"
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
							r.TestCheckResourceAttrWith("synology_core_package.noble", "url", func(attr string) error {
								if attr != "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img" {
									return fmt.Errorf("expected package url to be 'https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img', got %s", attr)
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
