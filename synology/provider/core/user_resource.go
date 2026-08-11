package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client core.Api
}

type UserResourceModel struct {
	Name              types.String `tfsdk:"name"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int32  `tfsdk:"password_wo_version"`
	Description       types.String `tfsdk:"description"`
	Email             types.String `tfsdk:"email"`
	Expire            types.String `tfsdk:"expire"`
	CannotChangePass  types.Bool   `tfsdk:"cannot_change_password"`
	PasswdNeverExpire types.Bool   `tfsdk:"password_never_expire"`
	Groups            types.Set    `tfsdk:"groups"`
	UID               types.Int64  `tfsdk:"uid"`
}

func (r *UserResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "user")
}

func (r *UserResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DSM local user. The initial password is write-only " +
			"(`password_wo`) and is never stored in state; bump `password_wo_version` to rotate.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Username. Changing this forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password_wo": schema.StringAttribute{
				MarkdownDescription: "Initial password (or rotation value). Never stored in state. " +
					"Requires OpenTofu >= 1.11 write-only attribute support.",
				Optional:  true,
				WriteOnly: true,
				Sensitive: true,
			},
			"password_wo_version": schema.Int32Attribute{
				MarkdownDescription: "Increment to apply a new `password_wo` value (rotation). " +
					"OpenTofu cannot diff write-only attributes, so this is the only rotation signal.",
				Optional: true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "User description.",
				Optional:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "User email address.",
				Optional:            true,
			},
			"expire": schema.StringAttribute{
				MarkdownDescription: "Account expiry setting as accepted by DSM " +
					"(`never`, a date, etc.).",
				Optional: true,
			},
			"cannot_change_password": schema.BoolAttribute{
				MarkdownDescription: "When true, the user cannot change their own password.",
				Optional:            true,
			},
			"password_never_expire": schema.BoolAttribute{
				MarkdownDescription: "When true, the password never expires.",
				Optional:            true,
			},
			"groups": schema.SetAttribute{
				MarkdownDescription: "Group names the user belongs to. Membership is declared " +
					"only on the user resource (not on `synology_core_group`).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"uid": schema.Int64Attribute{
				MarkdownDescription: "Numeric user id assigned by DSM.",
				Computed:            true,
			},
		},
	}
}

func (r *UserResource) Configure(
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

func (r *UserResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only values are only readable from Config, never Plan/State.
	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.PasswordWO.IsNull() || config.PasswordWO.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password_wo"),
			"Password required on create",
			"password_wo must be set when creating a user.",
		)
		return
	}

	createReq := core.UserCreateRequest{
		Name:              plan.Name.ValueString(),
		Password:          config.PasswordWO.ValueString(),
		Description:       plan.Description.ValueString(),
		Email:             plan.Email.ValueString(),
		Expire:            plan.Expire.ValueString(),
		CannotChangePass:  plan.CannotChangePass.ValueBool(),
		PasswdNeverExpire: plan.PasswdNeverExpire.ValueBool(),
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		var groups []string
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Groups = groups
	}

	created, err := r.client.UserCreate(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}

	plan.UID = types.Int64Value(int64(created.ID))
	// Do not set PasswordWO on the model — write-only must stay null in state.
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := r.client.UserList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list users", err.Error())
		return
	}

	name := state.Name.ValueString()
	var found *core.User
	for i := range list.Users {
		if list.Users[i].Name == name {
			found = &list.Users[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if found.ID != 0 {
		state.UID = types.Int64Value(int64(found.ID))
	}
	// User.list additional fields are best-effort; keep prior state when DSM
	// returns empty strings rather than wiping managed attributes.
	if found.Description != "" {
		state.Description = types.StringValue(found.Description)
	}
	if found.Email != "" {
		state.Email = types.StringValue(found.Email)
	}
	if found.Expire != "" {
		state.Expire = types.StringValue(found.Expire)
	}
	// Groups are not returned by UserList on this DSM build; keep state membership.
	state.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mod := core.UserModifyRequest{
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		Email:             plan.Email.ValueString(),
		Expire:            plan.Expire.ValueString(),
		CannotChangePass:  plan.CannotChangePass.ValueBool(),
		PasswdNeverExpire: plan.PasswdNeverExpire.ValueBool(),
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		var groups []string
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		mod.Groups = groups
	}

	// Rotate password only when password_wo_version changes and a password is supplied.
	if !plan.PasswordWOVersion.Equal(state.PasswordWOVersion) &&
		!config.PasswordWO.IsNull() &&
		config.PasswordWO.ValueString() != "" {
		mod.Password = config.PasswordWO.ValueString()
	}

	modified, err := r.client.UserModify(ctx, mod)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update user", err.Error())
		return
	}

	plan.UID = types.Int64Value(int64(modified.ID))
	if plan.UID.ValueInt64() == 0 {
		plan.UID = state.UID
	}
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UserDelete(ctx, core.UserDeleteRequest{
		Name: state.Name.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
	}
}

func (r *UserResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
