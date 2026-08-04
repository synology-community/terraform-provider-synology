package core_test

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
)

// The package resource's Update was an empty function body. Changing `version`
// produced a plan diff, an apply that reported success, and no change on the
// NAS; state then recorded a version the device did not have and the next plan
// was clean. The fix is RequiresReplace on everything DSM cannot change in
// place, and a real Update for `run` alone.
//
// These assertions pin that contract without touching a NAS. The acceptance
// test in package_resource_test.go installs MariaDB10 on real hardware, so it
// cannot be the only guard on this behaviour.

func packageSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	core.NewPackageResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// requiresReplace reports whether the named attribute carries a
// RequiresReplace plan modifier, whatever its concrete attribute type.
func requiresReplace(t *testing.T, s schema.Schema, name string) bool {
	t.Helper()
	attr, ok := s.Attributes[name]
	if !ok {
		t.Fatalf("attribute %q is not in the schema", name)
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
	case schema.MapAttribute:
		for _, m := range a.PlanModifiers {
			descriptions = append(descriptions, m.Description(context.Background()))
		}
	default:
		t.Fatalf("attribute %q has an unhandled type %T", name, attr)
	}

	// Match the modifier's own description of its behaviour, which reads
	// "If the value of this attribute changes, Terraform will destroy and
	// recreate the resource." Note it never contains the word "replace", and
	// the concrete type is the unexported requiresReplaceIfModifier, so both
	// the obvious substring and a type assertion are the wrong anchor.
	for _, d := range descriptions {
		if strings.Contains(strings.ToLower(d), "destroy and recreate") {
			return true
		}
	}
	return false
}

func TestPackageSchema_AttributesDSMCannotChangeInPlaceRequireReplacement(t *testing.T) {
	t.Parallel()
	s := packageSchema(t)

	// name: DSM has no rename. A different name is a different package.
	// version/url/file/beta: all require a reinstall on DSM.
	// wizard: consumed only during installation, so a change is inert
	//         without one.
	for _, name := range []string{"name", "version", "url", "file", "beta", "wizard"} {
		if !requiresReplace(t, s, name) {
			t.Errorf(
				"attribute %q has no RequiresReplace plan modifier; "+
					"without it a change routes to Update, which cannot apply it on DSM "+
					"and would report success while the NAS is unchanged",
				name,
			)
		}
	}
}

func TestPackageSchema_RunIsUpdatableInPlace(t *testing.T) {
	t.Parallel()
	s := packageSchema(t)

	// The carve-out. SYNO.Core.Package.Control exposes start and stop, so
	// toggling `run` must not destroy and reinstall the package -- that would
	// delete the data a package owns in order to stop a service.
	if requiresReplace(t, s, "run") {
		t.Error(
			"attribute \"run\" carries RequiresReplace; toggling it would uninstall and " +
				"reinstall the package. SYNO.Core.Package.Control start/stop exists, so it " +
				"must be handled by Update instead",
		)
	}
}

func TestPackageSchema_NameDocumentsThatReplacementCanRemoveData(t *testing.T) {
	t.Parallel()
	s := packageSchema(t)

	attr, ok := s.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatalf(`attribute "name" is not a StringAttribute`)
	}
	// Replacement here is correct, but its consequence is not obvious from the
	// plan: uninstalling a package can take its data with it. The generated
	// docs are the only place a reader learns that before running apply.
	if !strings.Contains(strings.ToLower(attr.MarkdownDescription), "uninstall") {
		t.Error(
			`attribute "name" does not document that changing it uninstalls the current ` +
				`package; that consequence only reaches users through the generated docs`,
		)
	}
}
