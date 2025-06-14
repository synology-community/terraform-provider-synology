package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/models/composetypes"
)

type Volume struct {
	Name       types.String `tfsdk:"name"`
	Driver     types.String `tfsdk:"driver"`
	DriverOpts types.Map    `tfsdk:"driver_opts"`
	External   types.Bool   `tfsdk:"external"`
	Labels     types.Map    `tfsdk:"labels"`
}

func (m Volume) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Volume) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"driver":      types.StringType,
		"driver_opts": types.MapType{ElemType: types.StringType},
		"external":    types.BoolType,
		"labels":      types.MapType{ElemType: types.StringType},
	}
}

func (m Volume) AsComposeConfig(
	ctx context.Context,
	network *composetypes.VolumeConfig,
) (d diag.Diagnostics) {
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

func (m *Volume) FromComposeConfig(
	ctx context.Context,
	volume *composetypes.VolumeConfig,
) (d diag.Diagnostics) {
	m.Name = types.StringValue(volume.Name)
	m.Driver = types.StringValue(volume.Driver)
	m.External = types.BoolValue(bool(volume.External))
	m.DriverOpts = types.MapNull(types.StringType)
	m.Labels = types.MapNull(types.StringType)

	driverOpts := map[string]types.String{}
	for k, v := range volume.DriverOpts {
		driverOpts[k] = types.StringValue(v)
	}
	driverOptsValue, diags := types.MapValueFrom(ctx, types.StringType, driverOpts)
	if diags.HasError() {
		d.Append(diags...)
	} else {
		m.DriverOpts = driverOptsValue
	}

	labels := map[string]types.String{}
	for k, v := range volume.Labels {
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
