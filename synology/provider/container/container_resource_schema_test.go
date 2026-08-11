package container_test

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container"
)

func containerSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	container.NewContainerResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func attrRequiresReplace(t *testing.T, s schema.Schema, name string) bool {
	t.Helper()
	attr, ok := s.Attributes[name]
	if !ok {
		t.Fatalf("missing attribute %q", name)
	}
	var descriptions []string
	switch a := attr.(type) {
	case schema.StringAttribute:
		for _, m := range a.PlanModifiers {
			descriptions = append(descriptions, m.Description(context.Background()))
		}
	case schema.BoolAttribute:
		for _, m := range a.PlanModifiers {
			descriptions = append(descriptions, m.Description(context.Background()))
		}
	case schema.Int64Attribute:
		for _, m := range a.PlanModifiers {
			descriptions = append(descriptions, m.Description(context.Background()))
		}
	case schema.ListAttribute:
		for _, m := range a.PlanModifiers {
			descriptions = append(descriptions, m.Description(context.Background()))
		}
	case schema.MapAttribute:
		for _, m := range a.PlanModifiers {
			descriptions = append(descriptions, m.Description(context.Background()))
		}
	default:
		t.Fatalf("unhandled type %T for %q", attr, name)
	}
	for _, d := range descriptions {
		if strings.Contains(strings.ToLower(d), "destroy and recreate") {
			return true
		}
	}
	return false
}

func TestContainerSchema_ConfigAttributesRequireReplace(t *testing.T) {
	t.Parallel()
	s := containerSchema(t)
	for _, name := range []string{
		"name", "image", "cmd", "privileged", "use_host_network",
		"enable_restart_policy", "cpu_priority", "memory_limit", "network", "env", "run_instantly",
	} {
		if !attrRequiresReplace(t, s, name) {
			t.Errorf("attribute %q must RequiresReplace (no DSM container update API)", name)
		}
	}
}
