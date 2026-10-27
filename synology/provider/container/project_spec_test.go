package container

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"
)

func TestShouldUploadFileContent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content types.String
		want    bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), false},
		{"set", types.StringValue("secret-bytes"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldUploadFileContent(tc.content); got != tc.want {
				t.Fatalf("shouldUploadFileContent(%v) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestProjectSpecChanged_RunOnly(t *testing.T) {
	t.Parallel()
	base := models.ProjectResourceModel{
		Name:      types.StringValue("web"),
		SharePath: types.StringValue("/projects/web"),
		Content:   types.StringValue("services: {}"),
		Run:       types.BoolValue(true),
		Services:  types.MapNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		Configs:   types.MapNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		Secrets:   types.MapNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		Networks:  types.MapNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		Volumes:   types.MapNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
	}
	plan := base
	plan.Run = types.BoolValue(false)
	if projectSpecChanged(plan, base) {
		t.Fatal("run-only change must not count as a spec change")
	}
}

func TestProjectSpecChanged_Networks(t *testing.T) {
	t.Parallel()
	elem := types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}
	state := models.ProjectResourceModel{
		Name:      types.StringValue("web"),
		SharePath: types.StringValue("/projects/web"),
		Content:   types.StringValue("services: {}"),
		Run:       types.BoolValue(false),
		Services:  types.MapNull(elem),
		Configs:   types.MapNull(elem),
		Secrets:   types.MapNull(elem),
		Networks:  types.MapNull(elem),
		Volumes:   types.MapNull(elem),
	}
	plan := state
	plan.Networks = types.MapValueMust(elem, map[string]attr.Value{})
	if !projectSpecChanged(plan, state) {
		t.Fatal("networks change must count as a spec change so ProjectUpdate runs")
	}
}
