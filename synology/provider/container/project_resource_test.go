package container_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

const (
	testProject = `
	provider "synology" {
		host            = "https://nas.appkins.io:5001"
		user            = "terraform"
		password        = "ABP8kdn7teu.fck-kzk"
		skip_cert_check = true
  	otp_secret      = ""
  }

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
	homebridgeProject = `
	resource "synology_container_project" "default" {
		name = "homebridge"

		services = {
			homebridge = {
				name     = "homebridge"
				replicas = 1

				image = "homebridge/homebridge:latest"

				network_mode = "host"

				healthcheck = {
					test         = [
						"curl --fail localhost:8581 || exit 1"
					]
					interval     = "60s"
					retries      = 5
					start_period = "300s"
					timeout      = "2s"
				}

				volumes = [
					{
						source = "/volume1/docker/homebridge"
						target = "/homebridge"
						bind {
							create_host_path = true
						}
					}
				]
			}
		}
	}`

	traefikProject = `
resource "synology_container_project" "default" {
	name = "traefik"

	services = {
		traefik = {
			name     = "traefik"
			replicas = 1

			image = "traefik:v2.4"

			network_mode = "host"

		}
	}
}

import {
	to = synology_container_project.default
	id = "traefik"
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
			"traefik",
			traefikProject,
		},
		{
			"foo",
			testProject,
		},
		{
			"k3s",
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
