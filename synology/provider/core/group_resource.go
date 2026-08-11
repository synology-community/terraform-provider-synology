package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

var (
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
)

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

type GroupResource struct {
	client core.Api
}

type GroupResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	GID         types.Int64  `tfsdk:"gid"`
}

func (r *GroupResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "group")
}

func (r *GroupResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DSM local group. Membership is declared on " +
			"`synology_core_user.groups`, not here — two resources claiming the same " +
			"relationship would fight on every plan.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Group name. Changing this forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Group description.",
				Optional:            true,
			},
			"gid": schema.Int64Attribute{
				MarkdownDescription: "Numeric group id assigned by DSM.",
				Computed:            true,
			},
		},
	}
}

func (r *GroupResource) Configure(
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

func (r *GroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan GroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.GroupCreate(ctx, core.GroupCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create group", err.Error())
		return
	}

	plan.GID = types.Int64Value(int64(created.ID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := r.client.GroupList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list groups", err.Error())
		return
	}

	name := state.Name.ValueString()
	var found *core.Group
	for i := range list.Groups {
		if list.Groups[i].Name == name {
			found = &list.Groups[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Group.list often omits gid/description even when requested. Keep prior
	// state rather than zeroing fields DSM declined to return.
	if found.ID != 0 {
		state.GID = types.Int64Value(int64(found.ID))
	}
	if found.Description != "" {
		state.Description = types.StringValue(found.Description)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state GroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modified, err := r.client.GroupModify(ctx, core.GroupModifyRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update group", err.Error())
		return
	}

	plan.GID = types.Int64Value(int64(modified.ID))
	if plan.GID.ValueInt64() == 0 {
		plan.GID = state.GID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.GroupDelete(ctx, core.GroupDeleteRequest{
		Name: state.Name.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to delete group", err.Error())
	}
}

func (r *GroupResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
