package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Secret struct {
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
	File    types.String `tfsdk:"file"`
}

func (m Secret) AsComposeConfig(ctx context.Context, secret *composetypes.SecretConfig) (d diag.Diagnostics) {
	secret.Name = m.Name.ValueString()
	if !m.File.IsNull() && !m.File.IsUnknown() {
		secret.File = m.File.ValueString()
	}
	return
}

func (m Secret) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Secret) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":    types.StringType,
		"content": types.StringType,
		"file":    types.StringType,
	}
}

func (m Secret) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":    types.StringValue(m.Name.ValueString()),
		"content": types.StringValue(m.Content.ValueString()),
		"file":    types.StringValue(m.File.ValueString()),
	})
}

func (m *Secret) FromComposeConfig(ctx context.Context, volume *composetypes.SecretConfig) (d diag.Diagnostics) {
	m.Name = types.StringValue(volume.Name)
	m.Content = types.StringValue(volume.Content)
	m.File = types.StringValue(volume.File)
	return
}
