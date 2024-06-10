package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VNic struct {
	ID  types.String `tfsdk:"id"`
	Mac types.String `tfsdk:"mac"`
	// Model  types.Int64  `tfsdk:"model"`
	Name types.String `tfsdk:"name"`
	// VNicID types.String `tfsdk:"vnic_id"`
}

func (m VNic) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VNic) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"mac":  types.StringType,
		// "model":   types.Int64Type,
		// "vnic_id": types.StringType,
	}
}

func (m VNic) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"id":   types.StringValue(m.ID.ValueString()),
		"name": types.StringValue(m.Name.ValueString()),
		"mac":  types.StringValue(m.Mac.ValueString()),
		// "model":   types.Int64Value(m.Model.ValueInt64()),
		// "vnic_id": types.StringValue(m.VNicID.ValueString()),
	})
}
