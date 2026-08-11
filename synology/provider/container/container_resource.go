package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/docker"
)

var (
	_ resource.Resource                = &ContainerResource{}
	_ resource.ResourceWithImportState = &ContainerResource{}
)

func NewContainerResource() resource.Resource {
	return &ContainerResource{}
}

type ContainerResource struct {
	client docker.Api
}

// ContainerResourceModel is a deliberately narrow profile: DSM has no container
// update method, so every configuration attribute forces replacement.
type ContainerResourceModel struct {
	Name                types.String `tfsdk:"name"`
	Image               types.String `tfsdk:"image"`
	Cmd                 types.String `tfsdk:"cmd"`
	Privileged          types.Bool   `tfsdk:"privileged"`
	UseHostNetwork      types.Bool   `tfsdk:"use_host_network"`
	EnableRestartPolicy types.Bool   `tfsdk:"enable_restart_policy"`
	CPUPriority         types.Int64  `tfsdk:"cpu_priority"`
	MemoryLimit         types.Int64  `tfsdk:"memory_limit"`
	Network             types.List   `tfsdk:"network"`
	Env                 types.Map    `tfsdk:"env"`
	RunInstantly        types.Bool   `tfsdk:"run_instantly"`
	Status              types.String `tfsdk:"status"`
}

func (r *ContainerResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "container")
}

func (r *ContainerResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a standalone Container Manager container. DSM has no " +
			"in-place update for container profiles, so configuration changes force " +
			"replacement (stop/delete/create). Start, stop, and restart are available as " +
			"the provider action, not attributes on this resource.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Container name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image reference (repository:tag or digest). The image " +
					"must already exist on the NAS or be pullable.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cmd": schema.StringAttribute{
				MarkdownDescription: "Command string passed to the container.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"privileged": schema.BoolAttribute{
				MarkdownDescription: "Run the container privileged.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"use_host_network": schema.BoolAttribute{
				MarkdownDescription: "Use the host network stack.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"enable_restart_policy": schema.BoolAttribute{
				MarkdownDescription: "Enable Container Manager's restart policy.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"cpu_priority": schema.Int64Attribute{
				MarkdownDescription: "CPU priority (DSM scale).",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"memory_limit": schema.Int64Attribute{
				MarkdownDescription: "Memory limit in bytes (0 = unlimited).",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network": schema.ListAttribute{
				MarkdownDescription: "Network names to attach (e.g. `bridge`).",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"env": schema.MapAttribute{
				MarkdownDescription: "Environment variables as a string map.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"run_instantly": schema.BoolAttribute{
				MarkdownDescription: "Start the container immediately after create. " +
					"Only consumed at create time.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Last observed container status from DSM.",
				Computed:            true,
			},
		},
	}
}

func (r *ContainerResource) Configure(
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

func (r *ContainerResource) profile(ctx context.Context, data ContainerResourceModel) (docker.Container, error) {
	c := docker.Container{
		Name:                data.Name.ValueString(),
		Image:               data.Image.ValueString(),
		Cmd:                 data.Cmd.ValueString(),
		Privileged:          data.Privileged.ValueBool(),
		UseHostNetwork:      data.UseHostNetwork.ValueBool(),
		EnableRestartPolicy: data.EnableRestartPolicy.ValueBool(),
		CPUPriority:         data.CPUPriority.ValueInt64(),
		MemoryLimit:         data.MemoryLimit.ValueInt64(),
	}

	if !data.Network.IsNull() && !data.Network.IsUnknown() {
		var nets []string
		diags := data.Network.ElementsAs(ctx, &nets, false)
		if diags.HasError() {
			return c, fmt.Errorf("invalid network list")
		}
		for _, n := range nets {
			c.Network = append(c.Network, docker.ContainerNetwork{Name: n})
		}
	}

	if !data.Env.IsNull() && !data.Env.IsUnknown() {
		m := map[string]string{}
		diags := data.Env.ElementsAs(ctx, &m, false)
		if diags.HasError() {
			return c, fmt.Errorf("invalid env map")
		}
		for k, v := range m {
			c.EnvVariables = append(c.EnvVariables, docker.EnvVariable{Key: k, Value: v})
		}
	}
	return c, nil
}

func (r *ContainerResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ContainerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, err := r.profile(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid container configuration", err.Error())
		return
	}

	if _, err := r.client.ContainerCreate(ctx, docker.CreateContainerRequest{
		Container:      profile,
		IsRunInstantly: plan.RunInstantly.ValueBool(),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to create container", err.Error())
		return
	}

	r.readStatus(ctx, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ContainerResource) readStatus(ctx context.Context, data *ContainerResourceModel) {
	list, err := r.client.ContainerList(ctx, docker.ContainerListRequest{Limit: -1})
	if err != nil {
		return
	}
	for _, c := range list.Containers {
		if c.Name == data.Name.ValueString() {
			if c.Status != "" {
				data.Status = types.StringValue(c.Status)
			}
			if c.Image != "" {
				// Keep planned image; list may return a digest form.
			}
			return
		}
	}
}

func (r *ContainerResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state ContainerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := r.client.ContainerList(ctx, docker.ContainerListRequest{Limit: -1})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list containers", err.Error())
		return
	}

	found := false
	for _, c := range list.Containers {
		if c.Name == state.Name.ValueString() {
			found = true
			if c.Status != "" {
				state.Status = types.StringValue(c.Status)
			}
			break
		}
	}
	if !found {
		// Confirm with Get — list can lag briefly.
		if _, gerr := r.client.ContainerGet(ctx, state.Name.ValueString()); gerr != nil {
			resp.State.RemoveResource(ctx)
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ContainerResource) Update(
	_ context.Context,
	_ resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	// Every config attribute RequiresReplace; reaching Update is a schema bug.
	resp.Diagnostics.AddError(
		"Unsupported in-place container update",
		"synology_container has no in-place Update; configuration changes must replace the resource.",
	)
}

func (r *ContainerResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ContainerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()
	// Best-effort stop so delete is not blocked by a running container.
	_, _ = r.client.ContainerStop(ctx, docker.ContainerOperationRequest{Name: name})

	if err := r.client.ContainerDelete(ctx, docker.ContainerDeleteRequest{
		Name:  name,
		Force: true,
	}); err != nil {
		// Already gone is success.
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(err.Error(), "117") {
			return
		}
		// Confirm absence: some DSM builds return a generic 114 on delete even
		// when the container is gone, or refuse delete while still listing it.
		list, lerr := r.client.ContainerList(ctx, docker.ContainerListRequest{Limit: -1})
		if lerr == nil {
			for _, c := range list.Containers {
				if c.Name == name {
					resp.Diagnostics.AddError(
						"Failed to delete container",
						fmt.Sprintf("%v (container %q still present after delete)", err, name),
					)
					return
				}
			}
			return // gone despite the error
		}
		resp.Diagnostics.AddError("Failed to delete container", err.Error())
	}
}

func (r *ContainerResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
