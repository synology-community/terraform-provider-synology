package container_test

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container"
)

func registrySchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	container.NewRegistryResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestRegistrySchema_AllConfigRequiresReplace(t *testing.T) {
	t.Parallel()
	s := registrySchema(t)
	for _, name := range []string{"name", "url", "enable_trust_ssc"} {
		if !attrRequiresReplace(t, s, name) {
			t.Errorf("attribute %q must RequiresReplace", name)
		}
	}
}
