package modifier

import (
	"context"

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
	if req.State.Raw.IsNull() {
		return
	}

	// Get the run attribute from the plan
	var planRun types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("run"), &planRun)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planRun.IsNull() || planRun.IsUnknown() {
		return
	}

	// Get the status attribute from the state
	var stateStatus types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("status"), &stateStatus)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if stateStatus.ValueString() != "RUNNING" && planRun.ValueBool() {
		resp.Diagnostics.Append(req.Plan.SetAttribute(ctx, path.Root("status"), "STARTING")...)
	}
}
