package container

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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

func (m useRunningStatus) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
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
