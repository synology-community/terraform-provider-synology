package container_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

const (
	testProject = `
	resource "synology_container_project" "default" {
		name = "foo"

		networks = {
			foo = {
				driver = "bridge"
			}

			bar = {
				driver = "macvlan"
				driver_opts = {
					"parent" = "ovs_bond0"
				}
				ipam = {
					driver = "macvlan"
					config = [{
						subnet  = "10.0.0.0/16"
						gateway = "10.0.0.1"
						ip_range = "10.0.60.1/28"
						aux_address = {
							host = "10.0.60.2"
						}
					}]
				}
			}
		}
		configs = {
			foo = {
				name    = "foo.txt"
				content = "Hello World"
			}
		}
		services = {
			bar = {
				name     = "bar"
				replicas = 1

				image = "nginx"

				logging = {
					driver = "json-file"
				}

				configs = [
					{
						source = "foo"
						target = "/etc/foo.txt"
						mode   = "777"
					}
				]

				ports = [{
					target    = 80
					published = "8557"
					protocol  = "tcp"
				}]

				networks = {
					foo = {}
				}
			}
		}
	}`
)

type ProjectResource struct{}

func TestAccProjectResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"basic project",
			testProject,
		},
		// {
		// 	"homebridge project",
		// 	homebridgeProject,
		// },
		// {
		// 	"project name is set",
		// 	`
		// 	resource "synology_container_project" "foo" {
		// 		name = "foo"

		// 		network {
		// 			name   = "foo"
		// 			driver = "bridge"
		// 		}

		// 		service {
		// 			name     = "bar"
		// 			replicas = 1

		// 			image {
		// 				name = "nginx"
		// 			}

		// 			logging {
		// 				driver = "json-file"
		// 			}

		// 			port {
		// 				target    = 80
		// 				published = "8557"
		// 				protocol  = "tcp"
		// 			}

		// 			network {
		// 				name = "foo"
		// 			}
		// 		}
		// 	}`,
		// },
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
						Config: tt.ResourceBlock,
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
