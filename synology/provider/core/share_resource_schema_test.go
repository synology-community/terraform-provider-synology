package core_test

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
)

func shareSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	core.NewShareResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestShareSchema_NameAndVolumeRequireReplace(t *testing.T) {
	t.Parallel()
	s := shareSchema(t)
	for _, name := range []string{"name", "vol_path"} {
		if !requiresReplace(t, s, name) {
			t.Errorf("attribute %q must RequiresReplace", name)
		}
	}
}

func TestShareSchema_DescriptionIsUpdatable(t *testing.T) {
	t.Parallel()
	s := shareSchema(t)
	// desc must NOT force replace — that is the whole point of Share.set.
	if requiresReplace(t, s, "desc") {
		t.Error(`attribute "desc" must be updatable in place via Share.set`)
	}
}
