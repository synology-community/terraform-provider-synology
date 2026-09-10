package provider

import (
	"context"
	"testing"
)

func TestBuildISOFromFiles_Deterministic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	files := map[string]string{"user-data": "This is a test"}

	a, err := buildISOFromFiles(ctx, "cidata", files)
	if err != nil {
		t.Fatalf("first build: %v", err)
	}
	b, err := buildISOFromFiles(ctx, "cidata", files)
	if err != nil {
		t.Fatalf("second build: %v", err)
	}
	if a == "" {
		t.Fatal("ISO is empty")
	}
	if a != b {
		t.Fatal("ISO output is not deterministic")
	}
}
