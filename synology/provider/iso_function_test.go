// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"regexp"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestISOFunction_Known(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.8.0"))),
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
				output "test" {
					value = provider::synology::iso("testiso", { "/testfile.txt": "contents of file" })
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("test", "contents of file"),
				),
			},
		},
	})
}

func TestISOFunction_Null(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.8.0"))),
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
				output "test" {
					value = provider::synology::iso(null, null)
				}
				`,
				// The parameter does not enable AllowNullValue
				ExpectError: regexp.MustCompile(`argument must not be null`),
			},
		},
	})
}

func TestISOFunction_Unknown(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.8.0"))),
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
				output "test" {
					value = base64encode(provider::synology::iso("testvalue", { "/testfile.txt" = "contents of file" }))
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchOutput("test", regexp.MustCompile(`.*`)),
				),
			},
		},
	})
}
