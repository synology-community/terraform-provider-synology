package container_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

type ProjectResource struct{}

func TestAccProjectResource_basic(t *testing.T) {
	testCases := []struct {
		Name         string
		ResourceFile config.TestStepConfigFunc
	}{
		{
			Name: "secrets project",
			ResourceFile: config.TestStepConfigFunc(
				func(req config.TestStepConfigRequest) string {
					return "fixtures/secret_project.tf"
				}),
		},
		{
			Name: "basic project",
			ResourceFile: config.TestStepConfigFunc(
				func(req config.TestStepConfigRequest) string {
					return "fixtures/basic_project.tf"
				}),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			r.UnitTest(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						ConfigFile: tt.ResourceFile,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_container_project.default",
								"name",
								func(attr string) error {
									if attr != tt.Name {
										return fmt.Errorf(
											"expected project name to be '%s', got %s",
											tt.Name,
											attr,
										)
									}
									return nil
								},
							),
							r.TestCheckResourceAttrWith(
								"synology_container_project.default",
								"content",
								func(attr string) error {
									if len(attr) < 1 {
										return fmt.Errorf(
											"expected resource to contain content, got %s",
											attr,
										)
									}
									return nil
								},
							),
						),
					},
					{
						ConfigFile: tt.ResourceFile,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_container_project.default",
								"name",
								func(attr string) error {
									if attr != tt.Name {
										return fmt.Errorf(
											"expected project name to be 'homebridge', got %s",
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
