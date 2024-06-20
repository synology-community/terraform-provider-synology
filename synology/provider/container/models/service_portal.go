package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServicePortal struct {
	Enable   types.Bool   `tfsdk:"enable"`
	Name     types.String `tfsdk:"name"`
	Port     types.Int64  `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
}

func (s *ServicePortal) First(ctx context.Context, m types.Set) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if !m.IsNull() && !m.IsUnknown() {
		elements := []ServicePortal{}
		diags = m.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			return diags
		}

		if len(elements) > 0 {
			*s = elements[0]
		}
	}

	return diags
}
