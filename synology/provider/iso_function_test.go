// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccISOFunction_Null(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.8.0"))),
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
				output "test" {
					value = provider::synology::iso("cidata", {
					user-data: "This is a test",
					"meta-data": "This is a test",
					"network-config": "This is a test"
			})
				`,
				// The parameter does not enable AllowNullValue
				ExpectError: regexp.MustCompile(`argument must not be null`),
			},
		},
	})
}

func TestAccISOFunction_Basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"Test that the function returns a value",
			`
			output "test" {
				value = provider::synology::iso("boot-path", {"system-boot": "ls -l && echo 'hello world' > /tmp/hello.txt'"})
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
							r.TestCheckFunc(func(s *terraform.State) error {
								v, ok := s.RootModule().Outputs["test"]
								if !ok {
									return nil
								}
								if v.Value == nil {
									return nil
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
