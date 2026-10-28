package core

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/synology-community/go-synology/pkg/api/core"
)

// sharePermissionResourceSchemaForModifyPlanTest returns the resource's real
// schema.Schema (not a hand-rolled stand-in), so the tftypes.Value built
// below is exactly what Terraform would send over the wire.
func sharePermissionResourceSchemaForModifyPlanTest(t *testing.T) rschema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&SharePermissionResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", resp.Diagnostics)
	}
	return resp.Schema
}

// permissionSetFromEntries builds a types.Set of types.Object -- constructed
// directly with types.ObjectValue against the resource's own
// sharePermissionEntryAttrTypes(), never through reflection over a Go
// struct -- so it matches exactly what real config/state decodes to on the
// wire. entries not listing a flag leave it null, the same as an unconfigured
// attribute in real HCL.
type permEntry struct {
	name                                    string
	readonly, writable, deny, custom, admin types.Bool
}

func permissionSetFromEntries(t *testing.T, entries []permEntry) types.Set {
	t.Helper()
	objs := make([]attr.Value, 0, len(entries))
	for _, e := range entries {
		obj, diags := types.ObjectValue(sharePermissionEntryAttrTypes(), map[string]attr.Value{
			"name":        types.StringValue(e.name),
			"is_readonly": e.readonly,
			"is_writable": e.writable,
			"is_deny":     e.deny,
			"is_custom":   e.custom,
			"is_admin":    e.admin,
		})
		if diags.HasError() {
			t.Fatalf("ObjectValue(%s) diagnostics: %s", e.name, diags)
		}
		objs = append(objs, obj)
	}
	set, diags := types.SetValue(types.ObjectType{AttrTypes: sharePermissionEntryAttrTypes()}, objs)
	if diags.HasError() {
		t.Fatalf("SetValue diagnostics: %s", diags)
	}
	return set
}

// TestModifyPlan_PreservesConfiguredTrueFlag is the PLAT-705 regression
// test. It reproduces the real end-to-end defect: a practitioner declares
// `is_writable = true` for an account that already has that grant in state
// (mirroring a `tofu import` followed by `tofu plan`), while Terraform
// Core's own proposed plan -- which this test stands in for by deliberately
// planning `is_writable = false`, the exact wrong value observed against a
// real NAS -- cannot be trusted for Set-nested-block Computed attributes.
// ModifyPlan must discard that proposal and rebuild the permission set from
// req.Config, so the final plan -- and everything planEntries() derives
// from it -- carries the practitioner's true value through untouched.
//
// A second row ("guest") pins the companion case from the bug report: an
// all-null (unconfigured) row must still resolve every flag to its false
// default, exactly as it did before this fix, so the correction does not
// trade one class of dropped value for another.
func TestModifyPlan_PreservesConfiguredTrueFlag(t *testing.T) {
	ctx := context.Background()
	sch := sharePermissionResourceSchemaForModifyPlanTest(t)

	// Config: paatwood declares is_writable = true and leaves every other
	// flag unconfigured (null); guest declares nothing at all (every flag
	// null).
	configPermission := permissionSetFromEntries(t, []permEntry{
		{name: "paatwood", writable: types.BoolValue(true)},
		{name: "guest"},
	})
	configModel := SharePermissionResourceModel{
		Share:         types.StringValue("photo"),
		UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
		Permission:    configPermission,
	}
	var configPlanForRaw tfsdk.Plan
	configPlanForRaw.Schema = sch
	if diags := configPlanForRaw.Set(ctx, &configModel); diags.HasError() {
		t.Fatalf("building config raw value: %s", diags)
	}
	reqConfig := tfsdk.Config{Raw: configPlanForRaw.Raw, Schema: sch}

	// Plan: stands in for Terraform Core's own (unreliable, for this
	// schema shape) proposed value -- paatwood's is_writable planned false,
	// the literal defect observed against a real NAS in PLAT-705.
	planPermission := permissionSetFromEntries(t, []permEntry{
		{
			name:     "paatwood",
			readonly: types.BoolValue(false),
			writable: types.BoolValue(false),
			deny:     types.BoolValue(false),
			custom:   types.BoolValue(false),
			admin:    types.BoolValue(false),
		},
		{
			name:     "guest",
			readonly: types.BoolValue(false),
			writable: types.BoolValue(false),
			deny:     types.BoolValue(false),
			custom:   types.BoolValue(false),
			admin:    types.BoolValue(false),
		},
	})
	planModel := SharePermissionResourceModel{
		Share:         types.StringValue("photo"),
		UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
		Permission:    planPermission,
	}
	var reqPlan tfsdk.Plan
	reqPlan.Schema = sch
	if diags := reqPlan.Set(ctx, &planModel); diags.HasError() {
		t.Fatalf("building plan raw value: %s", diags)
	}

	// State: mirrors what `tofu import` would have populated -- paatwood
	// genuinely has is_writable = true on the real share today.
	statePermission := permissionSetFromEntries(t, []permEntry{
		{
			name:     "paatwood",
			readonly: types.BoolValue(false),
			writable: types.BoolValue(true),
			deny:     types.BoolValue(false),
			custom:   types.BoolValue(false),
			admin:    types.BoolValue(false),
		},
		{
			name:     "guest",
			readonly: types.BoolValue(false),
			writable: types.BoolValue(false),
			deny:     types.BoolValue(false),
			custom:   types.BoolValue(false),
			admin:    types.BoolValue(false),
		},
	})
	stateModel := SharePermissionResourceModel{
		Share:         types.StringValue("photo"),
		UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
		Permission:    statePermission,
	}
	var reqState tfsdk.State
	reqState.Schema = sch
	if diags := reqState.Set(ctx, &stateModel); diags.HasError() {
		t.Fatalf("building state raw value: %s", diags)
	}

	r := &SharePermissionResource{}
	req := resource.ModifyPlanRequest{
		Config: reqConfig,
		Plan:   reqPlan,
		State:  reqState,
	}
	resp := &resource.ModifyPlanResponse{
		Plan: reqPlan,
	}

	r.ModifyPlan(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() diagnostics: %s", resp.Diagnostics)
	}

	var correctedPermission types.Set
	if diags := resp.Plan.GetAttribute(ctx, path.Root("permission"), &correctedPermission); diags.HasError() {
		t.Fatalf("reading corrected permission from resp.Plan: %s", diags)
	}

	correctedModel := SharePermissionResourceModel{Permission: correctedPermission}
	entries, err := r.planEntries(ctx, correctedModel)
	if err != nil {
		t.Fatalf("planEntries() error = %v", err)
	}

	byName := map[string]core.SharePermissionSetEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	paatwood, ok := byName["paatwood"]
	if !ok {
		t.Fatal("paatwood missing from corrected plan entries")
	}
	if !paatwood.IsWritable {
		t.Errorf(
			"paatwood.IsWritable = false after ModifyPlan, want true: the configured "+
				"is_writable = true was dropped (entries=%+v)",
			entries,
		)
	}
	if paatwood.IsReadonly || paatwood.IsDeny || paatwood.IsCustom {
		t.Errorf("paatwood unexpected flags true: %+v, want only IsWritable", paatwood)
	}

	guest, ok := byName["guest"]
	if !ok {
		t.Fatal("guest missing from corrected plan entries")
	}
	if guest.IsReadonly || guest.IsWritable || guest.IsDeny || guest.IsCustom {
		t.Errorf("guest = %+v, want every flag false (unconfigured -> default)", guest)
	}
}

// TestModifyPlan_AllFlagsRoundTrip pins is_readonly, is_deny and is_custom
// alongside is_writable: PLAT-705 was found via is_writable, but the same
// Set/Computed/Default interaction is not specific to any one flag, so each
// must survive ModifyPlan when configured true.
func TestModifyPlan_AllFlagsRoundTrip(t *testing.T) {
	ctx := context.Background()
	sch := sharePermissionResourceSchemaForModifyPlanTest(t)

	cases := []struct {
		name    string
		configd permEntry
	}{
		{name: "is_readonly", configd: permEntry{name: "acct", readonly: types.BoolValue(true)}},
		{name: "is_writable", configd: permEntry{name: "acct", writable: types.BoolValue(true)}},
		{name: "is_deny", configd: permEntry{name: "acct", deny: types.BoolValue(true)}},
		{name: "is_custom", configd: permEntry{name: "acct", custom: types.BoolValue(true)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configModel := SharePermissionResourceModel{
				Share:         types.StringValue("photo"),
				UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
				Permission:    permissionSetFromEntries(t, []permEntry{tc.configd}),
			}
			var configPlanForRaw tfsdk.Plan
			configPlanForRaw.Schema = sch
			if diags := configPlanForRaw.Set(ctx, &configModel); diags.HasError() {
				t.Fatalf("building config raw value: %s", diags)
			}
			reqConfig := tfsdk.Config{Raw: configPlanForRaw.Raw, Schema: sch}

			// Deliberately wrong stand-in for Core's proposal: every flag
			// false, regardless of what config says.
			wrongPermission := permissionSetFromEntries(t, []permEntry{
				{
					name:     "acct",
					readonly: types.BoolValue(false),
					writable: types.BoolValue(false),
					deny:     types.BoolValue(false),
					custom:   types.BoolValue(false),
					admin:    types.BoolValue(false),
				},
			})
			planModel := SharePermissionResourceModel{
				Share:         types.StringValue("photo"),
				UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
				Permission:    wrongPermission,
			}
			var reqPlan tfsdk.Plan
			reqPlan.Schema = sch
			if diags := reqPlan.Set(ctx, &planModel); diags.HasError() {
				t.Fatalf("building plan raw value: %s", diags)
			}

			// No prior state: simulate Create, where ModifyPlan must not
			// look up an is_admin value that does not exist yet.
			reqState := tfsdk.State{
				Schema: sch,
				Raw:    tftypes.NewValue(sch.Type().TerraformType(ctx), nil),
			}

			r := &SharePermissionResource{}
			req := resource.ModifyPlanRequest{Config: reqConfig, Plan: reqPlan, State: reqState}
			resp := &resource.ModifyPlanResponse{Plan: reqPlan}

			r.ModifyPlan(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan() diagnostics: %s", resp.Diagnostics)
			}

			var correctedPermission types.Set
			if diags := resp.Plan.GetAttribute(ctx, path.Root("permission"), &correctedPermission); diags.HasError() {
				t.Fatalf("reading corrected permission: %s", diags)
			}
			entries, err := r.planEntries(ctx, SharePermissionResourceModel{Permission: correctedPermission})
			if err != nil {
				t.Fatalf("planEntries() error = %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("entries = %+v, want exactly 1", entries)
			}

			e := entries[0]
			got := map[string]bool{
				"is_readonly": e.IsReadonly,
				"is_writable": e.IsWritable,
				"is_deny":     e.IsDeny,
				"is_custom":   e.IsCustom,
			}
			if !got[tc.name] {
				t.Errorf("entries[0] = %+v, want %s = true", e, tc.name)
			}
			for flag, v := range got {
				if flag != tc.name && v {
					t.Errorf("entries[0] = %+v, want only %s true", e, tc.name)
				}
			}
		})
	}
}
