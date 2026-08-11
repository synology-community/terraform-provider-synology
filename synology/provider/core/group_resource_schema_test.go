package core_test

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
)

func groupSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	core.NewGroupResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestGroupSchema_NameRequiresReplace(t *testing.T) {
	t.Parallel()
	if !requiresReplace(t, groupSchema(t), "name") {
		t.Error(`attribute "name" must RequiresReplace`)
	}
}

func TestGroupSchema_HasNoMembersAttribute(t *testing.T) {
	t.Parallel()
	s := groupSchema(t)
	if _, ok := s.Attributes["members"]; ok {
		t.Error("group must not declare members; membership lives on synology_core_user.groups")
	}
	if !strings.Contains(s.MarkdownDescription, "synology_core_user") {
		t.Error("schema description should point membership at the user resource")
	}
}
