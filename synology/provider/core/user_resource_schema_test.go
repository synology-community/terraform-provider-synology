package core_test

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
)

func userSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	core.NewUserResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestUserSchema_NameRequiresReplace(t *testing.T) {
	t.Parallel()
	s := userSchema(t)
	if !requiresReplace(t, s, "name") {
		t.Error(`attribute "name" must RequiresReplace`)
	}
}

func TestUserSchema_PasswordIsWriteOnly(t *testing.T) {
	t.Parallel()
	s := userSchema(t)
	attr, ok := s.Attributes["password_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatal(`password_wo is not a StringAttribute`)
	}
	if !attr.WriteOnly {
		t.Error(`password_wo must be WriteOnly so the value never enters state`)
	}
	if !attr.Sensitive {
		t.Error(`password_wo must be Sensitive`)
	}
	if attr.Computed {
		t.Error(`password_wo must not be Computed`)
	}
}

func TestUserSchema_DocumentsWriteOnlyPassword(t *testing.T) {
	t.Parallel()
	s := userSchema(t)
	attr := s.Attributes["password_wo"].(schema.StringAttribute)
	if !strings.Contains(strings.ToLower(attr.MarkdownDescription), "never stored") {
		t.Errorf("password_wo description should say it is never stored; got %q", attr.MarkdownDescription)
	}
}
