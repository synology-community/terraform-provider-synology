package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/models/composetypes"
)

type Secret struct {
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
	File    types.String `tfsdk:"file"`
}

func (m Secret) AsComposeConfig(
	ctx context.Context,
	secret *composetypes.SecretConfig,
) (d diag.Diagnostics) {
	secret.Name = m.Name.ValueString()
	if !m.File.IsNull() && !m.File.IsUnknown() {
		secret.File = m.File.ValueString()
	} else {
		secret.File = m.Name.ValueString() // Default to name if file is not set
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
	// Pass the typed values through. Converting with ValueString() turned
	// null/unknown content into a known empty string, which then looked like
	// "upload this file" in handleSecrets and hit File Station for host-path
	// secrets (PLAT-511 / DSM 160).
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":    m.Name,
		"content": m.Content,
		"file":    m.File,
	})
}

func (m *Secret) FromComposeConfig(
	ctx context.Context,
	volume *composetypes.SecretConfig,
) (d diag.Diagnostics) {
	m.Name = types.StringValue(volume.Name)
	m.Content = types.StringValue(volume.Content)
	m.File = types.StringValue(volume.File)
	return
}
