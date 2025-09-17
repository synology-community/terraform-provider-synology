package modifier

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

// SetSecretPathsFromContent returns a plan modifier that populates secret file paths
// based on their names when content is present.
func SetSecretPathsFromContent() planmodifier.Map {
	return setSecretPathsFromContent{}
}

// setSecretPathsFromContent implements the plan modifier.
type setSecretPathsFromContent struct{}

// Description returns a human-readable description of the plan modifier.
func (m setSecretPathsFromContent) Description(_ context.Context) string {
	return "Populates secret file paths based on names when content is present."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m setSecretPathsFromContent) MarkdownDescription(_ context.Context) string {
	return "Populates secret file paths based on names when content is present."
}

// PlanModifyMap implements the plan modification logic.
func (m setSecretPathsFromContent) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	var secretMap types.Map
	resp.Diagnostics.Append(populateSecretPathsInMap(ctx, req.Path, req.ConfigValue, &secretMap)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() {
		var planMap types.Map
		resp.Diagnostics.Append(populateSecretPathsInMap(ctx, req.Path, req.PlanValue, &planMap)...)
		if !resp.Diagnostics.HasError() {
			if reflect.DeepEqual(secretMap, planMap) {
				return
			}
		}
	}

	resp.PlanValue = secretMap
}

func populateSecretPathsInMap(
	ctx context.Context,
	reqPath path.Path,
	src types.Map,
	dst *types.Map,
) (diags diag.Diagnostics) {
	elements := map[string]models.Secret{}
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
					"Project Secrets Error",
					"Secrets must contain either file content or file path",
				)
			} else if !v.Content.IsUnknown() {
				v.File = types.StringValue(v.Name.ValueString())
			}
		}
		elementValues[k] = v.Value()
	}

	*dst = types.MapValueMust(models.Secret{}.ModelType(), elementValues)

	return diags
}
