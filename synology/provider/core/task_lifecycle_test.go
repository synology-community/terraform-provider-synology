package core

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core/taskclient"
)

func TestTaskTK07ReadErrorsRetainState(t *testing.T) {
	for _, err := range []error{errors.New("permission denied"), errors.New("SID not found"), errors.New("TLS failure"), context.DeadlineExceeded, errors.New("malformed JSON")} {
		t.Run(err.Error(), func(t *testing.T) {
			f := &taskAPIFake{readErr: err}
			p := &TaskResource{tasks: f}
			state := taskState(t, p, taskModel())
			response := resource.ReadResponse{State: state}
			p.Read(t.Context(), resource.ReadRequest{State: state}, &response)
			if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatal("read error discarded state")
			}
		})
	}
	f := &taskAPIFake{readErr: taskclient.ErrAbsent}
	p := &TaskResource{tasks: f}
	state := taskState(t, p, taskModel())
	response := resource.ReadResponse{State: state}
	p.Read(t.Context(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
		t.Fatal("typed absence did not remove state")
	}
}
