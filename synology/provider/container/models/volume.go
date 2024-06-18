package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Volume struct {
	Name       types.String `tfsdk:"name"`
	Driver     types.String `tfsdk:"driver"`
	DriverOpts types.Map    `tfsdk:"driver_opts"`
	External   types.Bool   `tfsdk:"external"`
	Labels     types.Map    `tfsdk:"labels"`
}

func (m Volume) AsComposeConfig(ctx context.Context, network *composetypes.VolumeConfig) (d diag.Diagnostics) {
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

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		var labels map[string]string
		d.Append(m.Labels.ElementsAs(ctx, &labels, true)...)
		if !d.HasError() {
			network.Labels = labels
		}
	}

	return

}
