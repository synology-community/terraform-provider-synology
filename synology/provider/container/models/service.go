package models

import (
	"context"
	"fmt"
	"log"
	"math/big"
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
	Add  types.List `tfsdk:"add"`
	Drop types.List `tfsdk:"drop"`
}

func (m Capabilities) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Capabilities) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"add":  types.ListType{ElemType: types.StringType},
		"drop": types.ListType{ElemType: types.StringType},
	}
}

type Logging struct {
	Driver  types.String `tfsdk:"driver"`
	Options types.Map    `tfsdk:"options"`
}

func (m Logging) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Logging) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"driver":  types.StringType,
		"options": types.MapType{ElemType: types.StringType},
	}
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

func (m ServiceNetwork) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		// "id":          types.StringValue(m.ID.ValueString()),
		"name":           types.StringValue(m.Name.ValueString()),
		"aliases":        types.SetValueMust(types.StringType, []attr.Value{}),
		"ipv4_address":   types.StringValue(m.Ipv4Address.ValueString()),
		"ipv6_address":   types.StringValue(m.Ipv6Address.ValueString()),
		"link_local_ips": types.SetValueMust(types.StringType, []attr.Value{}),
		"mac_address":    types.StringValue(m.MacAddress.ValueString()),
		"driver_opts":    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		"priority":       types.Int64Value(m.Priority.ValueInt64()),
	})
}

func (m ServiceNetwork) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m ServiceNetwork) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":           types.StringType,
		"aliases":        types.SetType{ElemType: types.StringType},
		"ipv4_address":   types.StringType,
		"ipv6_address":   types.StringType,
		"link_local_ips": types.SetType{ElemType: types.StringType},
		"mac_address":    types.StringType,
		"driver_opts":    types.MapType{ElemType: types.StringType},
		"priority":       types.Int64Type,
	}
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

func (m Port) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Port) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":         types.StringType,
		"target":       types.Int64Type,
		"published":    types.StringType,
		"protocol":     types.StringType,
		"app_protocol": types.StringType,
		"mode":         types.StringType,
		"host_ip":      types.StringType,
	}
}

type HealthCheck struct {
	Test          types.List           `tfsdk:"test"`
	Interval      timetypes.GoDuration `tfsdk:"interval"`
	Timeout       timetypes.GoDuration `tfsdk:"timeout"`
	StartInterval timetypes.GoDuration `tfsdk:"start_interval"`
	StartPeriod   timetypes.GoDuration `tfsdk:"start_period"`
	Retries       types.Number         `tfsdk:"retries"`
}

func (m HealthCheck) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m HealthCheck) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"test":           types.ListType{ElemType: types.StringType},
		"interval":       timetypes.GoDurationType{},
		"timeout":        timetypes.GoDurationType{},
		"start_interval": timetypes.GoDurationType{},
		"start_period":   timetypes.GoDurationType{},
		"retries":        types.NumberType,
	}
}

type VolumeBind struct {
	Propagation    types.String `tfsdk:"propagation"`
	CreateHostPath types.Bool   `tfsdk:"create_host_path"`
	SELinux        types.String `tfsdk:"selinux"`
}

func (m VolumeBind) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VolumeBind) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"propagation":      types.StringType,
		"create_host_path": types.BoolType,
		"selinux":          types.StringType,
	}
}

type ServiceConfig struct {
	Source types.String `tfsdk:"source"`
	Target types.String `tfsdk:"target"`
	UID    types.String `tfsdk:"uid"`
	GID    types.String `tfsdk:"gid"`
	Mode   types.String `tfsdk:"mode"`
}

func (m ServiceConfig) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m ServiceConfig) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"source": types.StringType,
		"target": types.StringType,
		"uid":    types.StringType,
		"gid":    types.StringType,
		"mode":   types.StringType,
	}
}

type ServiceVolume struct {
	Source   types.String `tfsdk:"source"`
	Target   types.String `tfsdk:"target"`
	ReadOnly types.Bool   `tfsdk:"read_only"`
	Bind     types.Object `tfsdk:"bind"`
	Type     types.String `tfsdk:"type"`
}

func (m ServiceVolume) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m ServiceVolume) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"source":    types.StringType,
		"target":    types.StringType,
		"read_only": types.BoolType,
		"type":      types.StringType,
		"bind":      VolumeBind{}.ModelType(),
	}
}

type Ulimit struct {
	Value types.Int64 `tfsdk:"single"`
	Soft  types.Int64 `tfsdk:"soft"`
	Hard  types.Int64 `tfsdk:"hard"`
}

func (m Ulimit) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Ulimit) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"single": types.Int64Type,
		"soft":   types.Int64Type,
		"hard":   types.Int64Type,
	}
}

type ServiceDependency struct {
	Condition types.String `tfsdk:"condition"`
	Restart   types.Bool   `tfsdk:"restart"`
}

func (m ServiceDependency) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m ServiceDependency) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"condition": types.StringType,
		"restart":   types.BoolType,
	}
}

type Service struct {
	ContainerName types.String `tfsdk:"container_name"`
	Image         types.String `tfsdk:"image"`
	MemLimit      types.String `tfsdk:"mem_limit"`
	Entrypoint    types.List   `tfsdk:"entrypoint"`
	Command       types.List   `tfsdk:"command"`
	Replicas      types.Int64  `tfsdk:"replicas"`
	Logging       types.Object `tfsdk:"logging"`
	Ports         types.List   `tfsdk:"ports"`
	Networks      types.Map    `tfsdk:"networks"`
	NetworkMode   types.String `tfsdk:"network_mode"`
	HealthCheck   types.Object `tfsdk:"healthcheck"`
	SecurityOpt   types.List   `tfsdk:"security_opt"`
	Volumes       types.List   `tfsdk:"volumes"`
	Dependencies  types.Map    `tfsdk:"depends_on"`
	Privileged    types.Bool   `tfsdk:"privileged"`
	Tmpfs         types.List   `tfsdk:"tmpfs"`
	Ulimits       types.Map    `tfsdk:"ulimits"`
	Environment   types.Map    `tfsdk:"environment"`
	Restart       types.String `tfsdk:"restart"`
	Configs       types.List   `tfsdk:"configs"`
	Secrets       types.List   `tfsdk:"secrets"`
	Labels        types.Map    `tfsdk:"labels"`
	DNS           types.List   `tfsdk:"dns"`
	User          types.String `tfsdk:"user"`
	Capabilities  types.Object `tfsdk:"capabilities"`
	CapAdd        types.List   `tfsdk:"cap_add"`
	CapDrop       types.List   `tfsdk:"cap_drop"`
	Sysctls       types.Map    `tfsdk:"sysctls"`
	// Extensions    types.Map    `tfsdk:"extensions"`
}

// func (m Service) Value() attr.Value {
// 	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
// 		"name":           types.StringValue(m.Name.ValueString()),
// 		"image":          types.StringValue(m.Image.ValueString()),
// 		"container_name": types.StringValue(m.ContainerName.ValueString()),
// 		"configs":        types.ListValueMust(ServiceConfig{}.ModelType(), []attr.Value{}),
// 		"entrypoint":     types.ListValueMust(types.StringType, []attr.Value{}),
// 		"command":        types.ListValueMust(types.StringType, []attr.Value{}),
// 		"restart":        types.StringValue(m.Restart.ValueString()), // "no", "always", "on-failure", "unless-stopped", "always", "unless-stopped", "on-failure", "no
// 		"network_mode":   types.StringValue(m.NetworkMode.ValueString()),
// 		"replicas":       types.Int64Value(m.Replicas.ValueInt64()),
// 		"user":           types.StringValue(m.User.ValueString()),
// 		"ports":          types.ListValueMust(Port{}.ModelType(), []attr.Value{}),
// 		"mem_limit":      types.StringValue(m.MemLimit.ValueString()),
// 		// "extensions":     types.MapType{ElemType: types.StringType},
// 		"depends_on":   types.MapValueMust(ServiceDependency{}.ModelType(), map[string]attr.Value{}),
// 		"healthcheck":  types.ObjectValueMust(HealthCheck{}.AttrType(), map[string]attr.Value{}),
// 		"privileged":   types.BoolValue(m.Privileged.ValueBool()),
// 		"security_opt": types.ListValueMust(types.StringType, []attr.Value{}),
// 		"tmpfs":        types.ListValueMust(types.StringType, []attr.Value{}),
// 		"networks":     types.MapValueMust(ServiceNetwork{}.ModelType(), map[string]attr.Value{}),
// 		"labels":       types.MapValueMust(types.StringType, map[string]attr.Value{}),
// 		"secrets":      types.ListValueMust(ServiceConfig{}.ModelType(), []attr.Value{}),
// 		"ulimits":      types.MapValueMust(Ulimit{}.ModelType(), map[string]attr.Value{}),
// 		"logging":      types.ObjectValueMust(Logging{}.AttrType(), map[string]attr.Value{}),
// 		"volumes":      types.ListValueMust(ServiceVolume{}.ModelType(), []attr.Value{}),
// 		"environment":  types.MapValueMust(types.StringType, map[string]attr.Value{}),
// 		"dns":          types.ListValueMust(types.StringType, []attr.Value{}),
// 		"capabilities": types.ObjectValueMust(Capabilities{}.AttrType(), map[string]attr.Value{}),
// 	})
// }

func (m Service) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Service) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":           types.StringType,
		"image":          types.StringType,
		"container_name": types.StringType,
		"configs":        types.ListType{ElemType: ServiceConfig{}.ModelType()},
		"entrypoint":     types.ListType{ElemType: types.StringType},
		"command":        types.ListType{ElemType: types.StringType},
		"restart":        types.StringType, // "no", "always", "on-failure", "unless-stopped", "always", "unless-stopped", "on-failure", "no
		"network_mode":   types.StringType,
		"replicas":       types.Int64Type,
		"user":           types.StringType,
		"ports":          Port{}.ModelType(),
		"mem_limit":      types.StringType,
		// "extensions":     types.MapType{ElemType: types.StringType},
		"depends_on": types.MapType{
			ElemType: ServiceDependency{}.ModelType(),
		},
		"healthcheck":  HealthCheck{}.ModelType(),
		"privileged":   types.BoolType,
		"security_opt": types.ListType{ElemType: types.StringType},
		"tmpfs":        types.ListType{ElemType: types.StringType},
		"networks": types.MapType{
			ElemType: ServiceNetwork{}.ModelType(),
		},
		"labels": types.MapType{
			ElemType: types.StringType,
		},
		"secrets": types.ListType{
			ElemType: ServiceConfig{}.ModelType(),
		},
		"ulimits": types.MapType{
			ElemType: Ulimit{}.ModelType(),
		},
		"logging": Logging{}.ModelType(),
		"volumes": types.ListType{
			ElemType: ServiceVolume{}.ModelType(),
		},
		"environment":  types.MapType{ElemType: types.StringType},
		"dns":          types.ListType{ElemType: types.StringType},
		"capabilities": Capabilities{}.ModelType(),
		"cap_add":      types.ListType{ElemType: types.StringType},
		"cap_drop":     types.ListType{ElemType: types.StringType},
		"sysctls":      types.MapType{ElemType: types.StringType},
	}
}

func (m Service) Value() attr.Value {

	var logging basetypes.ObjectValue
	var entrypoints basetypes.ListValue
	var commands basetypes.ListValue
	var ports basetypes.ListValue
	var networks basetypes.MapValue
	var healthcheck basetypes.ObjectValue
	var dependencies basetypes.MapValue
	var volumes basetypes.ListValue
	var tmpfs basetypes.ListValue
	var ulimits basetypes.MapValue
	var environment basetypes.MapValue
	var configs basetypes.ListValue
	var secrets basetypes.ListValue
	var labels basetypes.MapValue
	var dns basetypes.ListValue
	var securityOpt basetypes.ListValue
	var capabilities basetypes.ObjectValue
	var sysctls basetypes.MapValue
	// var extensions basetypes.MapValue

	// if e, diag := m.Extensions.ToMapValue(context.Background()); !diag.HasError() {
	// 	extensions = e
	// }

	if s, diag := m.SecurityOpt.ToListValue(context.Background()); !diag.HasError() {
		securityOpt = s
	}

	if l, diag := m.Logging.ToObjectValue(context.Background()); !diag.HasError() {
		logging = l
	}

	if d, diag := m.Dependencies.ToMapValue(context.Background()); !diag.HasError() {
		dependencies = d
	}

	if c, diag := m.Command.ToListValue(context.Background()); !diag.HasError() {
		commands = c
	}

	if p, diag := m.Ports.ToListValue(context.Background()); !diag.HasError() {
		ports = p
	}

	if n, diag := m.Networks.ToMapValue(context.Background()); !diag.HasError() {
		networks = n
	}

	if hc, diag := m.HealthCheck.ToObjectValue(context.Background()); !diag.HasError() {
		healthcheck = hc
	}

	if v, diag := m.Volumes.ToListValue(context.Background()); !diag.HasError() {
		volumes = v
	}

	if t, diag := m.Tmpfs.ToListValue(context.Background()); !diag.HasError() {
		tmpfs = t
	}

	if u, diag := m.Ulimits.ToMapValue(context.Background()); !diag.HasError() {
		ulimits = u
	}

	if e, diag := m.Environment.ToMapValue(context.Background()); !diag.HasError() {
		environment = e
	}

	if l, diag := m.Labels.ToMapValue(context.Background()); !diag.HasError() {
		labels = l
	}

	if c, diag := m.Configs.ToListValue(context.Background()); !diag.HasError() {
		configs = c
	}

	if s, diag := m.Secrets.ToListValue(context.Background()); !diag.HasError() {
		secrets = s
	}

	if d, diag := m.DNS.ToListValue(context.Background()); !diag.HasError() {
		dns = d
	}

	if c, diag := m.Capabilities.ToObjectValue(context.Background()); !diag.HasError() {
		capabilities = c
	}

	if s, diag := m.Sysctls.ToMapValue(context.Background()); !diag.HasError() {
		sysctls = s
	}

	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"container_name": types.StringValue(m.ContainerName.ValueString()),
		"image":          types.StringValue(m.Image.ValueString()),
		"entrypoint":     entrypoints,
		"command":        commands,
		"mem_limit":      types.StringValue(m.MemLimit.ValueString()),
		"replicas":       types.Int64Value(m.Replicas.ValueInt64()),
		"ports":          ports,
		"network":        networks,
		"logging":        logging,
		"network_mode":   types.StringValue(m.NetworkMode.ValueString()),
		"healthcheck":    healthcheck,
		"security_opt":   securityOpt,
		"depends_on":     dependencies,
		"volume":         volumes,
		"privileged":     types.BoolValue(m.Privileged.ValueBool()),
		"tmpfs":          tmpfs,
		"ulimit":         ulimits,
		"environment":    environment,
		"restart":        types.StringValue(m.Restart.ValueString()),
		"configs":        configs,
		"secrets":        secrets,
		"labels":         labels,
		// "extensions":     extensions,
		"dns":          dns,
		"user":         types.StringValue(m.User.ValueString()),
		"capabilities": capabilities,
		"sysctls":      sysctls,
	})
}

func (m Service) AsComposeConfig(ctx context.Context, service *composetypes.ServiceConfig) (d diag.Diagnostics) {
	d = []diag.Diagnostic{}

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
		if diag := m.Logging.As(ctx, &logging, basetypes.ObjectAsOptions{}); !diag.HasError() {
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
					bind := VolumeBind{}
					if diag := v.Bind.As(ctx, &bind, basetypes.ObjectAsOptions{}); !diag.HasError() {
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
		dependencies := map[string]ServiceDependency{}
		if diag := m.Dependencies.ElementsAs(ctx, &dependencies, true); !diag.HasError() {
			service.DependsOn = map[string]composetypes.ServiceDependency{}
			for dk, d := range dependencies {
				dependency := composetypes.ServiceDependency{
					Condition: d.Condition.ValueString(),
					Restart:   d.Restart.ValueBool(),
				}
				service.DependsOn[dk] = dependency
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

	if !m.Secrets.IsNull() && !m.Secrets.IsUnknown() {
		secrets := []ServiceConfig{}
		if diag := m.Secrets.ElementsAs(ctx, &secrets, true); !diag.HasError() {
			service.Secrets = []composetypes.ServiceSecretConfig{}
			for _, c := range secrets {
				cfg := composetypes.ServiceSecretConfig{
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
				service.Secrets = append(service.Secrets, cfg)
			}
		} else {
			d = append(d, diag...)
		}
	}

	if !m.HealthCheck.IsNull() && !m.HealthCheck.IsUnknown() {
		service.HealthCheck = &composetypes.HealthCheckConfig{}
		hc := HealthCheck{}
		if diag := m.HealthCheck.As(ctx, &hc, basetypes.ObjectAsOptions{}); !diag.HasError() {
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
		service.Image = m.Image.ValueString()
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
		networks := map[string]ServiceNetwork{}
		if diag := m.Networks.ElementsAs(ctx, &networks, true); !diag.HasError() {
			service.Networks = map[string]*composetypes.ServiceNetworkConfig{}
			for k, n := range networks {
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

				service.Networks[k] = &network
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
		ulimits := map[string]Ulimit{}
		if diag := m.Ulimits.ElementsAs(ctx, &ulimits, true); !diag.HasError() {
			service.Ulimits = map[string]*composetypes.UlimitsConfig{}
			for k, v := range ulimits {
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

	if !m.CapAdd.IsNull() && !m.CapAdd.IsUnknown() {
		capAdd := []string{}
		if diag := m.CapAdd.ElementsAs(ctx, &capAdd, true); !diag.HasError() {
			service.CapAdd = capAdd
		} else {
			d = append(d, diag...)
		}
	}

	if !m.CapDrop.IsNull() && !m.CapDrop.IsUnknown() {
		capDrop := []string{}
		if diag := m.CapDrop.ElementsAs(ctx, &capDrop, true); !diag.HasError() {
			service.CapDrop = capDrop
		} else {
			d = append(d, diag...)
		}
	}

	if !m.Sysctls.IsNull() && !m.Sysctls.IsUnknown() {
		sysctls := map[string]string{}
		if diag := m.Sysctls.ElementsAs(ctx, &sysctls, true); !diag.HasError() {
			service.Sysctls = sysctls
		} else {
			d = append(d, diag...)
		}
	}

	service.ContainerName = m.ContainerName.ValueString()
	replicas := m.Replicas.ValueInt64()
	intReplicas := int(replicas)
	if m.Replicas.IsNull() || m.Replicas.IsUnknown() {
		intReplicas = 1
	}

	service.Deploy = &composetypes.DeployConfig{
		Replicas: &intReplicas,
	}
	service.User = m.User.ValueString()

	return d
}

func (m *Service) FromComposeConfig(ctx context.Context, service *composetypes.ServiceConfig) (d diag.Diagnostics) {
	d = []diag.Diagnostic{}

	m.ContainerName = types.StringValue(service.ContainerName)
	m.MemLimit = types.StringValue(fmt.Sprintf("%d", service.MemLimit))

	if service.Image != "" {
		m.Image = types.StringValue(service.Image)
	}

	if service.MemLimit != 0 {
		m.MemLimit = types.StringValue(fmt.Sprintf("%d", service.MemLimit))
	}

	if service.Logging != nil {
		logging := Logging{}
		logging.Driver = types.StringValue(service.Logging.Driver)
		opts := map[string]attr.Value{}
		for k, v := range service.Logging.Options {
			opts[k] = types.StringValue(v)
		}

		logging.Options = types.MapValueMust(types.StringType, opts)

		loggingValue, diags := types.ObjectValueFrom(ctx, Logging{}.AttrType(), logging)

		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Logging = loggingValue
		}
	}

	if len(service.DNS) > 0 {
		dns := []string{}
		for _, v := range service.DNS {
			dns = append(dns, v)
		}
		dnsValue, diags := types.ListValueFrom(ctx, types.StringType, dns)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.DNS = dnsValue
		}
	}

	if m.Ulimits.IsNull() || m.Ulimits.IsUnknown() {
		m.Ulimits = types.MapNull(Ulimit{}.ModelType())
	}

	if len(service.Ulimits) > 0 {
		ulimits := map[string]attr.Value{}
		for k, v := range service.Ulimits {
			ulimit := types.ObjectValueMust(Ulimit{}.AttrType(), map[string]attr.Value{
				"value": types.Int64Value(int64(v.Single)),
				"soft":  types.Int64Value(int64(v.Soft)),
				"hard":  types.Int64Value(int64(v.Hard)),
			})
			ulimits[k] = ulimit
		}

		ulimitsValues := types.MapValueMust(Ulimit{}.ModelType(), ulimits)

		m.Ulimits = ulimitsValues

		// ulimitsValue, diags := types.MapValueFrom(ctx, Ulimit{}.ModelType(), ulimits)
		// if diags.HasError() {
		// 	d = append(d, diags...)
		// } else {
		// 	m.Ulimits = ulimitsValue
		// }
	}

	if len(service.Volumes) > 0 {
		volumes := []ServiceVolume{}
		for _, v := range service.Volumes {
			volume := ServiceVolume{
				Source:   types.StringValue(v.Source),
				Target:   types.StringValue(v.Target),
				ReadOnly: types.BoolValue(v.ReadOnly),
				Type:     types.StringValue(v.Type),
			}
			if v.Bind != nil {
				bind := VolumeBind{
					Propagation:    types.StringValue(v.Bind.Propagation),
					CreateHostPath: types.BoolValue(v.Bind.CreateHostPath),
					SELinux:        types.StringValue(v.Bind.SELinux),
				}
				bindValue, diags := types.ObjectValueFrom(ctx, VolumeBind{}.AttrType(), bind)

				if diags.HasError() {
					d = append(d, diags...)
				} else {
					volume.Bind = bindValue
				}
			}
			volumes = append(volumes, volume)
		}

		volumesValue, diags := types.ListValueFrom(ctx, ServiceVolume{}.ModelType(), volumes)

		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Volumes = volumesValue
		}
	}

	// if len(service.Extensions) > 0 {
	// 	extensions := map[string]types.String{}
	// 	for k, v := range service.Extensions {
	// 		b, err := json.Marshal(v)
	// 		if err != nil {
	// 			log.Printf("error marshalling extension: %v", err)
	// 		} else {
	// 			extensions[k] = types.StringValue(string(b))
	// 		}
	// 	}
	// 	extensionsValue, diags := types.MapValueFrom(ctx, types.StringType, extensions)
	// 	if diags.HasError() {
	// 		d = append(d, diags...)
	// 	} else {
	// 		m.Extensions = extensionsValue
	// 	}
	// }

	if len(service.DependsOn) > 0 {
		dependencies := map[string]ServiceDependency{}
		for k, v := range service.DependsOn {
			dependency := ServiceDependency{
				Condition: types.StringValue(v.Condition),
				Restart:   types.BoolValue(v.Restart),
			}
			dependencies[k] = dependency
		}
		dependenciesValue, diags := types.MapValueFrom(ctx, ServiceDependency{}.ModelType(), dependencies)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Dependencies = dependenciesValue
		}
	}

	if len(service.Configs) > 0 {
		configs := []ServiceConfig{}
		for _, v := range service.Configs {
			config := ServiceConfig{
				Source: types.StringValue(v.Source),
				Target: types.StringValue(v.Target),
				UID:    types.StringValue(v.UID),
				GID:    types.StringValue(v.GID),
			}
			if v.Mode != nil {
				config.Mode = types.StringValue(fmt.Sprintf("%d", *v.Mode))
			}
			configs = append(configs, config)
		}

		configsValue, diags := types.ListValueFrom(ctx, ServiceConfig{}.ModelType(), configs)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Configs = configsValue
		}
	}

	if len(service.Secrets) > 0 {
		secrets := []ServiceConfig{}
		for _, v := range service.Secrets {
			secret := ServiceConfig{
				Source: types.StringValue(v.Source),
				Target: types.StringValue(v.Target),
				UID:    types.StringValue(v.UID),
				GID:    types.StringValue(v.GID),
			}
			if v.Mode != nil {
				secret.Mode = types.StringValue(fmt.Sprintf("%d", *v.Mode))
			}
			secrets = append(secrets, secret)
		}

		secretsValue, diags := types.ListValueFrom(ctx, Secret{}.ModelType(), secrets)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Secrets = secretsValue
		}
	}

	healthCheck := HealthCheck{
		Interval:      timetypes.NewGoDurationNull(),
		Timeout:       timetypes.NewGoDurationNull(),
		StartInterval: timetypes.NewGoDurationNull(),
		StartPeriod:   timetypes.NewGoDurationNull(),
		Retries:       types.NumberNull(),
	}
	if service.HealthCheck != nil {
		if service.HealthCheck.Timeout != nil {
			healthCheck.Timeout = timetypes.NewGoDurationValueFromStringMust(service.HealthCheck.Timeout.String())
		}

		if service.HealthCheck.Interval != nil {
			healthCheck.Interval = timetypes.NewGoDurationValueFromStringMust(service.HealthCheck.Interval.String())
		}

		if service.HealthCheck.StartInterval != nil {
			healthCheck.StartInterval = timetypes.NewGoDurationValueFromStringMust(service.HealthCheck.StartInterval.String())
		}

		if service.HealthCheck.StartPeriod != nil {
			healthCheck.StartPeriod = timetypes.NewGoDurationValueFromStringMust(service.HealthCheck.StartPeriod.String())
		}

		if service.HealthCheck.Retries != nil {
			healthCheck.Retries = types.NumberValue(big.NewFloat(float64(*service.HealthCheck.Retries)))
		}

		if len(service.HealthCheck.Test) > 0 {
			testValue, diags := types.ListValueFrom(ctx, types.StringType, service.HealthCheck.Test)
			if diags.HasError() {
				d = append(d, diags...)
			} else {
				healthCheck.Test = testValue
			}
		}
	}

	healthCheckValue, diags := types.ObjectValueFrom(ctx, HealthCheck{}.AttrType(), healthCheck)
	if diags.HasError() {
		d = append(d, diags...)
	} else {
		m.HealthCheck = healthCheckValue
	}
	m.HealthCheck = healthCheckValue

	if len(service.Entrypoint) > 0 {
		entrypoints, diags := types.ListValueFrom(ctx, types.StringType, service.Entrypoint)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Entrypoint = entrypoints
		}
	}

	if len(service.Command) > 0 {
		commands, diags := types.ListValueFrom(ctx, types.StringType, service.Command)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Command = commands
		}
	}

	if m.Command.IsNull() || m.Command.IsUnknown() {
		m.Command = types.ListNull(types.StringType)
	}

	if len(service.Ports) > 0 {
		ports := []Port{}
		for _, v := range service.Ports {
			port := Port{
				Name:        types.StringValue(v.Name),
				Target:      types.Int64Value(int64(v.Target)),
				Published:   types.StringValue(v.Published),
				Protocol:    types.StringValue(v.Protocol),
				AppProtocol: types.StringValue(v.AppProtocol),
				Mode:        types.StringValue(v.Mode),
				HostIP:      types.StringValue(v.HostIP),
			}
			ports = append(ports, port)
		}

		portsValue, diags := types.ListValueFrom(ctx, Port{}.ModelType(), ports)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Ports = portsValue
		}
	}

	if m.Ports.IsNull() || m.Ports.IsUnknown() {
		m.Ports = types.ListNull(Port{}.ModelType())
	}

	if service.NetworkMode != "" {
		m.NetworkMode = types.StringValue(service.NetworkMode)
	}

	if service.Networks != nil {
		networks := map[string]ServiceNetwork{}
		for k, v := range service.Networks {
			network := ServiceNetwork{
				Name:         types.StringValue(k),
				Aliases:      types.SetNull(types.StringType),
				Ipv4Address:  types.StringValue(v.Ipv4Address),
				Ipv6Address:  types.StringValue(v.Ipv6Address),
				LinkLocalIPs: types.SetNull(types.StringType),
				MacAddress:   types.StringValue(v.MacAddress),
				DriverOpts:   types.MapNull(types.StringType),
				Priority:     types.Int64Value(int64(v.Priority)),
			}

			if v.DriverOpts != nil {
				driverOpts := map[string]string{}
				for k, v := range v.DriverOpts {
					driverOpts[k] = v
				}
				driverOptsValue, diags := types.MapValueFrom(ctx, types.StringType, driverOpts)
				if diags.HasError() {
					d = append(d, diags...)
				} else {
					network.DriverOpts = driverOptsValue
				}
			}

			if v.LinkLocalIPs != nil {
				if len(v.LinkLocalIPs) > 0 {
					linkLocalIPsValue, diags := types.SetValueFrom(ctx, types.StringType, v.LinkLocalIPs)
					if diags.HasError() {
						d = append(d, diags...)
					} else {
						network.LinkLocalIPs = linkLocalIPsValue
					}
				}
			}

			if v.Aliases != nil {
				if len(v.Aliases) > 0 {
					aliasesValue, diags := types.SetValueFrom(ctx, types.StringType, v.Aliases)
					if diags.HasError() {
						d = append(d, diags...)
					} else {
						network.Aliases = aliasesValue
					}
				}
			}

			networks[k] = network
		}

		networksValue, diags := types.MapValueFrom(ctx, ServiceNetwork{}.ModelType(), networks)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Networks = networksValue
		}
	}

	m.Privileged = types.BoolValue(service.Privileged)

	if len(service.Tmpfs) > 0 {
		tmpfsValue, diags := types.ListValueFrom(ctx, types.StringType, service.Tmpfs)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Tmpfs = tmpfsValue
		}
	}

	if service.Ulimits != nil {
		ulimits := map[string]Ulimit{}
		for k, v := range service.Ulimits {
			ulimit := Ulimit{
				Value: types.Int64Value(int64(v.Single)),
				Soft:  types.Int64Value(int64(v.Soft)),
				Hard:  types.Int64Value(int64(v.Hard)),
			}
			ulimits[k] = ulimit
		}

		ulimitsValue, diags := types.MapValueFrom(ctx, Ulimit{}.ModelType(), ulimits)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Ulimits = ulimitsValue
		}
	}

	if service.Environment != nil {
		environment := map[string]types.String{}
		for k, v := range service.Environment {
			environment[k] = types.StringPointerValue(v)
		}

		environmentValue, diags := types.MapValueFrom(ctx, types.StringType, environment)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Environment = environmentValue
		}
	}

	if service.Labels != nil {
		labels := map[string]types.String{}
		for k, v := range service.Labels {
			labels[k] = types.StringValue(v)
		}

		labelsValue, diags := types.MapValueFrom(ctx, types.StringType, labels)
		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Labels = labelsValue
		}
	}

	if service.Image != "" {
		m.Image = types.StringValue(service.Image)
	}

	if service.Restart != "" {
		m.Restart = types.StringValue(service.Restart)
	}

	if service.CapAdd != nil || service.CapDrop != nil {
		capabilities := Capabilities{
			Add:  types.ListNull(types.StringType),
			Drop: types.ListNull(types.StringType),
		}

		if service.CapAdd != nil {
			if len(service.CapAdd) > 0 {
				capAddValues, diags := types.ListValueFrom(ctx, types.StringType, service.CapAdd)
				if diags.HasError() {
					d = append(d, diags...)
				} else {
					capabilities.Add = capAddValues
				}
			}
		}

		if service.CapDrop != nil {
			if len(service.CapDrop) > 0 {
				capDropValues, diags := types.ListValueFrom(ctx, types.StringType, service.CapDrop)
				if diags.HasError() {
					d = append(d, diags...)
				} else {
					capabilities.Drop = capDropValues
				}
			}
		}

		capabilitiesValue, diags := types.ObjectValueFrom(ctx, Capabilities{}.AttrType(), capabilities)

		if diags.HasError() {
			d = append(d, diags...)
		} else {
			m.Capabilities = capabilitiesValue
		}
	}

	if m.Capabilities.IsNull() || m.Capabilities.IsUnknown() {
		m.Capabilities = types.ObjectNull(Capabilities{}.AttrType())
	}

	if service.Sysctls != nil {
		sysctls := map[string]types.String{}
		for k, v := range service.Sysctls {
			sysctls[k] = types.StringValue(v)
		}
	}

	return d
}
