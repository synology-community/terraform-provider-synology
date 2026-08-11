package modifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"
)

// UseSchemaForUnknownContent returns a plan modifier that sets the Container
// Project Resource content from the configured arguments when content is not
// already known from config or state.
func UseSchemaForUnknownContent() planmodifier.String {
	return useArgumentsForUnknownContent{}
}

// useArgumentsForUnknownContent implements the plan modifier.
type useArgumentsForUnknownContent struct{}

// Description returns a human-readable description of the plan modifier.
func (m useArgumentsForUnknownContent) Description(_ context.Context) string {
	return "Creates content from the Container Project Resource arguments when content is unset."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useArgumentsForUnknownContent) MarkdownDescription(_ context.Context) string {
	return "Creates content from the Container Project Resource arguments when content is unset."
}

// PlanModifyString implements the plan modification logic.
func (m useArgumentsForUnknownContent) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Explicit content in config wins (framework already copied it into plan).
	if !req.ConfigValue.IsNull() && req.ConfigValue.ValueString() != "" {
		return
	}

	// Import / freeze path: content only in state (config omitted, often with
	// lifecycle.ignore_changes). Keep the prior value so we do not rewrite
	// live compose to an empty "services: {}" document (PLAT-552).
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() &&
		req.StateValue.ValueString() != "" {
		resp.PlanValue = req.StateValue
		return
	}

	// Get the current plan value - Should run after config modification
	var config models.ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"failed to get plan value during plan modification",
			"",
		)
		return
	}

	// Convert the config to yaml content
	var yamlContent string
	resp.Diagnostics.Append(config.ConfigRaw(ctx, &yamlContent)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"failed to build yaml content during plan modification",
			"",
		)
		return
	}

	// Set the plan value to the yaml content
	resp.PlanValue = types.StringValue(yamlContent)
}
