package container_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
			// r.UnitTest is not actually a unit test: it only skips the
			// TF_ACC check that r.Test performs, so this built and ran a real
			// container project on the NAS with no opt-in required. r.Test
			// puts the TF_ACC gate back.
			r.Test(t, r.TestCase{
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

// TestAccProjectResource_import creates a project from a compose file, imports
// it by name, and checks that the import holds everything the created
// resource did. A later change to the compose file's environment must then
// plan as an in-place update, not a replacement.
func TestAccProjectResource_import(t *testing.T) {
	const name = "tfacc-import"
	projectConfig := func(greeting string) string {
		return fmt.Sprintf(`
resource "synology_container_project" "default" {
  name    = %q
  run     = true
  content = <<-YAML
    services:
      web:
        image: nginx:alpine
        environment:
          GREETING: %q
        ports:
          - "8558:80"
  YAML
}
`, name, greeting)
	}

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: projectConfig("hello"),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("synology_container_project.default", "status", "RUNNING"),
					r.TestCheckResourceAttrSet("synology_container_project.default", "share_path"),
				),
			},
			{
				ResourceName:            "synology_container_project.default",
				ImportState:             true,
				ImportStateId:           name,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			{
				Config: projectConfig("hello again"),
				ConfigPlanChecks: r.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"synology_container_project.default",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				Check: r.TestCheckResourceAttr("synology_container_project.default", "status", "RUNNING"),
			},
		},
	})
}
