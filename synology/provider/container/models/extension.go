package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Extension struct {
	Name types.String `tfsdk:"name"`
}

func (e Extension) ModelType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: e.AttrType(),
	}
}

func (e Extension) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
	}
}
