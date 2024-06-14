package models

import (
	"context"
	"fmt"

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

type Service struct {
	Name        types.String `tfsdk:"name"`
	Image       types.Set    `tfsdk:"image"`
	Replicas    types.Int64  `tfsdk:"replicas"`
	Logging     types.Set    `tfsdk:"logging"`
	Ports       types.Set    `tfsdk:"port"`
	Networks    types.Set    `tfsdk:"network"`
	NetworkMode types.String `tfsdk:"network_mode"`
	HealthCheck types.Set    `tfsdk:"health_check"`
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
					"interval": types.StringType,
					"timeout":  types.StringType,
					"retries":  types.Int64Type,
					"start":    types.StringType,
				},
			},
		},
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
	}
}

func (m Service) Value() attr.Value {

	var logging basetypes.SetValue
	var image basetypes.SetValue
	var ports basetypes.SetValue
	var networks basetypes.SetValue
	var health_check basetypes.SetValue

	if l, diag := m.Logging.ToSetValue(context.Background()); !diag.HasError() {
		logging = l
	}

	if i, diag := m.Image.ToSetValue(context.Background()); !diag.HasError() {
		image = i
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

	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":         types.StringValue(m.Name.ValueString()),
		"image":        image,
		"replicas":     types.Int64Value(m.Replicas.ValueInt64()),
		"port":         ports,
		"network":      networks,
		"logging":      logging,
		"network_mode": types.StringValue(m.NetworkMode.ValueString()),
		"health_check": health_check,
	})
}

func (m Service) AsComposeServiceConfig() composetypes.ServiceConfig {
	service := composetypes.ServiceConfig{}

	sName := m.Name.ValueString()

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

	service.Name = sName
	replicas := m.Replicas.ValueInt64()
	intReplicas := int(replicas)
	service.Deploy = &composetypes.DeployConfig{
		Replicas: &intReplicas,
	}

	return service
}
