package models

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

type Logging struct {
	Driver  types.String `tfsdk:"driver"`
	Options types.Map    `tfsdk:"options"`
}

type Image struct {
	Name       types.String `tfsdk:"name"`
	Repository types.String `tfsdk:"repository"`
	Tag        types.String `tfsdk:"tag"`
}

type ServiceNetwork struct {
	Name         types.String `tfsdk:"name"`
	Aliases      types.Set    `tfsdk:"aliases"`
	Ipv4Address  types.String `tfsdk:"ipv4_address"`
	Ipv6Address  types.String `tfsdk:"ipv6_address"`
	LinkLocalIPs types.Set    `tfsdk:"link_local_ips"`
	MacAddress   types.String `tfsdk:"mac_address"`
	DriverOpts   types.Map    `tfsdk:"driver_opts"`
	Priority     types.Int64  `tfsdk:"priority"`
}

type Port struct {
	Name        types.String `tfsdk:"name"`
	Target      types.Int64  `tfsdk:"target"`
	Published   types.String `tfsdk:"published"`
	Protocol    types.String `tfsdk:"protocol"`
	AppProtocol types.String `tfsdk:"app_protocol"`
	Mode        types.String `tfsdk:"mode"`
	HostIP      types.String `tfsdk:"host_ip"`
}

type HealthCheck struct {
	Test          types.List           `tfsdk:"test"`
	Interval      timetypes.GoDuration `tfsdk:"interval"`
	Timeout       timetypes.GoDuration `tfsdk:"timeout"`
	StartInterval timetypes.GoDuration `tfsdk:"start_interval"`
	StartPeriod   timetypes.GoDuration `tfsdk:"start_period"`
	Retries       types.Number         `tfsdk:"retries"`
}

type VolumeBind struct {
	Propagation    types.String `tfsdk:"propagation"`
	CreateHostPath types.Bool   `tfsdk:"create_host_path"`
	SELinux        types.String `tfsdk:"selinux"`
}

type ServiceVolume struct {
	Source   types.String `tfsdk:"source"`
	Target   types.String `tfsdk:"target"`
	ReadOnly types.Bool   `tfsdk:"read_only"`
	Bind     types.Set    `tfsdk:"bind"`
	Type     types.String `tfsdk:"type"`
}

type Ulimit struct {
	Name  types.String `tfsdk:"name"`
	Value types.Int64  `tfsdk:"single"`
	Soft  types.Int64  `tfsdk:"soft"`
	Hard  types.Int64  `tfsdk:"hard"`
}

type Service struct {
	Name        types.String `tfsdk:"name"`
	Image       types.Set    `tfsdk:"image"`
	MemLimit    types.String `tfsdk:"mem_limit"`
	Command     types.List   `tfsdk:"command"`
	Replicas    types.Int64  `tfsdk:"replicas"`
	Logging     types.Set    `tfsdk:"logging"`
	Ports       types.Set    `tfsdk:"port"`
	Networks    types.Set    `tfsdk:"network"`
	NetworkMode types.String `tfsdk:"network_mode"`
	HealthCheck types.Set    `tfsdk:"health_check"`
	Volumes     types.Set    `tfsdk:"volume"`
	Privileged  types.Bool   `tfsdk:"privileged"`
	Tmpfs       types.List   `tfsdk:"tmpfs"`
	Ulimits     types.Set    `tfsdk:"ulimit"`
	Environment types.Map    `tfsdk:"environment"`
	Restart     types.String `tfsdk:"restart"`
}

func (m Service) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Service) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"image": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":       types.StringType,
					"repository": types.StringType,
					"tag":        types.StringType,
				},
			},
		},
		"command":      types.ListType{ElemType: types.StringType},
		"restart":      types.StringType, // "no", "always", "on-failure", "unless-stopped", "always", "unless-stopped", "on-failure", "no
		"network_mode": types.StringType,
		"replicas":     types.Int64Type,
		"port": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":         types.StringType,
					"target":       types.Int64Type,
					"published":    types.StringType,
					"protocol":     types.StringType,
					"app_protocol": types.StringType,
					"mode":         types.StringType,
					"host_ip":      types.StringType,
				},
			},
		},
		"health_check": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"test":     types.ListType{ElemType: types.StringType},
					"interval": timetypes.GoDurationType{},
					"timeout":  timetypes.GoDurationType{},
					"retries":  types.Int64Type,
					"start":    timetypes.GoDurationType{},
				},
			},
		},
		"privileged": types.BoolType,
		"tmpfs":      types.ListType{ElemType: types.StringType},
		"network": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":           types.StringType,
					"aliases":        types.SetType{ElemType: types.StringType},
					"ipv4_address":   types.StringType,
					"ipv6_address":   types.StringType,
					"link_local_ips": types.SetType{ElemType: types.StringType},
					"mac_address":    types.StringType,
					"driver_opts":    types.MapType{ElemType: types.StringType},
					"priority":       types.Int64Type,
				},
			},
		},
		"ulimits": types.MapType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"single": types.Int64Type,
					"soft":   types.Int64Type,
					"hard":   types.Int64Type,
				},
			},
		},
		"logging": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"driver": types.StringType,
					"options": types.MapType{
						ElemType: types.StringType,
					},
				},
			},
		},
		"volume": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"source":    types.StringType,
					"target":    types.StringType,
					"read_only": types.BoolType,
					"type":      types.StringType,
					"bind": types.SetType{
						ElemType: types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"propagation":      types.StringType,
								"create_host_path": types.BoolType,
								"selinux":          types.StringType,
							},
						},
					},
				},
			},
		},
		"environment": types.MapType{ElemType: types.StringType},
	}
}

func (m Service) Value() attr.Value {

	var logging basetypes.SetValue
	var image basetypes.SetValue
	var commands basetypes.ListValue
	var ports basetypes.SetValue
	var networks basetypes.SetValue
	var health_check basetypes.SetValue
	var volumes basetypes.SetValue
	var tmpfs basetypes.ListValue
	var ulimits basetypes.SetValue
	var environment basetypes.MapValue

	if l, diag := m.Logging.ToSetValue(context.Background()); !diag.HasError() {
		logging = l
	}

	if i, diag := m.Image.ToSetValue(context.Background()); !diag.HasError() {
		image = i
	}

	if c, diag := m.Command.ToListValue(context.Background()); !diag.HasError() {
		commands = c
	}

	if p, diag := m.Ports.ToSetValue(context.Background()); !diag.HasError() {
		ports = p
	}

	if n, diag := m.Networks.ToSetValue(context.Background()); !diag.HasError() {
		networks = n
	}

	if hc, diag := m.HealthCheck.ToSetValue(context.Background()); !diag.HasError() {
		health_check = hc
	}

	if v, diag := m.Volumes.ToSetValue(context.Background()); !diag.HasError() {
		volumes = v
	}

	if t, diag := m.Tmpfs.ToListValue(context.Background()); !diag.HasError() {
		tmpfs = t
	}

	if u, diag := m.Ulimits.ToSetValue(context.Background()); !diag.HasError() {
		ulimits = u
	}

	if e, diag := m.Environment.ToMapValue(context.Background()); !diag.HasError() {
		environment = e
	}

	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":         types.StringValue(m.Name.ValueString()),
		"image":        image,
		"command":      commands,
		"replicas":     types.Int64Value(m.Replicas.ValueInt64()),
		"port":         ports,
		"network":      networks,
		"logging":      logging,
		"network_mode": types.StringValue(m.NetworkMode.ValueString()),
		"health_check": health_check,
		"volume":       volumes,
		"privileged":   types.BoolValue(m.Privileged.ValueBool()),
		"tmpfs":        tmpfs,
		"ulimit":       ulimits,
		"environment":  environment,
		"restart":      types.StringValue(m.Restart.ValueString()),
	})
}

func (m Service) AsComposeServiceConfig() composetypes.ServiceConfig {
	service := composetypes.ServiceConfig{}

	sName := m.Name.ValueString()

	if !m.MemLimit.IsNull() && !m.MemLimit.IsUnknown() {
		b, err := units.RAMInBytes(m.MemLimit.ValueString())
		if err != nil {
			log.Printf("error parsing memory limit: %v", err)
		} else {
			service.MemLimit = composetypes.UnitBytes(b)
		}
	}

	if !m.Logging.IsNull() && !m.Logging.IsUnknown() {
		service.Logging = &composetypes.LoggingConfig{}
		logging := []Logging{}
		if diag := m.Logging.ElementsAs(context.Background(), &logging, true); !diag.HasError() {
			service.Logging.Driver = logging[0].Driver.ValueString()

			opts := map[string]string{}
			if diag := logging[0].Options.ElementsAs(context.Background(), &opts, true); !diag.HasError() {
				service.Logging.Options = opts
			}
		}
	}

	if !m.Volumes.IsNull() && !m.Volumes.IsUnknown() {
		volumes := []ServiceVolume{}
		if diag := m.Volumes.ElementsAs(context.Background(), &volumes, true); !diag.HasError() {
			service.Volumes = []composetypes.ServiceVolumeConfig{}
			for _, v := range volumes {
				volume := composetypes.ServiceVolumeConfig{
					Source:   v.Source.ValueString(),
					Target:   v.Target.ValueString(),
					ReadOnly: v.ReadOnly.ValueBool(),
					Type:     v.Type.ValueString(),
				}
				if !v.Bind.IsNull() && !v.Bind.IsUnknown() {
					binds := []VolumeBind{}
					if diag := v.Bind.ElementsAs(context.Background(), &binds, true); !diag.HasError() {
						bind := binds[0]
						volume.Bind = &composetypes.ServiceVolumeBind{
							Propagation:    bind.Propagation.ValueString(),
							CreateHostPath: bind.CreateHostPath.ValueBool(),
							SELinux:        bind.SELinux.ValueString(),
						}
					}
				}
				service.Volumes = append(service.Volumes, volume)
			}
		}
	}

	if !m.HealthCheck.IsNull() && !m.HealthCheck.IsUnknown() && len(m.HealthCheck.Elements()) == 1 {
		service.HealthCheck = &composetypes.HealthCheckConfig{}
		healthCheck := []HealthCheck{}
		if diag := m.HealthCheck.ElementsAs(context.Background(), &healthCheck, true); !diag.HasError() {
			hc := healthCheck[0]
			if !hc.Test.IsNull() || !hc.Test.IsUnknown() {
				test := []string{}
				if diag := hc.Test.ElementsAs(context.Background(), &test, true); !diag.HasError() {
					service.HealthCheck.Test = test
				}
			}
			if !hc.Interval.IsNull() || !hc.Interval.IsUnknown() {
				t, diag := hc.Interval.ValueGoDuration()
				if !diag.HasError() {
					ii := composetypes.Duration(t)
					service.HealthCheck.Interval = &ii
				}
			}
			if !hc.Timeout.IsNull() || !hc.Timeout.IsUnknown() {
				t, diag := hc.Timeout.ValueGoDuration()
				if !diag.HasError() {
					ii := composetypes.Duration(t)
					service.HealthCheck.Timeout = &ii
				}
			}
			if !hc.StartInterval.IsNull() || !hc.StartInterval.IsUnknown() {
				t, diag := hc.StartInterval.ValueGoDuration()
				if !diag.HasError() {
					ii := composetypes.Duration(t)
					service.HealthCheck.StartInterval = &ii
				}
			}
			if !hc.StartPeriod.IsNull() || !hc.StartPeriod.IsUnknown() {
				t, diag := hc.StartPeriod.ValueGoDuration()
				if !diag.HasError() {
					ii := composetypes.Duration(t)
					service.HealthCheck.StartPeriod = &ii
				}
			}
			if !hc.Retries.IsNull() || !hc.Retries.IsUnknown() {
				retries, _ := hc.Retries.ValueBigFloat().Uint64()
				service.HealthCheck.Retries = &retries
			}
		}
	}

	if !m.Image.IsNull() && !m.Image.IsUnknown() {
		image := []Image{}
		if diag := m.Image.ElementsAs(context.Background(), &image, true); !diag.HasError() {
			i := image[0]
			iName := i.Name.ValueString()
			var iTag, iRepo string
			if i.Repository.IsNull() || i.Repository.IsUnknown() {
				iRepo = "docker.io"
			} else {
				iRepo = i.Repository.ValueString()
			}
			if i.Tag.IsNull() || i.Tag.IsUnknown() {
				iTag = "latest"
			} else {
				iTag = i.Tag.ValueString()
			}
			service.Image = fmt.Sprintf("%s/%s:%s", iRepo, iName, iTag)
		}
	}

	if !m.Command.IsNull() && !m.Command.IsUnknown() {
		commands := []string{}
		if diag := m.Command.ElementsAs(context.Background(), &commands, true); !diag.HasError() {
			service.Command = commands
		}
	}

	if !m.Ports.IsNull() && !m.Ports.IsUnknown() {
		ports := []Port{}
		if diag := m.Ports.ElementsAs(context.Background(), &ports, true); !diag.HasError() {
			service.Ports = []composetypes.ServicePortConfig{}
			for _, p := range ports {
				port := composetypes.ServicePortConfig{
					Name:        p.Name.ValueString(),
					Target:      uint32(p.Target.ValueInt64()),
					Published:   p.Published.ValueString(),
					Protocol:    p.Protocol.ValueString(),
					AppProtocol: p.AppProtocol.ValueString(),
					Mode:        p.Mode.ValueString(),
					HostIP:      p.HostIP.ValueString(),
				}
				service.Ports = append(service.Ports, port)
			}
		}
	}

	if !m.NetworkMode.IsNull() && !m.NetworkMode.IsUnknown() {
		service.NetworkMode = m.NetworkMode.ValueString()
	}

	if !m.Networks.IsNull() && !m.Networks.IsUnknown() {
		networks := []ServiceNetwork{}
		if diag := m.Networks.ElementsAs(context.Background(), &networks, true); !diag.HasError() {
			service.Networks = map[string]*composetypes.ServiceNetworkConfig{}
			for _, n := range networks {
				network := composetypes.ServiceNetworkConfig{
					Aliases:      []string{},
					Ipv4Address:  n.Ipv4Address.ValueString(),
					Ipv6Address:  n.Ipv6Address.ValueString(),
					LinkLocalIPs: []string{},
					MacAddress:   n.MacAddress.ValueString(),
					DriverOpts:   map[string]string{},
					Priority:     int(n.Priority.ValueInt64()),
				}

				if n.Aliases.IsNull() || n.Aliases.IsUnknown() {
					aliases := []string{}
					if diag := n.Aliases.ElementsAs(context.Background(), &aliases, true); !diag.HasError() {
						network.Aliases = aliases
					}
				}

				if n.LinkLocalIPs.IsNull() || n.LinkLocalIPs.IsUnknown() {
					linkLocalIPs := []string{}
					if diag := n.LinkLocalIPs.ElementsAs(context.Background(), &linkLocalIPs, true); !diag.HasError() {
						network.LinkLocalIPs = linkLocalIPs
					}
				}

				if !n.DriverOpts.IsNull() && !n.DriverOpts.IsUnknown() {
					driverOpts := map[string]string{}
					if diag := n.DriverOpts.ElementsAs(context.Background(), &driverOpts, true); !diag.HasError() {
						network.DriverOpts = driverOpts
					}
				}

				service.Networks[n.Name.ValueString()] = &network
			}
		}
	}

	if !m.Privileged.IsNull() && !m.Privileged.IsUnknown() {
		service.Privileged = m.Privileged.ValueBool()
	}

	if !m.Tmpfs.IsNull() && !m.Tmpfs.IsUnknown() {
		tmpfs := []string{}
		if diag := m.Tmpfs.ElementsAs(context.Background(), &tmpfs, true); !diag.HasError() {
			service.Tmpfs = tmpfs
		}
	}

	if !m.Ulimits.IsNull() && !m.Ulimits.IsUnknown() {
		ulimits := []Ulimit{}
		if diag := m.Ulimits.ElementsAs(context.Background(), &ulimits, true); !diag.HasError() {
			service.Ulimits = map[string]*composetypes.UlimitsConfig{}
			for _, v := range ulimits {
				ulimit := composetypes.UlimitsConfig{}
				if !v.Hard.IsNull() && !v.Hard.IsUnknown() {
					ulimit.Hard = int(v.Hard.ValueInt64())
				}
				if !v.Soft.IsNull() && !v.Soft.IsUnknown() {
					ulimit.Soft = int(v.Soft.ValueInt64())
				}
				if !v.Value.IsNull() && !v.Value.IsUnknown() {
					ulimit.Single = int(v.Value.ValueInt64())
				}
				k := v.Name.ValueString()
				service.Ulimits[k] = &ulimit
			}
		}
	}

	ptr := func(s string) *string {
		return &s
	}

	if !m.Environment.IsNull() && !m.Environment.IsUnknown() {
		if service.Environment == nil {
			service.Environment = map[string]*string{}
		}
		environment := map[string]string{}
		if diag := m.Environment.ElementsAs(context.Background(), &environment, true); !diag.HasError() {
			for k, v := range environment {
				service.Environment[k] = ptr(v)
			}
		}
	}

	if !m.Restart.IsNull() && !m.Restart.IsUnknown() {
		service.Restart = m.Restart.ValueString()
	}

	service.Name = sName
	replicas := m.Replicas.ValueInt64()
	intReplicas := int(replicas)
	service.Deploy = &composetypes.DeployConfig{
		Replicas: &intReplicas,
	}

	return service
}
