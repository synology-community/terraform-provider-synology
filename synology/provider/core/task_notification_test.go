package core

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	coreapi "github.com/synology-community/go-synology/pkg/api/core"
)

type stubTaskNotificationUpdateAPI struct {
	coreapi.Api
}

func (*stubTaskNotificationUpdateAPI) TaskUpdate(
	context.Context,
	coreapi.TaskRequest,
) (*coreapi.TaskResult, error) {
	return &coreapi.TaskResult{}, nil
}

func TestGetTaskRequestMapsNotifications(t *testing.T) {
	data := taskNotificationModel()
	data.NotifyEnable = types.BoolValue(true)
	data.NotifyIfError = types.BoolValue(true)
	data.NotifyMail = types.StringValue("admin@example.com")

	req, err := getTaskRequest(data)
	if err != nil {
		t.Fatalf("getTaskRequest() error = %v", err)
	}
	if !req.Extra.NotifyEnable || !req.Extra.NotifyIfError ||
		req.Extra.NotifyMail != "admin@example.com" {
		t.Errorf("notification request = %#v, want enabled error notifications", req.Extra)
	}
}

func TestUpdateWritesNotificationsToState(t *testing.T) {
	ctx := context.Background()
	provider := &TaskResource{client: &stubTaskNotificationUpdateAPI{}}
	schemaResp := &resource.SchemaResponse{}
	provider.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", schemaResp.Diagnostics)
	}

	prior := taskNotificationModel()
	planned := prior
	planned.NotifyEnable = types.BoolValue(true)
	planned.NotifyIfError = types.BoolValue(true)
	planned.NotifyMail = types.StringValue("admin@example.com")

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diagnostics := state.Set(ctx, &prior); diagnostics.HasError() {
		t.Fatalf("state.Set() diagnostics: %v", diagnostics)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diagnostics := plan.Set(ctx, &planned); diagnostics.HasError() {
		t.Fatalf("plan.Set() diagnostics: %v", diagnostics)
	}
	response := &resource.UpdateResponse{State: state}
	provider.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics: %v", response.Diagnostics)
	}

	var result TaskResourceModel
	if diagnostics := response.State.Get(ctx, &result); diagnostics.HasError() {
		t.Fatalf("response.State.Get() diagnostics: %v", diagnostics)
	}
	if !result.NotifyEnable.ValueBool() || !result.NotifyIfError.ValueBool() ||
		result.NotifyMail.ValueString() != "admin@example.com" {
		t.Errorf("notification state = %#v, want planned values", result)
	}
}

func TestValidateConfigRequiresNotificationMail(t *testing.T) {
	tests := []struct {
		name        string
		notify      types.Bool
		mail        types.String
		expectError bool
	}{
		{"enabled with mail", types.BoolValue(true), types.StringValue("admin@example.com"), false},
		{"enabled with empty mail", types.BoolValue(true), types.StringValue(""), true},
		{"enabled with null mail", types.BoolValue(true), types.StringNull(), true},
		{"disabled without mail", types.BoolValue(false), types.StringNull(), false},
		{"enabled with unknown mail", types.BoolValue(true), types.StringUnknown(), false},
	}

	ctx := context.Background()
	provider := &TaskResource{}
	schemaResp := &resource.SchemaResponse{}
	provider.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", schemaResp.Diagnostics)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := taskNotificationModel()
			model.NotifyEnable = tt.notify
			model.NotifyMail = tt.mail
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diagnostics := state.Set(ctx, &model); diagnostics.HasError() {
				t.Fatalf("state.Set() diagnostics: %v", diagnostics)
			}

			response := &resource.ValidateConfigResponse{}
			provider.ValidateConfig(ctx, resource.ValidateConfigRequest{
				Config: tfsdk.Config{Raw: state.Raw, Schema: schemaResp.Schema},
			}, response)
			if got := response.Diagnostics.HasError(); got != tt.expectError {
				t.Errorf("ValidateConfig() has error = %v, want %v", got, tt.expectError)
			}
		})
	}
}

func taskNotificationModel() TaskResourceModel {
	return TaskResourceModel{
		ID:            types.Int64Value(1),
		Name:          types.StringValue("test"),
		Service:       types.StringNull(),
		Script:        types.StringValue("echo test"),
		Schedule:      types.StringNull(),
		User:          types.StringValue("terraform"),
		NotifyEnable:  types.BoolValue(false),
		NotifyIfError: types.BoolValue(false),
		NotifyMail:    types.StringValue(""),
		Run:           types.BoolValue(false),
		When:          types.StringValue("apply"),
	}
}
