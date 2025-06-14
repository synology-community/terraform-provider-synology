package container

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	AttrName = "name"
)

// useRunningStatus implements the plan modifier.
type useRunningStatus struct{}

func UseRunningStatus() planmodifier.String {
	return useRunningStatus{}
}

// Description returns a human-readable description of the plan modifier.
func (m useRunningStatus) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useRunningStatus) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

func (m useRunningStatus) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	var plan, state ProjectResourceModel

	if req.State.Raw.IsNull() {
		return
	}

	// Check if the resource is being created.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Run.IsNull() || plan.Run.IsUnknown() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Status.ValueString() != "RUNNING" && plan.Run.ValueBool() {
		resp.Diagnostics.Append(req.Plan.SetAttribute(ctx, path.Root("status"), "STARTING")...)
	}
}

type useDefaultSharePath struct{}

// Description implements planmodifier.String.
func (u useDefaultSharePath) Description(context.Context) string {
	return "If the share path is not set, it will be set to /docker/<name>."
}

// MarkdownDescription implements planmodifier.String.
func (u useDefaultSharePath) MarkdownDescription(context.Context) string {
	return "If the share path is not set, it will be set to /docker/<name>."
}

// PlanModifyString implements planmodifier.String.
func (u useDefaultSharePath) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	if !req.PlanValue.IsUnknown() {
		return
	}

	// Set the share path to /docker/<name> if it is not set.
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(AttrName), &name)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(
			path.Root(AttrName),
			"failed to get name attribute during plan modification",
			"",
		)
		return
	}

	resp.PlanValue = types.StringValue(fmt.Sprintf("/docker/%s", name))
}

func UseDefaultSharePath() planmodifier.String {
	return useDefaultSharePath{}
}
