package core

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PLAT-522: synology_core_task created every DSM task disabled because
// getTaskRequest never set TaskRequest.Enable at all, so DSM's own default
// (disabled) always won and the resource could never enable, let alone
// disable, a scheduled task's crontab row. These pin the mapping from the
// resource model onto the wire request, for both directions -- a regression
// that only broke "true" would look identical to a superficial read.
func TestGetTaskRequest_MapsEnableTrue(t *testing.T) {
	data := TaskResourceModel{
		Name:   types.StringValue("test"),
		User:   types.StringValue("terraform"),
		Enable: types.BoolValue(true),
	}

	req, err := getTaskRequest(data)
	if err != nil {
		t.Fatalf("getTaskRequest() error = %v", err)
	}
	if !req.Enable {
		t.Errorf("getTaskRequest().Enable = %v, want true", req.Enable)
	}
}

func TestGetTaskRequest_MapsEnableFalse(t *testing.T) {
	data := TaskResourceModel{
		Name:   types.StringValue("test"),
		User:   types.StringValue("terraform"),
		Enable: types.BoolValue(false),
	}

	req, err := getTaskRequest(data)
	if err != nil {
		t.Fatalf("getTaskRequest() error = %v", err)
	}
	if req.Enable {
		t.Errorf("getTaskRequest().Enable = %v, want false", req.Enable)
	}
}
