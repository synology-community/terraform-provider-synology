package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VDisk struct {
	// ID         types.String `tfsdk:"id"`
	Size types.Int64 `tfsdk:"size"`
	// Controller types.Int64  `tfsdk:"controller"`
	// Unmap      types.Bool   `tfsdk:"unmap"`
	ImageID   types.String `tfsdk:"image_id"`
	ImageName types.String `tfsdk:"image_name"`
}

func (m VDisk) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VDisk) AttrType() map[string]attr.Type {
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

func (m VDisk) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		// "id":          types.StringValue(m.ID.ValueString()),
		"size": types.Int64Value(m.Size.ValueInt64()),
		// "controller":  types.Int64Value(m.Controller.ValueInt64()),
		// "unmap":       types.BoolValue(m.Unmap.ValueBool()),
		"image_id":   types.StringValue(m.ImageID.ValueString()),
		"image_name": types.StringValue(m.ImageName.ValueString()),
	})
}
