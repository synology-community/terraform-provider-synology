package container

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/docker"
)

var (
	_ resource.Resource                = &RegistryResource{}
	_ resource.ResourceWithImportState = &RegistryResource{}
)

func NewRegistryResource() resource.Resource {
	return &RegistryResource{}
}

type RegistryResource struct {
	client docker.Api
}

type RegistryResourceModel struct {
	Name           types.String `tfsdk:"name"`
	URL            types.String `tfsdk:"url"`
	EnableTrustSSC types.Bool   `tfsdk:"enable_trust_ssc"`
	Syno           types.Bool   `tfsdk:"syno"`
}

func (r *RegistryResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "registry")
}

func (r *RegistryResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Container Manager registry entry. DSM exposes create " +
			"and delete for third-party registries; `set` rejects the documented parameter " +
			"shapes on the tested DSM, so every attribute forces replacement.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Registry display name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Registry base URL.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enable_trust_ssc": schema.BoolAttribute{
				MarkdownDescription: "Trust self-signed certificates for this registry.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"syno": schema.BoolAttribute{
				MarkdownDescription: "True when this is a built-in Synology registry (read-only).",
				Computed:            true,
			},
		},
	}
}

func (r *RegistryResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Api)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Api, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c.DockerAPI()
}

func (r *RegistryResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RegistryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	trust := plan.EnableTrustSSC.ValueBool()
	create := docker.RegistryCreateRequest{
		Name: plan.Name.ValueString(),
		URL:  plan.URL.ValueString(),
	}
	if !plan.EnableTrustSSC.IsNull() && !plan.EnableTrustSSC.IsUnknown() {
		create.EnableTrustSSC = &trust
	}

	if err := r.client.RegistryCreate(ctx, create); err != nil {
		resp.Diagnostics.AddError("Failed to create registry", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read registry after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RegistryResource) refresh(ctx context.Context, data *RegistryResourceModel) error {
	list, err := r.client.RegistryList(ctx, docker.ListRegistryRequest{})
	if err != nil {
		return err
	}
	for _, reg := range list.Registries {
		if reg.Name == data.Name.ValueString() {
			data.URL = types.StringValue(reg.URL)
			data.EnableTrustSSC = types.BoolValue(reg.EnableTrustSSC)
			data.Syno = types.BoolValue(reg.Syno)
			return nil
		}
	}
	return fmt.Errorf("registry %q not found", data.Name.ValueString())
}

func (r *RegistryResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state RegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refresh(ctx, &state); err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RegistryResource) Update(
	_ context.Context,
	_ resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddError(
		"Unsupported in-place registry update",
		"synology_container_registry forces replacement for every attribute; "+
			"DSM's registry set method rejected parameter shapes on the tested firmware.",
	)
}

func (r *RegistryResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state RegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Syno.ValueBool() {
		resp.Diagnostics.AddError(
			"Cannot delete built-in registry",
			fmt.Sprintf("Registry %q is a built-in Synology registry", state.Name.ValueString()),
		)
		return
	}

	if err := r.client.RegistryDelete(ctx, docker.RegistryDeleteRequest{
		Name: state.Name.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to delete registry", err.Error())
	}
}

func (r *RegistryResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
