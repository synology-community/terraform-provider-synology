package container_test

import (
	"fmt"
	"testing"

	"github.com/appkins/terraform-provider-synology/synology/acctest"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	homebridgeProject = `
	resource "synology_container_project" "foo" {
		name = "homebridge"

		service {
			name     = "homebridge"
			replicas = 1

			image {
				name = "homebridge/homebridge"
				tag  = "latest"
			}

			network_mode = "host"

			health_check {
				test         = [
					"curl --fail localhost:8581 || exit 1"
				]
				interval     = "60s"
				retries      = 5
				start_period = "300s"
				timeout      = "2s"
			}

			volume {
				source = "/volume1/docker/homebridge"
				target = "/homebridge"
				bind {
					create_host_path = true
				}
			}
		}
	}`
	k3sProject = `
	resource "synology_container_project" "foo" {
		name = "k3s"

		service {
			name     = "server"
			replicas = 1

			image {
				name = "rancher/k3s"
				tag  = "latest"
			}

			network_mode = "bridge"

			environment = {
				"K3S_TOKEN"             = "token"
				"K3S_KUBECONFIG_OUTPUT" = "/output/kubeconfig.yaml"
				"K3S_KUBECONFIG_MODE"   = "666"
			}

			port {
				target    = 6443
				published = 6443
			}

			port {
				target    = 80
				published = 8088
			}

			port {
				target    = 443
				published = 8443
			}

			ulimit {
				name = "nofile"
				soft = 65535
				hard = 65535
			}

			ulimit {
				name  = "nproc"
				value = 65535
			}

			tmpfs = [
				"/run",
				"/var/run"
			]

			restart = "unless-stopped"

			# volume {
			# 	source = "k3s-server"
			# 	target = "/var/lib/rancher/k3s"
	    # }

			volume {
				source = "/volume1/docker/k3s"
				target = "/output"
				type   = "bind"
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
			"k3s project",
			k3sProject,
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
							r.TestCheckResourceAttrWith("synology_container_project.foo", "name", func(attr string) error {
								if attr != "homebridge" {
									return fmt.Errorf("expected project name to be 'homebridge', got %s", attr)
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
