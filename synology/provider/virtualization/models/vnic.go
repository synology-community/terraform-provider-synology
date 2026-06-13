package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VNic struct {
	ID    types.String `tfsdk:"id"`
	Mac   types.String `tfsdk:"mac"`
	Model types.String `tfsdk:"model"`
	Name  types.String `tfsdk:"name"`
}

func (m VNic) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VNic) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":    types.StringType,
		"name":  types.StringType,
		"mac":   types.StringType,
		"model": types.StringType,
	}
}

func (m VNic) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"id":    types.StringValue(m.ID.ValueString()),
		"name":  types.StringValue(m.Name.ValueString()),
		"mac":   types.StringValue(m.Mac.ValueString()),
		"model": types.StringValue(m.Model.ValueString()),
	})
}
