package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type IPAMPool struct {
	Subnet     types.String `tfsdk:"subnet"`
	Gateway    types.String `tfsdk:"gateway"`
	IPRange    types.String `tfsdk:"ip_range"`
	AuxAddress types.Map    `tfsdk:"aux_addresses"`
}

func (m IPAMPool) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m IPAMPool) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"subnet":      types.StringType,
		"gateway":     types.StringType,
		"ip_range":    types.StringType,
		"aux_address": types.MapType{ElemType: types.StringType},
	}
}

type IPAMConfig struct {
	Driver  types.String `tfsdk:"driver"`
	Configs types.List   `tfsdk:"config"`
}

func (m IPAMConfig) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m IPAMConfig) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"driver": types.StringType,
		"config": types.ListType{ElemType: IPAMPool{}.ModelType()},
	}
}

type Network struct {
	Name       types.String `tfsdk:"name"`
	Driver     types.String `tfsdk:"driver"`
	DriverOpts types.Map    `tfsdk:"driver_opts"`
	Ipam       types.Object `tfsdk:"ipam"`
	External   types.Bool   `tfsdk:"external"`
	Internal   types.Bool   `tfsdk:"internal"`
	Attachable types.Bool   `tfsdk:"attachable"`
	Labels     types.Map    `tfsdk:"labels"`
	EnableIPv6 types.Bool   `tfsdk:"enable_ipv6"`
}

func (m Network) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Network) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"driver":      types.StringType,
		"driver_opts": types.MapType{ElemType: types.StringType},
		"ipam":        types.ObjectType{AttrTypes: IPAMConfig{}.AttrType()},
		"external":    types.BoolType,
		"internal":    types.BoolType,
		"attachable":  types.BoolType,
		"labels":      types.MapType{ElemType: types.StringType},
		"enable_ipv6": types.BoolType,
	}
}

func (m Network) AsComposeConfig(ctx context.Context, network *composetypes.NetworkConfig) (d diag.Diagnostics) {
	network.Name = m.Name.ValueString()

	if !m.DriverOpts.IsNull() && !m.DriverOpts.IsUnknown() {
		var driverOpts map[string]string
		d.Append(m.DriverOpts.ElementsAs(ctx, &driverOpts, true)...)
		if !d.HasError() {
			network.DriverOpts = driverOpts
		}
	}

	if !m.Driver.IsNull() && !m.Driver.IsUnknown() {
		network.Driver = m.Driver.ValueString()
	}

	if !m.External.IsNull() && !m.External.IsUnknown() {
		network.External = composetypes.External(m.External.ValueBool())
	}

	if !m.Internal.IsNull() && !m.Internal.IsUnknown() {
		network.Internal = m.Internal.ValueBool()
	}

	if !m.Attachable.IsNull() && !m.Attachable.IsUnknown() {
		network.Attachable = m.Attachable.ValueBool()
	}

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		var labels map[string]string
		d.Append(m.Labels.ElementsAs(ctx, &labels, true)...)
		if !d.HasError() {
			network.Labels = labels
		}
	}

	if !m.EnableIPv6.IsNull() && !m.EnableIPv6.IsUnknown() {
		network.EnableIPv6 = m.EnableIPv6.ValueBoolPointer()
	}

	if !m.Ipam.IsNull() && !m.Ipam.IsUnknown() {
		ipam := IPAMConfig{}
		if diag := m.Ipam.As(ctx, &ipam, basetypes.ObjectAsOptions{}); !diag.HasError() {
			network.Ipam = composetypes.IPAMConfig{
				Driver: ipam.Driver.ValueString(),
			}

			ipamPools := []IPAMPool{}
			if diag := ipam.Configs.ElementsAs(ctx, &ipamPools, true); !diag.HasError() {
				for _, ipamCfg := range ipamPools {
					ipamPool := composetypes.IPAMPool{
						Subnet:  ipamCfg.Subnet.ValueString(),
						Gateway: ipamCfg.Gateway.ValueString(),
						IPRange: ipamCfg.IPRange.ValueString(),
					}

					if !ipamCfg.AuxAddress.IsNull() && !ipamCfg.AuxAddress.IsUnknown() {
						var auxAddress map[string]string
						d.Append(ipamCfg.AuxAddress.ElementsAs(ctx, &auxAddress, true)...)
						if !d.HasError() {
							ipamPool.AuxiliaryAddresses = auxAddress
						}
					}

					network.Ipam.Config = append(network.Ipam.Config, &ipamPool)
				}
			} else {
				d = append(d, diag...)
			}
		} else {
			d = append(d, diag...)
		}
	}

	return

}

func (m *Network) FromComposeConfig(ctx context.Context, network *composetypes.NetworkConfig) (d diag.Diagnostics) {
	m.Name = types.StringValue(network.Name)
	m.Driver = types.StringValue(network.Driver)
	m.External = types.BoolValue(bool(network.External))
	m.DriverOpts = types.MapNull(types.StringType)
	m.Labels = types.MapNull(types.StringType)
	m.Attachable = types.BoolValue(network.Attachable)
	m.Internal = types.BoolValue(network.Internal)
	m.EnableIPv6 = types.BoolPointerValue(network.EnableIPv6)

	driverOpts := map[string]types.String{}
	for k, v := range network.DriverOpts {
		driverOpts[k] = types.StringValue(v)
	}
	driverOptsValue, diags := types.MapValueFrom(ctx, types.StringType, driverOpts)
	if diags.HasError() {
		d.Append(diags...)
	} else {
		m.DriverOpts = driverOptsValue
	}

	labels := map[string]types.String{}
	for k, v := range network.Labels {
		labels[k] = types.StringValue(v)
	}
	labelsValue, diags := types.MapValueFrom(ctx, types.StringType, labels)
	if diags.HasError() {
		d.Append(diags...)
	} else {
		m.Labels = labelsValue
	}
	return
}
