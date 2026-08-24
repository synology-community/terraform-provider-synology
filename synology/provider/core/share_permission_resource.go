package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

var (
	_ resource.Resource                = &SharePermissionResource{}
	_ resource.ResourceWithImportState = &SharePermissionResource{}
)

func NewSharePermissionResource() resource.Resource {
	return &SharePermissionResource{}
}

type SharePermissionResource struct {
	client core.Api
}

// SharePermissionResourceModel owns the *entire* SYNO.Core.Share.Permission
// list for one (share, user_group_type) pair -- see the schema description
// for why full ownership was chosen over a per-account resource.
type SharePermissionResourceModel struct {
	Share         types.String `tfsdk:"share"`
	UserGroupType types.String `tfsdk:"user_group_type"`
	Permission    types.Set    `tfsdk:"permission"`
}

type sharePermissionEntryModel struct {
	Name       types.String `tfsdk:"name"`
	IsReadonly types.Bool   `tfsdk:"is_readonly"`
	IsWritable types.Bool   `tfsdk:"is_writable"`
	IsDeny     types.Bool   `tfsdk:"is_deny"`
	IsCustom   types.Bool   `tfsdk:"is_custom"`
	IsAdmin    types.Bool   `tfsdk:"is_admin"`
}

func sharePermissionEntryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"is_readonly": types.BoolType,
		"is_writable": types.BoolType,
		"is_deny":     types.BoolType,
		"is_custom":   types.BoolType,
		"is_admin":    types.BoolType,
	}
}

func (r *SharePermissionResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "share_permission")
}

func (r *SharePermissionResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the complete `SYNO.Core.Share.Permission` list for one " +
			"share and one `user_group_type` (all local-user grants, or all local-group grants, " +
			"on that share -- never both from the same resource instance).\n\n" +
			"**Full ownership.** Every apply overwrites the declared side's permission list to " +
			"exactly the `permission` blocks in config. Any row added out-of-band -- DSM Control " +
			"Panel, another tool, an operator -- is treated as drift and removed on the next " +
			"apply. This mirrors how `synology_core_firewall_profile` already owns its full rule " +
			"set in this provider: a resource that only reconciled the rows it happened to " +
			"declare could never express \"revoke a grant someone else added,\" and two writers " +
			"of the same share's permissions could drift forever with no single source of truth. " +
			"Declare every row you want to exist; nothing else survives an apply.\n\n" +
			"**The `administrators` local group is not manageable here.** DSM grants it " +
			"read/write on every share unconditionally and offers no way to revoke it -- it is " +
			"excluded from state on every read and rejected if declared in config, so this " +
			"resource never plans a change it cannot converge.\n\n" +
			"**`is_admin` is read-only.** It reflects DSM's own admin-group flag on the row, not " +
			"a settable grant; it cannot be supplied in config.\n\n" +
			"**Deny vs. no access.** `is_deny = true` is an explicit deny entry, distinct from a " +
			"row with every flag false (no access granted, no explicit deny either).",
		Attributes: map[string]schema.Attribute{
			"share": schema.StringAttribute{
				MarkdownDescription: "Share name. Changing this forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_group_type": schema.StringAttribute{
				MarkdownDescription: "Which side of the permission list this resource owns: " +
					"`local_user` or `local_group`. Changing this forces replacement.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						core.ShareUserGroupTypeLocalUser,
						core.ShareUserGroupTypeLocalGroup,
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"permission": schema.SetNestedBlock{
				MarkdownDescription: "One permission row per account (user or group name, " +
					"matching `user_group_type`). Order does not matter.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Account (user or group) name.",
							Required:            true,
						},
						"is_readonly": schema.BoolAttribute{
							MarkdownDescription: "Read-only access.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"is_writable": schema.BoolAttribute{
							MarkdownDescription: "Read/write access.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"is_deny": schema.BoolAttribute{
							MarkdownDescription: "Explicit deny, distinct from no access " +
								"(all flags false).",
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
						"is_custom": schema.BoolAttribute{
							MarkdownDescription: "Custom (non-preset) permission.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"is_admin": schema.BoolAttribute{
							MarkdownDescription: "DSM-reported administrator flag for this row. " +
								"Read-only: reflects group membership, not a grant, and cannot be " +
								"set from config.",
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (r *SharePermissionResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(synology.Api)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected synology.Api, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client.CoreAPI()
}

// planEntries decodes the plan's permission set into client entries, in a
// stable (name-sorted) order so successive applies produce a stable diff.
func (r *SharePermissionResource) planEntries(
	ctx context.Context,
	data SharePermissionResourceModel,
) ([]core.SharePermissionSetEntry, error) {
	if data.Permission.IsNull() || data.Permission.IsUnknown() {
		return nil, nil
	}
	var rows []sharePermissionEntryModel
	if diags := data.Permission.ElementsAs(ctx, &rows, false); diags.HasError() {
		return nil, fmt.Errorf("permission decode: %s", diags.Errors())
	}
	entries := make([]core.SharePermissionSetEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, core.SharePermissionSetEntry{
			Name:       row.Name.ValueString(),
			IsReadonly: row.IsReadonly.ValueBool(),
			IsWritable: row.IsWritable.ValueBool(),
			IsDeny:     row.IsDeny.ValueBool(),
			IsCustom:   row.IsCustom.ValueBool(),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// rejectAdminEntries fails the plan if config declares a row that DSM
// reports as administrator-flagged (constant 1 above: never plan a change
// this resource can't converge -- administrators always has RW and cannot
// be revoked).
func (r *SharePermissionResource) rejectAdminEntries(
	entries []core.SharePermissionSetEntry,
	current *core.SharePermissionListResponse,
) error {
	admin := map[string]bool{}
	if current != nil {
		for _, item := range current.Items {
			if item.IsAdmin {
				admin[item.Name] = true
			}
		}
	}
	var bad []string
	for _, e := range entries {
		if admin[e.Name] {
			bad = append(bad, e.Name)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf(
			"account(s) %s are DSM administrator-flagged: they always have read/write on "+
				"every share and DSM offers no way to revoke it; remove the permission block(s) "+
				"declaring them",
			strings.Join(bad, ", "),
		)
	}
	return nil
}

func (r *SharePermissionResource) apply(
	ctx context.Context,
	data SharePermissionResourceModel,
) error {
	entries, err := r.planEntries(ctx, data)
	if err != nil {
		return err
	}

	share := data.Share.ValueString()
	userGroupType := data.UserGroupType.ValueString()

	current, err := r.client.SharePermissionList(ctx, share, userGroupType)
	if err != nil {
		return fmt.Errorf("permission list: %w", err)
	}
	if err := r.rejectAdminEntries(entries, current); err != nil {
		return err
	}

	// Full ownership: also revoke any row present remotely but absent from
	// config (dropped by name from the current list; a row with no flags
	// set is equivalent to no access).
	declared := map[string]bool{}
	for _, e := range entries {
		declared[e.Name] = true
	}
	if current != nil {
		for _, item := range current.Items {
			if item.IsAdmin || declared[item.Name] {
				continue
			}
			entries = append(entries, core.SharePermissionSetEntry{Name: item.Name})
		}
	}

	if err := r.client.SharePermissionSet(ctx, share, userGroupType, entries); err != nil {
		return fmt.Errorf("permission set: %w", err)
	}
	return nil
}

func (r *SharePermissionResource) refresh(
	ctx context.Context,
	data *SharePermissionResourceModel,
) error {
	got, err := r.client.SharePermissionList(
		ctx,
		data.Share.ValueString(),
		data.UserGroupType.ValueString(),
	)
	if err != nil {
		return err
	}

	items := make([]core.SharePermission, 0, len(got.Items))
	for _, item := range got.Items {
		// administrators (or any other DSM-flagged admin row) is never
		// managed by this resource and must never enter state -- otherwise
		// every plan would want to delete a row config never declared.
		if item.IsAdmin {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	rowObjs := make([]attr.Value, 0, len(items))
	for _, item := range items {
		obj, diags := types.ObjectValue(sharePermissionEntryAttrTypes(), map[string]attr.Value{
			"name":        types.StringValue(item.Name),
			"is_readonly": types.BoolValue(item.IsReadonly),
			"is_writable": types.BoolValue(item.IsWritable),
			"is_deny":     types.BoolValue(item.IsDeny),
			"is_custom":   types.BoolValue(item.IsCustom),
			"is_admin":    types.BoolValue(item.IsAdmin),
		})
		if diags.HasError() {
			return fmt.Errorf("permission object: %s", diags.Errors())
		}
		rowObjs = append(rowObjs, obj)
	}
	set, diags := types.SetValue(
		types.ObjectType{AttrTypes: sharePermissionEntryAttrTypes()},
		rowObjs,
	)
	if diags.HasError() {
		return fmt.Errorf("permission set: %s", diags.Errors())
	}
	data.Permission = set
	return nil
}

func (r *SharePermissionResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SharePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Failed to set share permissions", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read share permissions after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SharePermissionResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SharePermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refresh(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Failed to read share permissions", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SharePermissionResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan SharePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Failed to update share permissions", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read share permissions after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SharePermissionResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SharePermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Deleting the resource revokes every non-admin row it owned -- an
	// empty declared set, applied the same way Update applies a shrunk one.
	empty := SharePermissionResourceModel{
		Share:         state.Share,
		UserGroupType: state.UserGroupType,
	}
	if err := r.apply(ctx, empty); err != nil {
		resp.Diagnostics.AddError("Failed to revoke share permissions", err.Error())
	}
}

func (r *SharePermissionResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf(
				"Expected import id in the form share/user_group_type, got: %q",
				req.ID,
			),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("share"), parts[0])...)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("user_group_type"), parts[1])...)
}
