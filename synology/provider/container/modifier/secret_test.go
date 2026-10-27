package modifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"
)

func TestSetSecretPathsFromContent(t *testing.T) {
	ctx := context.Background()
	modifier := SetSecretPathsFromContent()

	// Test the description methods
	if modifier.Description(ctx) == "" {
		t.Error("Description should not be empty")
	}

	if modifier.MarkdownDescription(ctx) == "" {
		t.Error("MarkdownDescription should not be empty")
	}

	// Test that the modifier implements the correct interface
	_ = modifier
}

func TestSetSecretPathsFromContentType(t *testing.T) {
	modifier := SetSecretPathsFromContent()

	// Verify it returns the correct type
	if modifier == nil {
		t.Error("SetSecretPathsFromContent should return a non-nil modifier")
	}
}

func TestPopulateSecretPaths_FileOnlyKeepsNullContent(t *testing.T) {
	t.Parallel()
	secret := models.Secret{
		Name:    types.StringValue("db"),
		File:    types.StringValue("/volume1/platform/secrets/db"),
		Content: types.StringNull(),
	}
	src := types.MapValueMust(secret.ModelType(), map[string]attr.Value{
		"db": secret.Value(),
	})

	var dst types.Map
	diags := populateSecretPathsInMap(context.Background(), path.Root("secrets"), src, &dst)
	if diags.HasError() {
		t.Fatalf("populateSecretPathsInMap: %v", diags)
	}

	out := map[string]models.Secret{}
	if d := dst.ElementsAs(context.Background(), &out, false); d.HasError() {
		t.Fatalf("ElementsAs: %v", d)
	}
	got := out["db"]
	if !got.Content.IsNull() {
		t.Fatalf("content = %#v, want null so handleSecrets skips File Station", got.Content)
	}
	if got.File.ValueString() != "/volume1/platform/secrets/db" {
		t.Fatalf("file = %q, want host path", got.File.ValueString())
	}
}
