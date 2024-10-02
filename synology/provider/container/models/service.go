package models

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

type Capabilities struct {
	Add  types.Set `tfsdk:"add"`
	Drop types.Set `tfsdk:"drop"`
}

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

type ServiceConfig struct {
	Source types.String `tfsdk:"source"`
	Target types.String `tfsdk:"target"`
	UID    types.String `tfsdk:"uid"`
	GID    types.String `tfsdk:"gid"`
	Mode   types.String `tfsdk:"mode"`
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

type ServiceDependency struct {
	Name      types.String `tfsdk:"name"`
	Condition types.String `tfsdk:"condition"`
	Restart   types.Bool   `tfsdk:"restart"`
	Required  types.Bool   `tfsdk:"required"`
}

type Service struct {
	Name          types.String `tfsdk:"name"`
	ContainerName types.String `tfsdk:"container_name"`
	Image         types.Object `tfsdk:"image"`
	MemLimit      types.String `tfsdk:"mem_limit"`
	Entrypoint    types.List   `tfsdk:"entrypoint"`
	Command       types.List   `tfsdk:"command"`
	Replicas      types.Int64  `tfsdk:"replicas"`
	Logging       types.Set    `tfsdk:"logging"`
	Ports         types.Set    `tfsdk:"port"`
	Networks      types.Set    `tfsdk:"network"`
	NetworkMode   types.String `tfsdk:"network_mode"`
	HealthCheck   types.Set    `tfsdk:"health_check"`
	SecurityOpt   types.List   `tfsdk:"security_opt"`
	Volumes       types.Set    `tfsdk:"volume"`
	Dependencies  types.Set    `tfsdk:"depends_on"`
	Privileged    types.Bool   `tfsdk:"privileged"`
	Tmpfs         types.List   `tfsdk:"tmpfs"`
	Ulimits       types.Set    `tfsdk:"ulimit"`
	Environment   types.Map    `tfsdk:"environment"`
	Restart       types.String `tfsdk:"restart"`
	Configs       types.Set    `tfsdk:"config"`
	Labels        types.Map    `tfsdk:"labels"`
	DNS           types.List   `tfsdk:"dns"`
	User          types.String `tfsdk:"user"`
	Capabilities  types.Object `tfsdk:"capabilities"`
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
		"entrypoint":   types.StringType,
		"command":      types.ListType{ElemType: types.StringType},
		"restart":      types.StringType, // "no", "always", "on-failure", "unless-stopped", "always", "unless-stopped", "on-failure", "no
		"network_mode": types.StringType,
		"replicas":     types.Int64Type,
		"user":         types.StringType,
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
		"privileged":   types.BoolType,
		"security_opt": types.ListType{ElemType: types.StringType},
		"tmpfs":        types.ListType{ElemType: types.StringType},
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
		"dns":         types.ListType{ElemType: types.StringType},
		"capabilities": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"add":  types.SetType{ElemType: types.StringType},
				"drop": types.SetType{ElemType: types.StringType},
			},
		},
	}
}

func (m Service) Value() attr.Value {

	var logging basetypes.SetValue
	var image basetypes.ObjectValue
	var entrypoints basetypes.ListValue
	var commands basetypes.ListValue
	var ports basetypes.SetValue
	var networks basetypes.SetValue
	var health_check basetypes.SetValue
	var dependencies basetypes.SetValue
	var volumes basetypes.SetValue
	var tmpfs basetypes.ListValue
	var ulimits basetypes.SetValue
	var environment basetypes.MapValue
	var configs basetypes.SetValue
	var labels basetypes.MapValue
	var dns basetypes.ListValue
	var securityOpt basetypes.ListValue
	var capabilities basetypes.ObjectValue

	if s, diag := m.SecurityOpt.ToListValue(context.Background()); !diag.HasError() {
		securityOpt = s
	}

	if l, diag := m.Logging.ToSetValue(context.Background()); !diag.HasError() {
		logging = l
	}

	if i, diag := m.Image.ToObjectValue(context.Background()); !diag.HasError() {
		image = i
	}

	if d, diag := m.Dependencies.ToSetValue(context.Background()); !diag.HasError() {
		dependencies = d
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

	if l, diag := m.Labels.ToMapValue(context.Background()); !diag.HasError() {
		labels = l
	}

	if c, diag := m.Configs.ToSetValue(context.Background()); !diag.HasError() {
		configs = c
	}

	if d, diag := m.DNS.ToListValue(context.Background()); !diag.HasError() {
		dns = d
	}

	if c, diag := m.Capabilities.ToObjectValue(context.Background()); !diag.HasError() {
		capabilities = c
	}

	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":           types.StringValue(m.Name.ValueString()),
		"container_name": types.StringValue(m.ContainerName.ValueString()),
		"image":          image,
		"entrypoint":     entrypoints,
		"command":        commands,
		"replicas":       types.Int64Value(m.Replicas.ValueInt64()),
		"port":           ports,
		"network":        networks,
		"logging":        logging,
		"network_mode":   types.StringValue(m.NetworkMode.ValueString()),
		"health_check":   health_check,
		"security_opt":   securityOpt,
		"depends_on":     dependencies,
		"volume":         volumes,
		"privileged":     types.BoolValue(m.Privileged.ValueBool()),
		"tmpfs":          tmpfs,
		"ulimit":         ulimits,
		"environment":    environment,
		"restart":        types.StringValue(m.Restart.ValueString()),
		"config":         configs,
		"labels":         labels,
		"dns":            dns,
		"user":           types.StringValue(m.User.ValueString()),
		"capabilities":   capabilities,
	})
}

func (m Service) AsComposeConfig(ctx context.Context, service *composetypes.ServiceConfig) (d diag.Diagnostics) {
	d = []diag.Diagnostic{}

	sName := m.Name.ValueString()

	if !m.SecurityOpt.IsNull() && !m.SecurityOpt.IsUnknown() {
		securityOpts := []string{}
		if diag := m.SecurityOpt.ElementsAs(ctx, &securityOpts, true); !diag.HasError() {
			service.SecurityOpt = securityOpts
		} else {
			d = append(d, diag...)
		}
	}

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
		if diag := m.Logging.ElementsAs(ctx, &logging, true); !diag.HasError() {
			service.Logging.Driver = logging[0].Driver.ValueString()

			opts := map[string]string{}
			if diag := logging[0].Options.ElementsAs(ctx, &opts, true); !diag.HasError() {
				service.Logging.Options = opts
			}
		}
	}

	if !m.DNS.IsNull() && !m.DNS.IsUnknown() {
		dns := []string{}
		if diag := m.DNS.ElementsAs(ctx, &dns, true); !diag.HasError() {
			service.DNS = dns
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Volumes.IsNull() && !m.Volumes.IsUnknown() {
		volumes := []ServiceVolume{}
		if diag := m.Volumes.ElementsAs(ctx, &volumes, true); !diag.HasError() {
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
					if diag := v.Bind.ElementsAs(ctx, &binds, true); !diag.HasError() {
						bind := binds[0]
						volume.Bind = &composetypes.ServiceVolumeBind{
							Propagation:    bind.Propagation.ValueString(),
							CreateHostPath: bind.CreateHostPath.ValueBool(),
							SELinux:        bind.SELinux.ValueString(),
						}
					} else {
						d = append(d, diag...)
					}
				}
				service.Volumes = append(service.Volumes, volume)
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Dependencies.IsNull() && !m.Dependencies.IsUnknown() {
		dependencies := []ServiceDependency{}
		if diag := m.Dependencies.ElementsAs(ctx, &dependencies, true); !diag.HasError() {
			service.DependsOn = map[string]composetypes.ServiceDependency{}
			for _, d := range dependencies {
				dependency := composetypes.ServiceDependency{
					Condition: d.Condition.ValueString(),
					Restart:   d.Restart.ValueBool(),
					Required:  d.Required.ValueBool(),
				}
				service.DependsOn[d.Name.ValueString()] = dependency
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Configs.IsNull() && !m.Configs.IsUnknown() {
		configs := []ServiceConfig{}
		if diag := m.Configs.ElementsAs(ctx, &configs, true); !diag.HasError() {
			service.Configs = []composetypes.ServiceConfigObjConfig{}
			for _, c := range configs {
				cfg := composetypes.ServiceConfigObjConfig{
					Source: c.Source.ValueString(),
					Target: c.Target.ValueString(),
					UID:    c.UID.ValueString(),
					GID:    c.GID.ValueString(),
				}
				if !c.Mode.IsNull() && !c.Mode.IsUnknown() {
					mode, err := strconv.ParseUint(c.Mode.ValueString(), 10, 32)

					if err != nil {
						log.Printf("error parsing mode: %v", err)
					} else {
						mode32 := uint32(mode)
						cfg.Mode = &mode32
					}
				}
				service.Configs = append(service.Configs, cfg)
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.HealthCheck.IsNull() && !m.HealthCheck.IsUnknown() && len(m.HealthCheck.Elements()) == 1 {
		service.HealthCheck = &composetypes.HealthCheckConfig{}
		healthCheck := []HealthCheck{}
		if diag := m.HealthCheck.ElementsAs(ctx, &healthCheck, true); !diag.HasError() {
			hc := healthCheck[0]
			if !hc.Test.IsNull() || !hc.Test.IsUnknown() {
				test := []string{}
				if diag := hc.Test.ElementsAs(ctx, &test, true); !diag.HasError() {
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
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Image.IsNull() && !m.Image.IsUnknown() {
		image := Image{}
		if diag := m.Image.As(ctx, &image, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		}); !diag.HasError() {
			iName := image.Name.ValueString()
			var iTag, iRepo string
			if image.Repository.IsNull() || image.Repository.IsUnknown() {
				iRepo = "docker.io"
			} else {
				iRepo = image.Repository.ValueString()
			}
			if image.Tag.IsNull() || image.Tag.IsUnknown() {
				iTag = "latest"
			} else {
				iTag = image.Tag.ValueString()
			}
			service.Image = fmt.Sprintf("%s/%s:%s", iRepo, iName, iTag)
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Entrypoint.IsNull() && !m.Entrypoint.IsUnknown() {
		entrypoints := []string{}
		if diag := m.Entrypoint.ElementsAs(ctx, &entrypoints, true); !diag.HasError() {
			service.Entrypoint = entrypoints
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Command.IsNull() && !m.Command.IsUnknown() {
		commands := []string{}
		if diag := m.Command.ElementsAs(ctx, &commands, true); !diag.HasError() {
			service.Command = commands
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Ports.IsNull() && !m.Ports.IsUnknown() {
		ports := []Port{}
		if diag := m.Ports.ElementsAs(ctx, &ports, true); !diag.HasError() {
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
		} else {
			d = append(d, diag...)
		}
	}

	if !m.NetworkMode.IsNull() && !m.NetworkMode.IsUnknown() {
		service.NetworkMode = m.NetworkMode.ValueString()
	}

	if !m.Networks.IsNull() && !m.Networks.IsUnknown() {
		networks := []ServiceNetwork{}
		if diag := m.Networks.ElementsAs(ctx, &networks, true); !diag.HasError() {
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
					if diag := n.Aliases.ElementsAs(ctx, &aliases, true); !diag.HasError() {
						network.Aliases = aliases
					}
				}

				if n.LinkLocalIPs.IsNull() || n.LinkLocalIPs.IsUnknown() {
					linkLocalIPs := []string{}
					if diag := n.LinkLocalIPs.ElementsAs(ctx, &linkLocalIPs, true); !diag.HasError() {
						network.LinkLocalIPs = linkLocalIPs
					} else {
						d = append(d, diag...)
					}
				}

				if !n.DriverOpts.IsNull() && !n.DriverOpts.IsUnknown() {
					driverOpts := map[string]string{}
					if diag := n.DriverOpts.ElementsAs(ctx, &driverOpts, true); !diag.HasError() {
						network.DriverOpts = driverOpts
					} else {
						d = append(d, diag...)
					}
				}

				service.Networks[n.Name.ValueString()] = &network
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Privileged.IsNull() && !m.Privileged.IsUnknown() {
		service.Privileged = m.Privileged.ValueBool()
	}

	if !m.Tmpfs.IsNull() && !m.Tmpfs.IsUnknown() {
		tmpfs := []string{}
		if diag := m.Tmpfs.ElementsAs(ctx, &tmpfs, true); !diag.HasError() {
			service.Tmpfs = tmpfs
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Ulimits.IsNull() && !m.Ulimits.IsUnknown() {
		ulimits := []Ulimit{}
		if diag := m.Ulimits.ElementsAs(ctx, &ulimits, true); !diag.HasError() {
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
		} else {
			d = append(d, diag...)
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
		if diag := m.Environment.ElementsAs(ctx, &environment, true); !diag.HasError() {
			for k, v := range environment {
				service.Environment[k] = ptr(v)
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}
		labels := map[string]string{}
		if diag := m.Labels.ElementsAs(ctx, &labels, true); !diag.HasError() {
			for k, v := range labels {
				service.Labels[k] = v
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Restart.IsNull() && !m.Restart.IsUnknown() {
		service.Restart = m.Restart.ValueString()
	}

	if !m.Capabilities.IsNull() && !m.Capabilities.IsUnknown() {
		capabilities := Capabilities{}
		if diag := m.Capabilities.As(ctx, &capabilities, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		}); !diag.HasError() {
			service.CapAdd = []string{}
			service.CapDrop = []string{}
			if !capabilities.Add.IsNull() && !capabilities.Add.IsUnknown() {
				add := []string{}
				if diag := capabilities.Add.ElementsAs(ctx, &add, true); !diag.HasError() {
					service.CapAdd = add
				} else {
					d = append(d, diag...)
				}
			}
			if !capabilities.Drop.IsNull() && !capabilities.Drop.IsUnknown() {
				drop := []string{}
				if diag := capabilities.Drop.ElementsAs(ctx, &drop, true); !diag.HasError() {
					service.CapDrop = drop
				} else {
					d = append(d, diag...)
				}
			}
		} else {
			d = append(d, diag...)
		}
	}

	service.ContainerName = m.ContainerName.ValueString()
	service.Name = sName
	replicas := m.Replicas.ValueInt64()
	intReplicas := int(replicas)
	service.Deploy = &composetypes.DeployConfig{
		Replicas: &intReplicas,
	}
	service.User = m.User.ValueString()

	return d
}
