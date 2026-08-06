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
	// volume_path: DSM cannot move an installed package between volumes, so
	//         changing it must reinstall rather than silently keep the data
	//         where it already is.
	for _, name := range []string{
		"name", "version", "url", "file", "beta", "wizard", "volume_path",
	} {
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

// TestPackageSchema_VolumePathIsOptionalNotComputed pins a choice that looks
// like an oversight and is not.
//
// Every other optional attribute on this resource is Optional+Computed with a
// default. volume_path is Optional alone, because the volume a package was
// installed onto is not reported back by the package list. A Computed
// attribute the provider cannot populate after apply produces "Provider
// produced inconsistent result after apply" -- the framework requires a known
// value for it, and there is none to be had.
//
// Null therefore means "not specified", and the client resolves the volume at
// install time. That is honest about what the provider knows.
func TestPackageSchema_VolumePathIsOptionalNotComputed(t *testing.T) {
	t.Parallel()
	s := packageSchema(t)

	attr, ok := s.Attributes["volume_path"].(schema.StringAttribute)
	if !ok {
		t.Fatalf(`attribute "volume_path" is not a StringAttribute`)
	}
	if !attr.Optional {
		t.Error(`attribute "volume_path" must be Optional: DSM resolves a volume when none is given`)
	}
	if attr.Computed {
		t.Error(
			`attribute "volume_path" must NOT be Computed. DSM does not report which ` +
				`volume a package was installed onto, so the provider cannot supply a ` +
				`known value after apply, and the framework fails with "inconsistent ` +
				`result after apply"`,
		)
	}
}

// TestPackageSchema_VolumePathDocumentsTheMultiVolumeCase guards the one thing
// a practitioner cannot discover from the plan: on a multi-volume NAS with no
// DSM default, omitting volume_path fails rather than picking a volume. Being
// refused is the intended behaviour, but only if the reason is discoverable
// before the apply that hits it.
func TestPackageSchema_VolumePathDocumentsTheMultiVolumeCase(t *testing.T) {
	t.Parallel()
	s := packageSchema(t)

	attr, ok := s.Attributes["volume_path"].(schema.StringAttribute)
	if !ok {
		t.Fatalf(`attribute "volume_path" is not a StringAttribute`)
	}
	doc := strings.ToLower(attr.MarkdownDescription)
	for _, want := range []string{"multi-volume", "error"} {
		if !strings.Contains(doc, want) {
			t.Errorf(
				`attribute "volume_path" does not document %q; a user on a multi-volume `+
					`NAS meets this as an apply failure with no hint that naming a volume `+
					`is the fix`,
				want,
			)
		}
	}
}

// TestPackageSchema_ComputedReplaceAttributesKeepTheirStateValue guards the
// defect that made RequiresReplace unusable on this resource.
//
// An Optional+Computed attribute the configuration does not set is planned as
// "(known after apply)" -- unknown. An unknown value on a RequiresReplace
// attribute forces replacement. So `version` and `url` forced a destroy-create
// on EVERY plan that changed anything, including a plain `run` toggle:
//
//	~ url     = "https://..." # forces replacement -> (known after apply)
//	~ version = "1.2.6-0260"  # forces replacement -> (known after apply)
//
// That defeats the whole point of the carve-out. `run` is meant to be the one
// attribute updatable in place, and it could not be, because two attributes
// nobody touched went unknown alongside it. On this resource replacement means
// uninstalling the package.
//
// UseStateForUnknown keeps the prior value instead. It is correct here rather
// than merely convenient: `version` and `url` only change when the package is
// actually reinstalled, which is itself a replacement.
func TestPackageSchema_ComputedReplaceAttributesKeepTheirStateValue(t *testing.T) {
	t.Parallel()
	s := packageSchema(t)

	for _, name := range []string{"version", "url"} {
		attr, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("attribute %q is not a StringAttribute", name)
		}
		if !attr.Computed {
			continue // only Computed attributes can plan as unknown
		}

		// Matched on the modifier's own Description, "Once set, the value of
		// this attribute in state will not change." Not on the word "unknown",
		// which does not appear in it, and not on a type assertion: the
		// concrete type is unexported.
		var found bool
		for _, m := range attr.PlanModifiers {
			if strings.Contains(
				strings.ToLower(m.Description(context.Background())),
				"will not change",
			) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(
				"attribute %q is Optional+Computed with RequiresReplace but has no "+
					"UseStateForUnknown; unset in config it plans as unknown, and unknown "+
					"on a RequiresReplace attribute forces replacement -- so `run` could "+
					"never be updated in place",
				name,
			)
		}
	}
}
