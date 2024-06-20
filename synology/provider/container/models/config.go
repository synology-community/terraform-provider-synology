package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Config struct {
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
	File    types.String `tfsdk:"file"`
}

func (m Config) AsComposeConfig(ctx context.Context, config *composetypes.ConfigObjConfig) (d diag.Diagnostics) {
	config.Name = m.Name.ValueString()

	if !m.File.IsNull() && !m.File.IsUnknown() {
		config.File = m.File.ValueString()
	}

	return
}

func (m Config) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Config) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":    types.StringType,
		"content": types.StringType,
		"file":    types.StringType,
	}
}

func (m Config) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":    types.StringValue(m.Name.ValueString()),
		"content": types.StringValue(m.Content.ValueString()),
		"file":    types.StringValue(m.File.ValueString()),
	})
}
