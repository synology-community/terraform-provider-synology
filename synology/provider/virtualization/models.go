package virtualization

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VDiskModel struct {
	// ID         types.String `tfsdk:"id"`
	Size       types.Int64 `tfsdk:"size"`
	CreateType types.Int64 `tfsdk:"create_type"`
	// Controller types.Int64  `tfsdk:"controller"`
	// Unmap      types.Bool   `tfsdk:"unmap"`
	ImageID   types.String `tfsdk:"image_id"`
	ImageName types.String `tfsdk:"image_name"`
}

func (m VDiskModel) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VDiskModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		// "id":          types.StringType,
		"size": types.Int64Type,
		// "controller":  types.Int64Type,
		// "unmap":       types.BoolType,
		"image_id":    types.StringType,
		"image_name":  types.StringType,
		"create_type": types.Int64Type,
	}
}

func (m VDiskModel) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		// "id":          types.StringValue(m.ID.ValueString()),
		"size": types.Int64Value(m.Size.ValueInt64()),
		// "controller":  types.Int64Value(m.Controller.ValueInt64()),
		// "unmap":       types.BoolValue(m.Unmap.ValueBool()),
		"image_id":    types.StringValue(m.ImageID.ValueString()),
		"image_name":  types.StringValue(m.ImageName.ValueString()),
		"create_type": types.Int64Value(m.CreateType.ValueInt64()),
	})
}

type VNicModel struct {
	ID  types.String `tfsdk:"id"`
	Mac types.String `tfsdk:"mac"`
	// Model  types.Int64  `tfsdk:"model"`
	Name types.String `tfsdk:"name"`
	// VNicID types.String `tfsdk:"vnic_id"`
}

func (m VNicModel) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VNicModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"mac":  types.StringType,
		// "model":   types.Int64Type,
		// "vnic_id": types.StringType,
	}
}

func (m VNicModel) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"id":   types.StringValue(m.ID.ValueString()),
		"name": types.StringValue(m.Name.ValueString()),
		"mac":  types.StringValue(m.Mac.ValueString()),
		// "model":   types.Int64Value(m.Model.ValueInt64()),
		// "vnic_id": types.StringValue(m.VNicID.ValueString()),
	})
}
