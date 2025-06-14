package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/models/composetypes"
)

type Config struct {
	Name     types.String `tfsdk:"name"`
	Content  types.String `tfsdk:"content"`
	File     types.String `tfsdk:"file"`
	External types.Bool   `tfsdk:"external"`
}

func (m Config) AsComposeConfig(
	ctx context.Context,
	config *composetypes.ConfigObjConfig,
) (d diag.Diagnostics) {
	config.Name = m.Name.ValueString()

	if !m.File.IsNull() && !m.File.IsUnknown() {
		config.File = m.File.ValueString()
	}

	if !m.External.IsNull() && !m.External.IsUnknown() {
		config.External = composetypes.External(m.External.ValueBool())
	}

	return
}

func (m *Config) FromComposeConfig(
	ctx context.Context,
	config *composetypes.ConfigObjConfig,
) (d diag.Diagnostics) {
	m.Name = types.StringValue(config.Name)
	m.File = types.StringValue(config.File)
	m.Content = types.StringValue(config.Content)
	if config.External {
		m.External = types.BoolValue(true)
	}
	return
}

func (m Config) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Config) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":     types.StringType,
		"content":  types.StringType,
		"file":     types.StringType,
		"external": types.BoolType,
	}
}

func (m Config) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":     m.Name,
		"content":  m.Content,
		"file":     m.File,
		"external": m.External,
	})
}
