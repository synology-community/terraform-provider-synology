package core

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTaskReadRefreshesEnable(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		prior := taskModel()
		prior.Enable = types.BoolValue(!enabled)
		fake := &taskAPIFake{detail: taskFixture(t)}
		fake.detail.Enable = enabled
		p := &TaskResource{tasks: fake}
		state := taskState(t, p, prior)
		response := resource.ReadResponse{State: state}
		p.Read(t.Context(), resource.ReadRequest{State: state}, &response)
		if response.Diagnostics.HasError() {
			t.Fatal(response.Diagnostics)
		}
		want := prior
		want.Enable = types.BoolValue(enabled)
		if have := readModel(t, response.State); have != want || fake.reads != 1 {
			t.Fatal("refresh did not record the observed enabled state")
		}
	}
}

func TestTaskReadRejectsUnknownOrInvalidIdentity(t *testing.T) {
	for _, id := range []types.Int64{types.Int64Null(), types.Int64Unknown(), types.Int64Value(0), types.Int64Value(-1)} {
		fake := &taskAPIFake{}
		p := &TaskResource{tasks: fake}
		prior := taskModel()
		prior.ID = id
		state := taskState(t, p, prior)
		response := resource.ReadResponse{State: state}
		p.Read(t.Context(), resource.ReadRequest{State: state}, &response)
		if !response.Diagnostics.HasError() || fake.reads != 0 ||
			!response.State.Raw.Equal(state.Raw) {
			t.Fatal("invalid identity dispatched a read or changed state")
		}
	}
}
