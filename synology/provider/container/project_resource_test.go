package container_test

import (
	"fmt"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type ProjectResource struct{}

func TestAccProjectResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"project name is set",
			`
			resource "synology_container_project" "foo" {
				name = "foo"
				
				network {
					name   = "foo"
					driver = "bridge"
				}

				service {
					name     = "bar"
					replicas = 1
					
					image {
						name = "nginx"
					}

					logging {
						driver = "json-file"
					}

					port {
						target    = 80
						published = "8557"
						protocol  = "tcp"
					}

					network {
						name = "foo"
					}
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
							r.TestCheckResourceAttrWith("synology_container_project.foo", "name", func(attr string) error {
								if attr != "foo" {
									return fmt.Errorf("expected project name to be 'foo', got %s", attr)
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
