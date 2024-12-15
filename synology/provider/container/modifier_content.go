package container

import (
	"context"

	"github.com/appkins/terraform-provider-synology/synology/provider/container/models"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UseArgumentsForUnknownContent returns a plan modifier that copies a known prior state
// value into the planned value. Use this when it is known that an unconfigured
// value will remain the same after a resource update.
//
// To prevent Terraform errors, the framework automatically sets unconfigured
// and Computed attributes to an unknown value "(known after apply)" on update.
// Using this plan modifier will instead display the prior state value in the
// plan, unless a prior plan modifier adjusts the value.
func UseArgumentsForUnknownContent() planmodifier.String {
	return useArgumentsForUnknownContent{}
}

// useArgumentsForUnknownContent implements the plan modifier.
type useArgumentsForUnknownContent struct{}

// Description returns a human-readable description of the plan modifier.
func (m useArgumentsForUnknownContent) Description(_ context.Context) string {
	return "Creates content from the Container Project Resource arguments."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useArgumentsForUnknownContent) MarkdownDescription(_ context.Context) string {
	return "Creates content from the Container Project Resource arguments."
}

// PlanModifyString implements the plan modification logic.
func (m useArgumentsForUnknownContent) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() {
		return
	}

	// Do nothing if there is no state value.
	var plan, state ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(req.Path, "failed to get plan value during plan modification", "")
		return
	}

	var yamlContent string

	resp.Diagnostics.Append(
		models.NewComposeContentBuilder(
			ctx,
		).SetServices(
			&plan.Services,
		).SetNetworks(
			&plan.Networks,
		).SetVolumes(
			&plan.Volumes,
		).SetConfigs(
			&plan.Configs,
		).SetSecrets(
			&plan.Secrets,
		).Build(
			&yamlContent,
		)...)

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(req.Path, "failed to build yaml content during plan modification", "")
		return
	}

	if req.State.Raw.IsKnown() && !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			resp.Diagnostics.AddAttributeError(req.Path, "failed to get state value during plan modification", "")
			return
		}

		if yamlContent != state.Content.ValueString() {
			resp.RequiresReplace = true
		}
	}

	resp.PlanValue = types.StringValue(yamlContent)
}
