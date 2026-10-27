package core

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology/pkg/api/core"
)

// stubTaskUpdateAPI implements core.Api by embedding it (nil) and overriding
// only TaskUpdate -- the one method TestUpdate_WritesEnableToState's call
// path reaches. Any other method call would nil-pointer-panic; that's
// intentional, since it would mean the test grew a dependency this stub
// doesn't account for.
type stubTaskUpdateAPI struct {
	core.Api
}

func (stubTaskUpdateAPI) TaskUpdate(
	_ context.Context,
	_ core.TaskRequest,
) (*core.TaskResult, error) {
	return &core.TaskResult{}, nil
}

// TestUpdate_WritesEnableToState is the PLAT-522 regression test.
//
// getTaskRequest mapped Enable correctly the whole time -- task_enable_test.go
// proves that and passed throughout the bug's life. The actual defect was
// downstream of the API call: Update wrote run/when/name/service onto
// resp.State when they changed, but never enable, so after an apply that
// only flipped enable, resp.State kept the *prior* enable value.
//
// That only breaks anything because of how the real framework server seeds
// UpdateResponse.State: internal/fwserver/server_updateresource.go sets
// `updateResp.State = *req.PriorState` (comment: "Require explicit provider
// updates for tracking successful updates") -- i.e. resp.State starts as the
// OLD state, not the plan. Any attribute the provider doesn't explicitly
// write back is left at its old value, and Terraform then rejects the apply
// because the reported new value doesn't match the plan. This test
// reproduces that exact seeding by hand; seeding resp.State from the plan
// instead would make the test pass against the unfixed code for the wrong
// reason.
func TestUpdate_WritesEnableToState(t *testing.T) {
	ctx := context.Background()

	p := &TaskResource{client: stubTaskUpdateAPI{}}

	var schemaResp resource.SchemaResponse
	p.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %s", schemaResp.Diagnostics)
	}

	prior := TaskResourceModel{
		ID:     types.Int64Value(1),
		Name:   types.StringValue("test"),
		User:   types.StringValue("terraform"),
		Enable: types.BoolValue(true),
		Run:    types.BoolValue(false),
		When:   types.StringValue("apply"),
	}
	planned := prior
	planned.Enable = types.BoolValue(false)

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("state.Set() diagnostics: %s", diags)
	}

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("plan.Set() diagnostics: %s", diags)
	}

	// Seeded from the PRIOR state, matching server_updateresource.go -- see
	// the doc comment above for why this, and not the plan, is correct here.
	respState := tfsdk.State{Schema: schemaResp.Schema}
	if diags := respState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("respState.Set() diagnostics: %s", diags)
	}

	req := resource.UpdateRequest{Plan: plan, State: state}
	resp := resource.UpdateResponse{State: respState}

	p.Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics: %s", resp.Diagnostics)
	}

	var got TaskResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("resp.State.Get() diagnostics: %s", diags)
	}

	if got.Enable.ValueBool() {
		t.Errorf(
			"resp.State enable = true, want false (the planned value); Update() must write "+
				"enable onto resp.State like it does run/when/name/service, or Terraform sees "+
				"an inconsistent result and fails the apply -- got %s",
			got.Enable,
		)
	}
}
