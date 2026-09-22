package core

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology/pkg/api/core"
)

// stubSharePermissionAPI implements core.Api by embedding it (nil) and
// overriding only SharePermissionList/SharePermissionSet -- the two methods
// SharePermissionResource's apply()/refresh() paths reach. Any other method
// call would nil-pointer-panic; that's intentional (see stubTaskUpdateAPI in
// task_update_state_test.go for the precedent this follows).
type stubSharePermissionAPI struct {
	core.Api
	current []core.SharePermission
	// sets records every call SharePermissionSet receives, in order, so
	// tests can assert exactly what was sent to DSM.
	sets [][]core.SharePermissionSetEntry
}

func (s *stubSharePermissionAPI) SharePermissionList(
	_ context.Context,
	_ string,
	_ string,
) (*core.SharePermissionListResponse, error) {
	items := make([]core.SharePermission, len(s.current))
	copy(items, s.current)
	return &core.SharePermissionListResponse{Items: items}, nil
}

func (s *stubSharePermissionAPI) SharePermissionSet(
	_ context.Context,
	_ string,
	_ string,
	perms []core.SharePermissionSetEntry,
) error {
	cp := make([]core.SharePermissionSetEntry, len(perms))
	copy(cp, perms)
	s.sets = append(s.sets, cp)

	// Apply the write onto `current` so a subsequent SharePermissionList
	// reflects it, the same way DSM would -- SharePermissionSetEntry has no
	// IsAdmin field, so a row's admin flag is preserved untouched across a
	// Set the way DSM's real behaviour is documented to work.
	byName := map[string]core.SharePermission{}
	for _, item := range s.current {
		byName[item.Name] = item
	}
	for _, e := range perms {
		prior := byName[e.Name]
		byName[e.Name] = core.SharePermission{
			Name:       e.Name,
			IsAdmin:    prior.IsAdmin,
			IsCustom:   e.IsCustom,
			IsDeny:     e.IsDeny,
			IsReadonly: e.IsReadonly,
			IsWritable: e.IsWritable,
		}
	}
	items := make([]core.SharePermission, 0, len(byName))
	for _, v := range byName {
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	s.current = items
	return nil
}

func newSharePermissionModel(
	t *testing.T,
	share, userGroupType string,
	rows []sharePermissionEntryModel,
) SharePermissionResourceModel {
	t.Helper()
	set, diags := types.SetValueFrom(
		context.Background(),
		types.ObjectType{AttrTypes: sharePermissionEntryAttrTypes()},
		rows,
	)
	if diags.HasError() {
		t.Fatalf("SetValueFrom() diagnostics: %s", diags)
	}
	return SharePermissionResourceModel{
		Share:         types.StringValue(share),
		UserGroupType: types.StringValue(userGroupType),
		Permission:    set,
	}
}

// TestApply_RejectsAdminDeclaredEntries is the PLAT-705 guardrail: the
// administrators group always has read/write on every share and DSM offers
// no way to revoke it, so declaring a permission block for an
// administrator-flagged account must fail the plan rather than send a Set
// call DSM will silently no-op or that plans a change that can never
// converge.
func TestApply_RejectsAdminDeclaredEntries(t *testing.T) {
	ctx := context.Background()
	stub := &stubSharePermissionAPI{
		current: []core.SharePermission{
			{Name: "administrators", IsAdmin: true, IsReadonly: false, IsWritable: true},
		},
	}
	r := &SharePermissionResource{client: stub}

	plan := newSharePermissionModel(t, "myshare", core.ShareUserGroupTypeLocalGroup, []sharePermissionEntryModel{
		{
			Name:       types.StringValue("administrators"),
			IsReadonly: types.BoolValue(false),
			IsWritable: types.BoolValue(true),
			IsDeny:     types.BoolValue(false),
			IsCustom:   types.BoolValue(false),
			IsAdmin:    types.BoolValue(true),
		},
	})

	err := r.apply(ctx, plan)
	if err == nil {
		t.Fatal("apply() error = nil, want an error rejecting the administrators block")
	}
	if !strings.Contains(err.Error(), "administrators") {
		t.Errorf("apply() error = %q, want it to name the rejected account", err.Error())
	}
	if len(stub.sets) != 0 {
		t.Errorf("SharePermissionSet was called %d time(s), want 0: a rejected plan must not "+
			"reach DSM", len(stub.sets))
	}
}

// TestApply_RevokesRowsAbsentFromConfig pins full-list ownership: a row that
// exists remotely but is not declared in config must be sent back to DSM
// with every flag false (no access), not merely left alone. Otherwise
// out-of-band grants would survive every apply, which is the exact "fight
// other tooling" tradeoff this resource is documented to resolve by owning
// the whole list.
func TestApply_RevokesRowsAbsentFromConfig(t *testing.T) {
	ctx := context.Background()
	stub := &stubSharePermissionAPI{
		current: []core.SharePermission{
			{Name: "administrators", IsAdmin: true, IsWritable: true},
			{Name: "alice", IsWritable: true},
			{Name: "bob", IsReadonly: true},
		},
	}
	r := &SharePermissionResource{client: stub}

	// Config declares only alice; bob was granted out-of-band.
	plan := newSharePermissionModel(t, "myshare", core.ShareUserGroupTypeLocalUser, []sharePermissionEntryModel{
		{
			Name:       types.StringValue("alice"),
			IsReadonly: types.BoolValue(false),
			IsWritable: types.BoolValue(true),
			IsDeny:     types.BoolValue(false),
			IsCustom:   types.BoolValue(false),
		},
	})

	if err := r.apply(ctx, plan); err != nil {
		t.Fatalf("apply() error = %v", err)
	}
	if len(stub.sets) != 1 {
		t.Fatalf("SharePermissionSet called %d time(s), want 1", len(stub.sets))
	}

	byName := map[string]core.SharePermissionSetEntry{}
	for _, e := range stub.sets[0] {
		byName[e.Name] = e
	}

	if _, ok := byName["administrators"]; ok {
		t.Error("SharePermissionSet was sent an administrators row; it must never be touched")
	}
	alice, ok := byName["alice"]
	if !ok || !alice.IsWritable {
		t.Errorf("alice entry = %+v, ok=%v; want a writable entry preserved from config", alice, ok)
	}
	bob, ok := byName["bob"]
	if !ok {
		t.Fatal("bob (undeclared, out-of-band) was not sent a revoke entry")
	}
	if bob.IsReadonly || bob.IsWritable || bob.IsDeny || bob.IsCustom {
		t.Errorf("bob revoke entry = %+v, want every flag false (no access)", bob)
	}
}

// TestRefresh_ExcludesAdminRows guards the read side of the same PLAT-705
// constraint: an administrator-flagged row must never enter state, or every
// subsequent plan would show Terraform wanting to delete a block config
// never declared (state has it, config doesn't -- a permanent diff).
func TestRefresh_ExcludesAdminRows(t *testing.T) {
	ctx := context.Background()
	stub := &stubSharePermissionAPI{
		current: []core.SharePermission{
			{Name: "administrators", IsAdmin: true, IsWritable: true},
			{Name: "alice", IsWritable: true},
		},
	}
	r := &SharePermissionResource{client: stub}

	data := SharePermissionResourceModel{
		Share:         types.StringValue("myshare"),
		UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
	}
	if err := r.refresh(ctx, &data); err != nil {
		t.Fatalf("refresh() error = %v", err)
	}

	var rows []sharePermissionEntryModel
	if diags := data.Permission.ElementsAs(ctx, &rows, false); diags.HasError() {
		t.Fatalf("ElementsAs() diagnostics: %s", diags)
	}
	if len(rows) != 1 || rows[0].Name.ValueString() != "alice" {
		t.Errorf("refreshed permission rows = %+v, want exactly [alice]", rows)
	}
}

// TestRefresh_RoundTripsDenyDistinctFromNoAccess pins schema constraint 3:
// is_deny=true is a different state than every flag false, and refresh()
// must preserve that distinction rather than collapsing it.
func TestRefresh_RoundTripsDenyDistinctFromNoAccess(t *testing.T) {
	ctx := context.Background()
	stub := &stubSharePermissionAPI{
		current: []core.SharePermission{
			{Name: "denied-user", IsDeny: true},
			{Name: "no-access-user"},
		},
	}
	r := &SharePermissionResource{client: stub}

	data := SharePermissionResourceModel{
		Share:         types.StringValue("myshare"),
		UserGroupType: types.StringValue(core.ShareUserGroupTypeLocalUser),
	}
	if err := r.refresh(ctx, &data); err != nil {
		t.Fatalf("refresh() error = %v", err)
	}

	var rows []sharePermissionEntryModel
	if diags := data.Permission.ElementsAs(ctx, &rows, false); diags.HasError() {
		t.Fatalf("ElementsAs() diagnostics: %s", diags)
	}
	byName := map[string]sharePermissionEntryModel{}
	for _, row := range rows {
		byName[row.Name.ValueString()] = row
	}

	denied, ok := byName["denied-user"]
	if !ok || !denied.IsDeny.ValueBool() {
		t.Errorf("denied-user = %+v, ok=%v; want is_deny=true", denied, ok)
	}
	noAccess, ok := byName["no-access-user"]
	if !ok {
		t.Fatal("no-access-user missing from refreshed state")
	}
	if noAccess.IsDeny.ValueBool() || noAccess.IsReadonly.ValueBool() ||
		noAccess.IsWritable.ValueBool() {
		t.Errorf("no-access-user = %+v, want every flag false (distinct from an explicit deny)",
			noAccess)
	}
}

// TestPlanEntries_SortsByName pins the stable-ordering contract planEntries
// documents: successive applies of the same config must produce the same
// SharePermissionSet payload order, or every plan would show a spurious diff
// from set-element reordering alone.
func TestPlanEntries_SortsByName(t *testing.T) {
	ctx := context.Background()
	r := &SharePermissionResource{}

	plan := newSharePermissionModel(t, "myshare", core.ShareUserGroupTypeLocalUser, []sharePermissionEntryModel{
		{Name: types.StringValue("zed"), IsReadonly: types.BoolValue(true)},
		{Name: types.StringValue("alice"), IsWritable: types.BoolValue(true)},
	})

	entries, err := r.planEntries(ctx, plan)
	if err != nil {
		t.Fatalf("planEntries() error = %v", err)
	}
	if len(entries) != 2 || entries[0].Name != "alice" || entries[1].Name != "zed" {
		t.Errorf("planEntries() = %+v, want [alice, zed] in that order", entries)
	}
}
