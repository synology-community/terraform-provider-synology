package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
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
