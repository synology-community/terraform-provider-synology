package container

import (
	"context"
	"fmt"

	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SetConfigPathsFromContent returns a plan modifier that copies a known prior state
// value into the planned value. Use this when it is known that an unconfigured
// value will remain the same after a resource update.
//
// To prevent Terraform errors, the framework automatically sets unconfigured
// and Computed attributes to an unknown value "(known after apply)" on update.
// Using this plan modifier will instead display the prior state value in the
// plan, unless a prior plan modifier adjusts the value.
func SetConfigPathsFromContent() planmodifier.Map {
	return setConfigPathsFromContent{}
}

// setConfigPathsFromContent implements the plan modifier.
type setConfigPathsFromContent struct{}

// Description returns a human-readable description of the plan modifier.
func (m setConfigPathsFromContent) Description(_ context.Context) string {
	return "Creates content from the Container Project Resource arguments."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m setConfigPathsFromContent) MarkdownDescription(_ context.Context) string {
	return "Creates content from the Container Project Resource arguments."
}

// PlanModifyString implements the plan modification logic.
func (m setConfigPathsFromContent) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	elements := map[string]models.Config{}
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &elements, true)...)

	if resp.Diagnostics.HasError() {
		return
	}

	elementValues := map[string]attr.Value{}

	for k, v := range elements {
		if v.File.IsNull() || v.File.IsUnknown() {

			if v.Content.IsNull() || v.Content.IsUnknown() {
				resp.Diagnostics.AddAttributeError(path.Root("configs"), "Project Configs Error", "Configs must contain either file content or file path")
				return
			}

			v.File = types.StringValue(fmt.Sprintf("config_%s", v.Name.ValueString()))
		}

		elementValues[k] = v.Value()
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.PlanValue = types.MapValueMust(models.Config{}.ModelType(), elementValues)
}
