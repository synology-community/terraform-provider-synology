package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Network struct {
	Name       types.String `tfsdk:"name"`
	Driver     types.String `tfsdk:"driver"`
	DriverOpts types.Map    `tfsdk:"driver_opts"`
	// Ipam       IPAMConfig `tfsdk:"ipam"`
	External   types.Bool `tfsdk:"external"`
	Internal   types.Bool `tfsdk:"internal"`
	Attachable types.Bool `tfsdk:"attachable"`
	Labels     types.Map  `tfsdk:"labels"`
	EnableIPv6 types.Bool `tfsdk:"enable_ipv6"`
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
		network.EnableIPv6 = m.EnableIPv6.ValueBool()
	}

	return

}
