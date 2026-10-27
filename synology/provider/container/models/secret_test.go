package models

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSecretValue_PreservesNullContent(t *testing.T) {
	t.Parallel()
	s := Secret{
		Name:    types.StringValue("db"),
		File:    types.StringValue("/volume1/platform/secrets/db"),
		Content: types.StringNull(),
	}
	obj, ok := s.Value().(types.Object)
	if !ok {
		t.Fatalf("Value() type %T, want types.Object", s.Value())
	}
	attrs := obj.Attributes()
	content, ok := attrs["content"].(types.String)
	if !ok {
		t.Fatalf("content type %T, want types.String", attrs["content"])
	}
	if !content.IsNull() {
		t.Fatalf(
			"content = %#v, want null (host-path secret must not become empty string)",
			content,
		)
	}
}
