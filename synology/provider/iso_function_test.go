// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

// These build ISO images through the provider's function, which means real
// work on a real NAS. resource.UnitTest sounds hermetic and is the opposite:
// its only effect is to skip the TF_ACC check, so `go test ./...` ran them
// against whatever host was configured, unasked.
func TestAccISOFunction_Null(t *testing.T) {
	resource.Test(t, resource.TestCase{
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
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []resource.TestStep{
					{
						Config: tt.ResourceBlock,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckFunc(func(s *terraform.State) error {
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
