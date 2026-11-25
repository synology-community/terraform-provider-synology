package core_test

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
)

func sharePermissionSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	core.NewSharePermissionResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestSharePermissionSchema_ShareAndUserGroupTypeRequireReplace(t *testing.T) {
	t.Parallel()
	s := sharePermissionSchema(t)
	for _, name := range []string{"share", "user_group_type"} {
		if !requiresReplace(t, s, name) {
			t.Errorf("attribute %q must RequiresReplace", name)
		}
	}
}

func TestSharePermissionSchema_UserGroupTypeIsRestrictedToTheTwoKnownValues(t *testing.T) {
	t.Parallel()
	s := sharePermissionSchema(t)
	attr, ok := s.Attributes["user_group_type"].(schema.StringAttribute)
	if !ok {
		t.Fatal(`attribute "user_group_type" is not a StringAttribute`)
	}
	if len(attr.Validators) == 0 {
		t.Fatal(`attribute "user_group_type" has no validators; it must be restricted to ` +
			`"local_user"/"local_group" so a typo fails at plan time rather than as an ` +
			`opaque DSM error`)
	}

	ctx := context.Background()
	req := validator.StringRequest{ConfigValue: types.StringValue("bogus_value")}
	for _, v := range attr.Validators {
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("validator %T accepted the bogus value %q", v, "bogus_value")
		}
	}
}

func TestSharePermissionSchema_PermissionBlockCoversDenyAndNoAccessSeparately(t *testing.T) {
	t.Parallel()
	s := sharePermissionSchema(t)
	block, ok := s.Blocks["permission"].(schema.SetNestedBlock)
	if !ok {
		t.Fatal(`block "permission" is not a SetNestedBlock (order must not matter)`)
	}

	for _, name := range []string{"is_readonly", "is_writable", "is_deny", "is_custom"} {
		attr, ok := block.NestedObject.Attributes[name].(schema.BoolAttribute)
		if !ok {
			t.Fatalf("permission attribute %q is not a BoolAttribute", name)
		}
		if !attr.Optional {
			t.Errorf("permission attribute %q must be Optional (settable from config)", name)
		}
	}
}

func TestSharePermissionSchema_IsAdminIsComputedOnlyNeverConfigurable(t *testing.T) {
	t.Parallel()
	s := sharePermissionSchema(t)
	block, ok := s.Blocks["permission"].(schema.SetNestedBlock)
	if !ok {
		t.Fatal(`block "permission" is not a SetNestedBlock`)
	}
	attr, ok := block.NestedObject.Attributes["is_admin"].(schema.BoolAttribute)
	if !ok {
		t.Fatal(`permission attribute "is_admin" is not a BoolAttribute`)
	}
	if !attr.Computed {
		t.Error(`"is_admin" must be Computed: it reflects DSM's admin-group flag, not a grant`)
	}
	if attr.Optional || attr.Required {
		t.Error(`"is_admin" must not be Optional or Required: config must never set it`)
	}
}

func TestSharePermissionSchema_NameAttributeIsRequired(t *testing.T) {
	t.Parallel()
	s := sharePermissionSchema(t)
	block, ok := s.Blocks["permission"].(schema.SetNestedBlock)
	if !ok {
		t.Fatal(`block "permission" is not a SetNestedBlock`)
	}
	attr, ok := block.NestedObject.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatal(`permission attribute "name" is not a StringAttribute`)
	}
	if !attr.Required {
		t.Error(`permission attribute "name" must be Required`)
	}
}

// TestSharePermissionSchema_DocumentsAdministratorsAndFullOwnership pins the
// two behaviours a practitioner cannot discover from the plan alone: the
// administrators group can never be managed here, and this resource owns
// (and will delete) the entire remote list for its (share, user_group_type).
func TestSharePermissionSchema_DocumentsAdministratorsAndFullOwnership(t *testing.T) {
	t.Parallel()
	s := sharePermissionSchema(t)
	doc := strings.ToLower(s.MarkdownDescription)
	for _, want := range []string{"administrators", "full ownership", "firewall_profile"} {
		if !strings.Contains(doc, want) {
			t.Errorf("schema description does not mention %q", want)
		}
	}
}
