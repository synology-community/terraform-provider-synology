package container

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"
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
func (m setConfigPathsFromContent) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	var configMap types.Map
	resp.Diagnostics.Append(populateConfigPathsInMap(ctx, req.Path, req.ConfigValue, &configMap)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() {
		var planMap types.Map
		resp.Diagnostics.Append(populateConfigPathsInMap(ctx, req.Path, req.PlanValue, &planMap)...)
		if !resp.Diagnostics.HasError() {
			if reflect.DeepEqual(configMap, planMap) {
				return
			}
		}
	}

	resp.PlanValue = configMap
}

func populateConfigPathsInMap(
	ctx context.Context,
	reqPath path.Path,
	src types.Map,
	dst *types.Map,
) (diags diag.Diagnostics) {
	elements := map[string]models.Config{}
	diags.Append(src.ElementsAs(ctx, &elements, true)...)
	if diags.HasError() {
		return
	}

	elementValues := map[string]attr.Value{}
	for k, v := range elements {
		if v.File.IsNull() {
			if v.Content.IsNull() {
				diags.AddAttributeError(
					reqPath,
					"Project Configs Error",
					"Configs must contain either file content or file path",
				)
			} else if !v.Content.IsUnknown() {
				v.File = types.StringValue(v.Name.ValueString())
			}
		}
		elementValues[k] = v.Value()
	}

	*dst = types.MapValueMust(models.Config{}.ModelType(), elementValues)

	return diags
}
