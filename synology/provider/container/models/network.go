package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IPAMPool struct {
	Subnet     types.String `tfsdk:"subnet"`
	Gateway    types.String `tfsdk:"gateway"`
	IPRange    types.String `tfsdk:"ip_range"`
	AuxAddress types.Map    `tfsdk:"aux_address"`
}

type IPAMConfig struct {
	Driver  types.String `tfsdk:"driver"`
	Configs types.Set    `tfsdk:"config"`
}

type Network struct {
	Name       types.String `tfsdk:"name"`
	Driver     types.String `tfsdk:"driver"`
	DriverOpts types.Map    `tfsdk:"driver_opts"`
	Ipam       types.Set    `tfsdk:"ipam"`
	External   types.Bool   `tfsdk:"external"`
	Internal   types.Bool   `tfsdk:"internal"`
	Attachable types.Bool   `tfsdk:"attachable"`
	Labels     types.Map    `tfsdk:"labels"`
	EnableIPv6 types.Bool   `tfsdk:"enable_ipv6"`
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
		ipams := []IPAMConfig{}
		ipam := composetypes.IPAMConfig{}
		if diag := m.Ipam.ElementsAs(ctx, &ipams, true); !diag.HasError() {
			for _, i := range ipams {
				ipam.Driver = i.Driver.ValueString()

				ipamConfigs := []IPAMPool{}
				if diag := i.Configs.ElementsAs(ctx, &ipamConfigs, true); !diag.HasError() {
					for _, ipamCfg := range ipamConfigs {
						ipamConfig := composetypes.IPAMPool{
							Subnet:  ipamCfg.Subnet.ValueString(),
							Gateway: ipamCfg.Gateway.ValueString(),
							IPRange: ipamCfg.IPRange.ValueString(),
						}

						if !ipamCfg.AuxAddress.IsNull() && !ipamCfg.AuxAddress.IsUnknown() {
							var auxAddress map[string]string
							d.Append(ipamCfg.AuxAddress.ElementsAs(ctx, &auxAddress, true)...)
							if !d.HasError() {
								ipamConfig.AuxiliaryAddresses = auxAddress
							}
						}

						ipam.Config = append(ipam.Config, &ipamConfig)
					}
				} else {
					d = append(d, diag...)
				}

				network.Ipam = ipam
			}
		} else {
			d = append(d, diag...)
		}
	}

	return

}
