package core

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core/taskclient"
)

type taskAPIFake struct {
	detail  *taskclient.Detail
	readErr error
	reads   int
}

func (f *taskAPIFake) Read(context.Context, int64) (*taskclient.Detail, error) {
	f.reads++
	if f.readErr != nil {
		return nil, f.readErr
	}
	snapshot := *f.detail
	return &snapshot, nil
}

func taskFixture(t *testing.T) *taskclient.Detail {
	t.Helper()
	body, err := os.ReadFile("taskclient/testdata/script.json")
	if err != nil {
		t.Fatal(err)
	}
	detail, err := taskclient.Decode(body, 42)
	if err != nil {
		t.Fatal(err)
	}
	return detail
}

func taskModel() TaskResourceModel {
	return TaskResourceModel{
		ID:            types.Int64Value(42),
		Name:          types.StringValue("Example maintenance runner"),
		User:          types.StringValue("runner"),
		Script:        types.StringValue("/usr/local/sbin/example-maintenance"),
		Schedule:      types.StringValue("17 3 * * *"),
		Enable:        types.BoolValue(false),
		NotifyEnable:  types.BoolValue(true),
		NotifyIfError: types.BoolValue(true),
		NotifyMail:    types.StringValue("operator@example.test"),
		Run:           types.BoolValue(false),
		When:          types.StringValue("apply"),
	}
}

func taskState(t *testing.T, p *TaskResource, model TaskResourceModel) tfsdk.State {
	t.Helper()
	var response resource.SchemaResponse
	p.Schema(t.Context(), resource.SchemaRequest{}, &response)
	if diags := response.Schema.ValidateImplementation(t.Context()); diags.HasError() {
		t.Fatal(diags)
	}
	state := tfsdk.State{Schema: response.Schema}
	if diags := state.Set(t.Context(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func readModel(t *testing.T, state tfsdk.State) TaskResourceModel {
	t.Helper()
	var model TaskResourceModel
	if diags := state.Get(t.Context(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}
