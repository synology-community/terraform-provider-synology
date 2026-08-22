package core_test

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
)

func taskSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	core.NewTaskResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// PLAT-522: a scheduled task that does not schedule is not a useful default,
// so `enable` must default to true rather than DSM's own default of false.
func TestTaskSchema_EnableDefaultsTrue(t *testing.T) {
	t.Parallel()
	s := taskSchema(t)

	attr, ok := s.Attributes["enable"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf(`attribute "enable" is not a BoolAttribute`)
	}
	if !attr.Optional || !attr.Computed {
		t.Errorf("attribute \"enable\" Optional=%v Computed=%v, want both true", attr.Optional, attr.Computed)
	}
	if attr.Default == nil {
		t.Fatal(`attribute "enable" has no Default`)
	}

	defResp := &defaults.BoolResponse{}
	attr.Default.DefaultBool(context.Background(), defaults.BoolRequest{}, defResp)
	if !defResp.PlanValue.ValueBool() {
		t.Errorf("attribute \"enable\" default = %v, want true", defResp.PlanValue)
	}
}

// TestTaskSchema_EnableDescriptionDistinguishesFromRun pins the exact
// confusion PLAT-522 is about: `enable` is DSM's schedule on/off state, `run`
// is a one-shot execution. The docs are the only place a reader learns that
// before running apply.
func TestTaskSchema_EnableDescriptionDistinguishesFromRun(t *testing.T) {
	t.Parallel()
	s := taskSchema(t)

	attr, ok := s.Attributes["enable"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf(`attribute "enable" is not a BoolAttribute`)
	}
	desc := strings.ToLower(attr.MarkdownDescription)
	if !strings.Contains(desc, "run") {
		t.Error(`attribute "enable" description does not mention "run", so it does not ` +
			`distinguish the two attributes`)
	}
	if !strings.Contains(desc, "crontab") && !strings.Contains(desc, "schedule") {
		t.Error(`attribute "enable" description does not explain that it is DSM's schedule ` +
			`enabled state`)
	}
}
