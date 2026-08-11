package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

var (
	_ resource.Resource                = &ShareResource{}
	_ resource.ResourceWithImportState = &ShareResource{}
)

func NewShareResource() resource.Resource {
	return &ShareResource{}
}

type ShareResource struct {
	client core.Api
}

type ShareResourceModel struct {
	Name                types.String `tfsdk:"name"`
	VolPath             types.String `tfsdk:"vol_path"`
	Desc                types.String `tfsdk:"desc"`
	Hidden              types.Bool   `tfsdk:"hidden"`
	EnableRecycleBin    types.Bool   `tfsdk:"enable_recycle_bin"`
	RecycleBinAdminOnly types.Bool   `tfsdk:"recycle_bin_admin_only"`
	EnableShareCompress types.Bool   `tfsdk:"enable_share_compress"`
	UUID                types.String `tfsdk:"uuid"`
}

func (r *ShareResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "share")
}

func (r *ShareResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DSM shared folder. Name and volume force replacement; " +
			"description, visibility, recycle bin, and compression update in place via " +
			"`SYNO.Core.Share` method `set`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Share name. Changing this forces replacement " +
					"(rename is not implemented).",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vol_path": schema.StringAttribute{
				MarkdownDescription: "Volume path, e.g. `/volume1`. Changing this forces " +
					"replacement — DSM does not move share data between volumes in place.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"desc": schema.StringAttribute{
				MarkdownDescription: "Share description.",
				Optional:            true,
			},
			"hidden": schema.BoolAttribute{
				MarkdownDescription: "Hide the share from browse lists.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"enable_recycle_bin": schema.BoolAttribute{
				MarkdownDescription: "Enable the share recycle bin.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"recycle_bin_admin_only": schema.BoolAttribute{
				MarkdownDescription: "Restrict recycle bin access to administrators.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"enable_share_compress": schema.BoolAttribute{
				MarkdownDescription: "Enable Btrfs compression for the share when supported.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "DSM-assigned share UUID.",
				Computed:            true,
			},
		},
	}
}

func (r *ShareResource) Configure(
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

func (r *ShareResource) shareInfo(data ShareResourceModel, nameOrg string) core.ShareInfo {
	return core.ShareInfo{
		Name:                data.Name.ValueString(),
		VolPath:             data.VolPath.ValueString(),
		Desc:                data.Desc.ValueString(),
		Hidden:              data.Hidden.ValueBool(),
		EnableRecycleBin:    data.EnableRecycleBin.ValueBool(),
		RecycleBinAdminOnly: data.RecycleBinAdminOnly.ValueBool(),
		EnableShareCompress: data.EnableShareCompress.ValueBool(),
		NameOrg:             nameOrg,
	}
}

func (r *ShareResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ShareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	info := r.shareInfo(plan, "")
	if err := r.client.ShareCreate(ctx, info); err != nil {
		resp.Diagnostics.AddError("Failed to create share", err.Error())
		return
	}

	// Create accepts a smaller shareinfo surface; apply full settings via set.
	if err := r.client.ShareModify(ctx, core.ShareModifyRequest{
		Name:      plan.Name.ValueString(),
		ShareInfo: r.shareInfo(plan, plan.Name.ValueString()),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to apply share settings after create", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read share after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ShareResource) refresh(ctx context.Context, data *ShareResourceModel) error {
	got, err := r.client.ShareGet(ctx, data.Name.ValueString())
	if err != nil {
		return err
	}
	data.UUID = types.StringValue(got.UUID)
	data.VolPath = types.StringValue(got.VolPath)
	data.Desc = types.StringValue(got.Desc)
	data.Hidden = types.BoolValue(got.Hidden)
	data.EnableRecycleBin = types.BoolValue(got.EnableRecycleBin)
	data.RecycleBinAdminOnly = types.BoolValue(got.RecycleBinAdminOnly)
	data.EnableShareCompress = types.BoolValue(got.EnableShareCompress)
	return nil
}

func (r *ShareResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state ShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refresh(ctx, &state); err != nil {
		// ShareGet returns an error when missing; treat as gone.
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(err.Error(), "no such") {
			resp.State.RemoveResource(ctx)
			return
		}
		// Some DSM builds return a generic API error for a missing share.
		list, lerr := r.client.ShareList(ctx)
		if lerr == nil {
			found := false
			for _, s := range list.Shares {
				if s.Name == state.Name.ValueString() {
					found = true
					break
				}
			}
			if !found {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError("Failed to read share", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ShareResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan ShareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.ShareModify(ctx, core.ShareModifyRequest{
		Name:      plan.Name.ValueString(),
		ShareInfo: r.shareInfo(plan, plan.Name.ValueString()),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to update share", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read share after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ShareResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.ShareDelete(ctx, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete share", err.Error())
	}
}

func (r *ShareResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
