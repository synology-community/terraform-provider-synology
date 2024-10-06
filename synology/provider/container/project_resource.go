package container

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
	"github.com/synology-community/go-synology/pkg/api/docker"
	"github.com/synology-community/go-synology/pkg/api/filestation"
	"github.com/synology-community/go-synology/pkg/util/form"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

type ProjectResource struct {
	client     docker.Api
	fsClient   filestation.Api
	coreClient core.Api
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	SharePath     types.String `tfsdk:"share_path"`
	Services      types.Map    `tfsdk:"services"`
	Networks      types.Map    `tfsdk:"networks"`
	Volumes       types.Map    `tfsdk:"volumes"`
	Secrets       types.Map    `tfsdk:"secrets"`
	Configs       types.Map    `tfsdk:"configs"`
	Extensions    types.Map    `tfsdk:"extensions"`
	Run           types.Bool   `tfsdk:"run"`
	Status        types.String `tfsdk:"status"`
	ServicePortal types.Object `tfsdk:"service_portal"`
	Content       types.String `tfsdk:"content"`
	Metadata      types.Map    `tfsdk:"metadata"`
	// ComposeFiles types.ListType `tfsdk:"compose_files"`
	// Environment  types.MapType  `tfsdk:"environment"`
	CreatedAt timetypes.RFC3339 `tfsdk:"created_at"`
	UpdatedAt timetypes.RFC3339 `tfsdk:"updated_at"`
}

func (p ProjectResourceModel) IsRunning() bool {
	return strings.ToUpper(p.Status.ValueString()) == "RUNNING"
}

func (p ProjectResourceModel) ShouldRun() bool {
	return !p.IsRunning() && p.Run.ValueBool()
}

const projectDescription = `A Docker Compose project for the Container Manager Synology API.

> **Note:** Synology creates a shared folder for each project. The shared folder is created in the ` + "`/projects`" + ` directory by default. The shared folder is named after the project name. The shared folder is used to store the project files and data. The shared folder is mounted to the ` + "`/volume1/projects`" + ` directory on the Synology NAS.

`

func projectExists(err error) bool {
	errs, ok := err.(*multierror.Error)
	if !ok {
		return false
	}

	for _, e := range errs.Errors {
		if e.Error() == "api response error code 2102: Project already exists" {
			return true
		}
	}

	return false
}

func (f *ProjectResource) handleConfigs(ctx context.Context, data ProjectResourceModel) (diags diag.Diagnostics) {
	if data.Configs.IsNull() || data.Configs.IsUnknown() {
		return
	}

	elements := map[string]models.Config{}
	diags = data.Configs.ElementsAs(ctx, &elements, true)
	if diags.HasError() {
		return
	}

	for _, v := range elements {
		if !v.Content.IsNull() || !v.Content.IsUnknown() {
			// Upload the file
			_, err := f.fsClient.Upload(
				ctx,
				data.SharePath.ValueString(),
				form.File{
					Name:    v.File.ValueString(),
					Content: v.Content.ValueString(),
				}, false,
				true)
			if err != nil {
				diags.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
				return
			}
		}
	}

	return
}

func (f *ProjectResource) ensureProjectShare(ctx context.Context, sharePath string) error {
	folderParts := strings.Split(sharePath, "/")
	plen := len(folderParts)

	if plen < 2 {
		return fmt.Errorf("Invalid share path: %s", sharePath)
	}

	share := folderParts[1]

	shares, err := f.coreClient.ShareList(ctx)
	if err != nil {
		return err
	}

	i := slices.IndexFunc(shares.Shares, func(s core.Share) bool {
		return s.Name == share
	})

	if i == -1 {
		volresp, err := f.coreClient.VolumeList(ctx)
		if err != nil {
			return err
		}

		vol := volresp.Volumes[0]

		volPath := vol.VolumePath

		err = f.coreClient.ShareCreate(ctx, core.ShareInfo{
			Name:    share,
			VolPath: volPath,
		})

		if err != nil {
			return err
		}
	}

	folderName := folderParts[plen-1]
	folderPath := strings.Join(folderParts[:plen-1], "/")
	_, err = f.fsClient.Get(ctx, sharePath)
	if err != nil {
		switch err.(type) {
		case filestation.FileNotFoundError:
			_, err = f.fsClient.CreateFolder(ctx, []string{folderPath}, []string{folderName}, true)
			if err != nil {
				return err
			}
		default:
			return err
		}
	}

	return nil
}

// Create implements resource.Resource.
func (f *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectYAML := ""

	if data.SharePath.IsNull() || data.SharePath.IsUnknown() {
		data.SharePath = types.StringValue(fmt.Sprintf("/projects/%s", data.Name.ValueString()))
	}

	if data.Metadata.IsNull() || data.Metadata.IsUnknown() {
		data.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	err := f.ensureProjectShare(ctx, data.SharePath.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to create project share", err.Error())
		return
	}

	if !data.Configs.IsNull() && !data.Configs.IsUnknown() {
		resp.Diagnostics.Append(f.handleConfigs(ctx, data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	servicePortal := models.ServicePortal{}

	if !data.ServicePortal.IsNull() && !data.ServicePortal.IsUnknown() {
		resp.Diagnostics.Append(data.ServicePortal.As(ctx, &servicePortal, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	shouldUpdate := false

	res, err := f.client.ProjectCreate(ctx, docker.ProjectCreateRequest{
		Name:                  data.Name.ValueString(),
		Content:               projectYAML,
		SharePath:             data.SharePath.ValueString(),
		EnableServicePortal:   servicePortal.Enable.ValueBoolPointer(),
		ServicePortalName:     servicePortal.Name.ValueString(),
		ServicePortalPort:     servicePortal.Port.ValueInt64Pointer(),
		ServicePortalProtocol: servicePortal.Protocol.ValueString(),
	})

	if err != nil {
		if projectExists(err) {
			shouldUpdate = true
		} else {
			resp.Diagnostics.AddError("Failed to create project", err.Error())
			return
		}
	} else {
		data.CreatedAt = timetypes.NewRFC3339TimeValue(res.CreatedAt)
		data.UpdatedAt = timetypes.NewRFC3339TimeValue(res.UpdatedAt)
	}

	if shouldUpdate {
		p, err := f.client.ProjectGetByName(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get project on create", err.Error())
			return
		}

		data.ID = types.StringValue(p.ID)
		data.Status = types.StringValue(p.Status)
		data.CreatedAt = timetypes.NewRFC3339TimeValue(p.CreatedAt)
		data.UpdatedAt = timetypes.NewRFC3339TimeValue(p.UpdatedAt)

		if data.Status.ValueString() == "RUNNING" {
			_, err = f.client.ProjectStopStream(ctx, docker.ProjectStreamRequest{
				ID: data.ID.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to stop project", err.Error())
				return
			}
			_, err = f.client.ProjectCleanStream(ctx, docker.ProjectStreamRequest{
				ID: data.ID.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to clean project", err.Error())
				return
			}
			if !data.Run.IsNull() && !data.Run.IsUnknown() && data.Run.ValueBool() {
				_, err = f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
					ID: data.ID.ValueString(),
				})
				if err != nil {
					resp.Diagnostics.AddError("Failed to build project", err.Error())
					return
				}
			}
		}

		_, err = f.client.ProjectUpdate(ctx, docker.ProjectUpdateRequest{
			ID:                    data.ID.ValueString(),
			Content:               projectYAML,
			EnableServicePortal:   servicePortal.Enable.ValueBoolPointer(),
			ServicePortalName:     servicePortal.Name.ValueString(),
			ServicePortalPort:     servicePortal.Port.ValueInt64Pointer(),
			ServicePortalProtocol: servicePortal.Protocol.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to update project", err.Error())
			return
		}

	} else {
		data.ID = types.StringValue(res.ID)
	}

	if !data.Run.IsNull() && !data.Run.IsUnknown() && data.Run.ValueBool() {
		_, err = f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to build project after update", err.Error())
			return
		}
	}

	proj, err := f.client.ProjectGet(ctx, data.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project after create", err.Error())
		return
	}

	data.Status = types.StringValue(proj.Status)
	data.UpdatedAt = timetypes.NewRFC3339TimeValue(proj.UpdatedAt)

	data.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})

	// data.Content = types.StringValue(proj.Content)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	proj, err := f.client.ProjectGet(ctx, data.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project before deletion", err.Error())
		return
	}

	if proj.Status == "RUNNING" {
		_, err = f.client.ProjectStopStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to stop project during deletion", err.Error())
			return
		}
	}

	_, err = f.client.ProjectCleanStream(ctx, docker.ProjectStreamRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to clean project for deletion", err.Error())
		return
	}

	_, err = f.client.ProjectDelete(ctx, docker.ProjectDeleteRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to delete project", err.Error())
		return
	}

	// Remove data from Terraform state
	resp.State.RemoveResource(ctx)
}

// Read implements resource.Resource.
func (f *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proj, err := f.client.ProjectGet(ctx, data.ID.ValueString())

	if err != nil {
		if proj, err = f.client.ProjectGetByName(ctx, data.Name.ValueString()); err != nil {
			switch err.(type) {
			case docker.ProjectNotFoundError:
				resp.State.RemoveResource(ctx)
				return
			default:
				resp.Diagnostics.AddError("Failed to get project on read", err.Error())
				return
			}
		} else if data.ID.IsNull() || data.ID.IsUnknown() || data.ID.ValueString() != proj.ID {
			if proj.ID != "" {
				data.ID = types.StringValue(proj.ID)
			}
		}
	}

	data.Status = types.StringValue(proj.Status)
	data.CreatedAt = timetypes.NewRFC3339TimeValue(proj.CreatedAt)
	data.UpdatedAt = timetypes.NewRFC3339TimeValue(proj.UpdatedAt)
	data.Content = types.StringValue(proj.Content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements resource.Resource.
func (f *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel

	if plan.Metadata.IsNull() || plan.Metadata.IsUnknown() {
		plan.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	if state.Metadata.IsNull() || state.Metadata.IsUnknown() {
		state.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var servicesChanged, configChanged bool

	if !reflect.DeepEqual(plan.Services, state.Services) {
		servicesChanged = true
	}

	if !reflect.DeepEqual(plan.Configs, state.Configs) {
		configChanged = true
	}

	if !servicesChanged && !configChanged {
		tflog.Info(ctx, "No changes detected in services or configs, skipping update")
		return
	}

	servicePortal := models.ServicePortal{}

	if !plan.ServicePortal.IsNull() && !plan.ServicePortal.IsUnknown() {
		resp.Diagnostics.Append(plan.ServicePortal.As(ctx, &servicePortal, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if configChanged {
		f.handleConfigs(ctx, plan)
	}

	content := plan.Content.ValueString()

	if servicesChanged {

		proj, err := f.client.ProjectGet(ctx, plan.ID.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Failed to get project on update", err.Error())
			return
		}

		if proj.Content == content {
			tflog.Info(ctx, "No changes detected in project, skipping update")
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status"), types.StringValue(proj.Status))...)
			return

		}

		if proj.IsRunning() {
			_, err = f.client.ProjectStopStream(ctx, docker.ProjectStreamRequest{
				ID: plan.ID.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to stop project", err.Error())
				return
			}
		}

		_, err = f.client.ProjectCleanStream(ctx, docker.ProjectStreamRequest{
			ID: plan.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to clean project", err.Error())
			return
		}

		_, err = f.client.ProjectUpdate(ctx, docker.ProjectUpdateRequest{
			ID:                    plan.ID.ValueString(),
			Content:               content,
			EnableServicePortal:   servicePortal.Enable.ValueBoolPointer(),
			ServicePortalName:     servicePortal.Name.ValueString(),
			ServicePortalPort:     servicePortal.Port.ValueInt64Pointer(),
			ServicePortalProtocol: servicePortal.Protocol.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to update project", err.Error())
			return
		}
	}

	if !plan.Run.IsNull() && !plan.Run.IsUnknown() && plan.Run.ValueBool() {
		_, err := f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
			ID: plan.ID.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to build project", err.Error())
			return
		}
	}

	proj, err := f.client.ProjectGet(ctx, plan.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project after update", err.Error())
		return
	}

	plan.Status = types.StringValue(proj.Status)
	plan.CreatedAt = timetypes.NewRFC3339TimeValue(proj.CreatedAt)
	plan.UpdatedAt = timetypes.NewRFC3339TimeValue(proj.UpdatedAt)

	plan.Metadata = types.MapValueMust(types.StringType, map[string]attr.Value{})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Metadata implements resource.Resource.
func (f *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "project")
}

// Schema implements resource.Resource.
func (f *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: projectDescription,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The content of the project.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					UseArgumentsForUnknownContent(),
				},
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "The metadata of the project.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"share_path": schema.StringAttribute{
				MarkdownDescription: "The share path of the project.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					UseDefaultSharePath(),
				},
				Computed: true,
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Whether to run the project.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					UseRunningStatus(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time the project was created.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The time the project was updated.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"service_portal": schema.SingleNestedAttribute{
				MarkdownDescription: "Synology Web Station configuration for the docker compose project.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enable": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable the service portal.",
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The name of the service portal.",
						Optional:            true,
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "The port of the service portal.",
						Optional:            true,
					},
					"protocol": schema.StringAttribute{
						MarkdownDescription: "The protocol of the service portal.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("http", "https"),
						},
					},
				},
			},
			"services": schema.MapNestedAttribute{
				MarkdownDescription: "Docker compose services.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"container_name": schema.StringAttribute{
							MarkdownDescription: "The container name.",
							Optional:            true,
						},
						"replicas": schema.Int64Attribute{
							MarkdownDescription: "The number of replicas.",
							Optional:            true,
						},
						"mem_limit": schema.StringAttribute{
							MarkdownDescription: "The memory limit.",
							Optional:            true,
						},
						"entrypoint": schema.ListAttribute{
							MarkdownDescription: "The entrypoint of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"command": schema.ListAttribute{
							MarkdownDescription: "The command of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"user": schema.StringAttribute{
							MarkdownDescription: "The user of the service.",
							Optional:            true,
						},
						"restart": schema.StringAttribute{
							MarkdownDescription: "The restart policy.",
							Optional:            true,
						},
						"network_mode": schema.StringAttribute{
							MarkdownDescription: "The network mode.",
							Optional:            true,
						},
						"privileged": schema.BoolAttribute{
							MarkdownDescription: "Whether the service is privileged.",
							Optional:            true,
						},
						"tmpfs": schema.ListAttribute{
							MarkdownDescription: "The tmpfs of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"security_opt": schema.ListAttribute{
							MarkdownDescription: "The security options of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"environment": schema.MapAttribute{
							MarkdownDescription: "The environment of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"labels": schema.MapAttribute{
							MarkdownDescription: "The labels of the network.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"dns": schema.ListAttribute{
							MarkdownDescription: "The DNS of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"capabilities": schema.SingleNestedAttribute{
							MarkdownDescription: "The capabilities of the service.",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"add": schema.ListAttribute{
									MarkdownDescription: "The capabilities to add.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"drop": schema.ListAttribute{
									MarkdownDescription: "The capabilities to drop.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
						"cap_add": schema.ListAttribute{
							MarkdownDescription: "The capabilities to add.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"cap_drop": schema.ListAttribute{
							MarkdownDescription: "The capabilities to drop.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"sysctls": schema.MapAttribute{
							MarkdownDescription: "The sysctls of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"image": schema.StringAttribute{
							MarkdownDescription: "The image of the service.",
							Optional:            true,
						},
						"ports": schema.ListNestedAttribute{
							MarkdownDescription: "The ports of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the port.",
										Optional:            true,
									},
									"target": schema.Int64Attribute{
										MarkdownDescription: "The target of the port.",
										Optional:            true,
									},
									"published": schema.StringAttribute{
										MarkdownDescription: "The published of the port.",
										Optional:            true,
									},
									"protocol": schema.StringAttribute{
										MarkdownDescription: "The protocol of the port.",
										Optional:            true,
									},
									"app_protocol": schema.StringAttribute{
										MarkdownDescription: "The app protocol of the port.",
										Optional:            true,
									},
									"mode": schema.StringAttribute{
										MarkdownDescription: "The mode of the port.",
										Optional:            true,
									},
									"host_ip": schema.StringAttribute{
										MarkdownDescription: "The host IP of the port.",
										Optional:            true,
									},
								},
							},
						},
						"depends_on": schema.MapNestedAttribute{
							MarkdownDescription: "The dependencies of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"condition": schema.StringAttribute{
										MarkdownDescription: "The condition of the dependency.",
										Optional:            true,
									},
									"restart": schema.BoolAttribute{
										MarkdownDescription: "Whether to restart.",
										Optional:            true,
									},
								},
							},
						},
						"networks": schema.MapNestedAttribute{
							MarkdownDescription: "The networks of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the network.",
										Optional:            true,
									},
									"aliases": schema.SetAttribute{
										MarkdownDescription: "The aliases of the network.",
										Optional:            true,
										ElementType:         types.StringType,
									},
									"ipv4_address": schema.StringAttribute{
										MarkdownDescription: "The IPv4 address of the network.",
										Optional:            true,
									},
									"ipv6_address": schema.StringAttribute{
										MarkdownDescription: "The IPv6 address of the network.",
										Optional:            true,
									},
									"link_local_ips": schema.SetAttribute{
										MarkdownDescription: "The link local IPs of the network.",
										Optional:            true,
										ElementType:         types.StringType,
									},
									"mac_address": schema.StringAttribute{
										MarkdownDescription: "The MAC address of the network.",
										Optional:            true,
									},
									"driver_opts": schema.MapAttribute{
										MarkdownDescription: "The driver options of the network.",
										Optional:            true,
										ElementType:         types.StringType,
									},
									"priority": schema.Int64Attribute{
										MarkdownDescription: "The priority of the network.",
										Optional:            true,
									},
								},
							},
						},
						"logging": schema.SingleNestedAttribute{
							MarkdownDescription: "Logging configuration for the docker service.",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"driver": schema.StringAttribute{
									MarkdownDescription: "The driver of the logging.",
									Optional:            true,
								},
								"options": schema.MapAttribute{
									MarkdownDescription: "The options of the logging.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
						"healthcheck": schema.SingleNestedAttribute{
							MarkdownDescription: "Health check configuration.",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"test": schema.ListAttribute{
									MarkdownDescription: "Test command to run.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"interval": schema.StringAttribute{
									MarkdownDescription: "Interval to run the test.",
									Optional:            true,
									CustomType:          timetypes.GoDurationType{},
								},
								"timeout": schema.StringAttribute{
									MarkdownDescription: "Timeout to run the test.",
									Optional:            true,
									CustomType:          timetypes.GoDurationType{},
								},
								"retries": schema.NumberAttribute{
									MarkdownDescription: "Number of retries.",
									Optional:            true,
								},
								"start_period": schema.StringAttribute{
									MarkdownDescription: "Start period.",
									Optional:            true,
									CustomType:          timetypes.GoDurationType{},
								},
								"start_interval": schema.StringAttribute{
									MarkdownDescription: "Start interval.",
									Optional:            true,
									CustomType:          timetypes.GoDurationType{},
								},
							},
						},
						"volumes": schema.ListNestedAttribute{
							MarkdownDescription: "The volumes of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										MarkdownDescription: "The source of the volume.",
										Optional:            true,
									},
									"target": schema.StringAttribute{
										MarkdownDescription: "The target of the volume.",
										Optional:            true,
									},
									"read_only": schema.BoolAttribute{
										MarkdownDescription: "Whether the volume is read only.",
										Optional:            true,
									},
									"type": schema.StringAttribute{
										MarkdownDescription: "The type of the volume.",
										Required:            true,
									},
									"bind": schema.SingleNestedAttribute{
										MarkdownDescription: "The bind of the volume.",
										Optional:            true,
										Attributes: map[string]schema.Attribute{
											"propagation": schema.StringAttribute{
												MarkdownDescription: "The propagation of the bind.",
												Optional:            true,
											},
											"create_host_path": schema.BoolAttribute{
												MarkdownDescription: "Whether to create the host path.",
												Optional:            true,
											},
											"selinux": schema.StringAttribute{
												MarkdownDescription: "The selinux of the bind.",
												Optional:            true,
											},
										},
									},
								},
							},
						},
						"ulimits": schema.MapNestedAttribute{
							MarkdownDescription: "The ulimits of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the ulimit.",
										Required:            true,
									},
									"value": schema.Int64Attribute{
										MarkdownDescription: "The value of the ulimit.",
										Optional:            true,
									},
									"soft": schema.Int64Attribute{
										MarkdownDescription: "The soft of the ulimit.",
										Optional:            true,
									},
									"hard": schema.Int64Attribute{
										MarkdownDescription: "The hard of the ulimit.",
										Optional:            true,
									},
								},
							},
						},
						"configs": schema.ListNestedAttribute{
							MarkdownDescription: "The configs of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										MarkdownDescription: "The source of the config.",
										Optional:            true,
									},
									"target": schema.StringAttribute{
										MarkdownDescription: "The target of the config.",
										Optional:            true,
									},
									"uid": schema.StringAttribute{
										MarkdownDescription: "The UID of the config.",
										Optional:            true,
									},
									"gid": schema.StringAttribute{
										MarkdownDescription: "The GID of the config.",
										Optional:            true,
									},
									"mode": schema.StringAttribute{
										MarkdownDescription: "The mode of the config.",
										Optional:            true,
									},
								},
							},
						},
						"secrets": schema.ListNestedAttribute{
							MarkdownDescription: "The secrets of the service.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										MarkdownDescription: "The source of the config.",
										Optional:            true,
									},
									"target": schema.StringAttribute{
										MarkdownDescription: "The target of the config.",
										Optional:            true,
									},
									"uid": schema.StringAttribute{
										MarkdownDescription: "The UID of the config.",
										Optional:            true,
									},
									"gid": schema.StringAttribute{
										MarkdownDescription: "The GID of the config.",
										Optional:            true,
									},
									"mode": schema.StringAttribute{
										MarkdownDescription: "The mode of the config.",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},
			"networks": schema.MapNestedAttribute{
				MarkdownDescription: "Docker compose networks.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the network.",
							Optional:            true,
						},
						"driver": schema.StringAttribute{
							MarkdownDescription: "The driver of the network.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("bridge", "host", "overlay", "macvlan", "none", "ipvlan"),
							},
						},
						"driver_opts": schema.MapAttribute{
							MarkdownDescription: "The driver options of the network.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"external": schema.BoolAttribute{
							MarkdownDescription: "Whether the network is external.",
							Optional:            true,
						},
						"internal": schema.BoolAttribute{
							MarkdownDescription: "Whether the network is internal.",
							Optional:            true,
						},
						"attachable": schema.BoolAttribute{
							MarkdownDescription: "Whether the network is attachable.",
							Optional:            true,
						},
						"labels": schema.MapAttribute{
							MarkdownDescription: "The labels of the network.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"enable_ipv6": schema.BoolAttribute{
							MarkdownDescription: "Whether to enable IPv6.",
							Optional:            true,
						},
						"ipam": schema.SingleNestedAttribute{
							MarkdownDescription: "The IPAM of the network.",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"driver": schema.StringAttribute{
									MarkdownDescription: "Custom IPAM driver, instead of the default.",
									Optional:            true,
								},
								"config": schema.ListNestedAttribute{
									MarkdownDescription: "The config of the IPAM.",
									Optional:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"subnet": schema.StringAttribute{
												MarkdownDescription: "The subnet of the config.",
												Optional:            true,
											},
											"ip_range": schema.StringAttribute{
												MarkdownDescription: "The IP range of the config.",
												Optional:            true,
											},
											"gateway": schema.StringAttribute{
												MarkdownDescription: "The gateway of the config.",
												Optional:            true,
											},
											"aux_addresses": schema.MapAttribute{
												MarkdownDescription: "The aux addresses of the config.",
												Optional:            true,
												ElementType:         types.StringType,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"volumes": schema.MapNestedAttribute{
				MarkdownDescription: "Docker compose volumes.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the volume.",
							Optional:            true,
						},
						"driver": schema.StringAttribute{
							MarkdownDescription: "The driver of the volume.",
							Optional:            true,
						},
						"driver_opts": schema.MapAttribute{
							MarkdownDescription: "The driver options of the volume.",
							Optional:            true,
							ElementType:         types.StringType,
						},
						"external": schema.BoolAttribute{
							MarkdownDescription: "Whether the volume is external.",
							Optional:            true,
						},
						"labels": schema.MapAttribute{
							MarkdownDescription: "The labels of the volume.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"secrets": schema.MapNestedAttribute{
				MarkdownDescription: "Docker compose secrets.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the secret.",
							Optional:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "The content of the config.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"file": schema.StringAttribute{
							MarkdownDescription: "The file of the config.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"configs": schema.MapNestedAttribute{
				MarkdownDescription: "Docker compose configs.",
				Optional:            true,
				PlanModifiers: []planmodifier.Map{
					SetConfigPathsFromContent(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the config.",
							Required:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "The content of the config.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"file": schema.StringAttribute{
							MarkdownDescription: "The file of the config.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"extensions": schema.MapNestedAttribute{
				MarkdownDescription: "Docker compose extensions.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the extension.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (f *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Api)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	f.client = client.DockerAPI()
	f.fsClient = client.FileStationAPI()
	f.coreClient = client.CoreAPI()
}

func (f *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	res, err := f.client.ProjectGetByName(ctx, req.ID)

	if err != nil {
		resp.Diagnostics.AddError("Failed to list package feeds", err.Error())
		return
	}

	// compProj := &composetypes.Project{
	// 	Name: res.Name,
	// }

	// services := map[string]models.Service{}
	// svcs := basetypes.NewMapNull(models.Service{}.ModelType())

	// if res.Content != "" {
	// 	model, err := loader.ParseYAML([]byte(res.Content))
	// 	if err == nil {
	// 		err := loader.Transform(model, compProj)
	// 		if err != nil {
	// 			resp.Diagnostics.AddError("Failed to transform project", err.Error())
	// 			return
	// 		}
	// 	}
	// }

	// if compProj.Services != nil {
	// 	for k, svc := range compProj.Services {
	// 		nSvc := models.Service{}
	// 		diags := nSvc.FromComposeConfig(ctx, &svc)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 			return
	// 		} else {
	// 			services[k] = nSvc
	// 		}
	// 	}

	// 	svcValues, diags := types.MapValueFrom(ctx, models.Service{}.ModelType(), services)
	// 	if diags.HasError() {
	// 		resp.Diagnostics.Append(diags...)
	// 	} else {
	// 		svcs = svcValues
	// 	}
	// }

	// volumes := map[string]models.Volume{}
	// vols := basetypes.NewMapNull(models.Volume{}.ModelType())

	// if compProj.Volumes != nil {
	// 	for k, vol := range compProj.Volumes {
	// 		nVol := models.Volume{}
	// 		diags := nVol.FromComposeConfig(ctx, &vol)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 			return
	// 		} else {
	// 			volumes[k] = nVol
	// 		}
	// 	}

	// 	volValues, diags := types.MapValueFrom(ctx, models.Volume{}.ModelType(), volumes)
	// 	if diags.HasError() {
	// 		resp.Diagnostics.Append(diags...)
	// 	} else {
	// 		vols = volValues
	// 	}
	// }

	// secrets := map[string]models.Secret{}
	// secretsMap := basetypes.NewMapNull(models.Secret{}.ModelType())

	// if compProj.Secrets != nil {
	// 	for k, sec := range compProj.Secrets {
	// 		nSec := models.Secret{}
	// 		diags := nSec.FromComposeConfig(ctx, &sec)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 			return
	// 		} else {
	// 			secrets[k] = nSec
	// 		}

	// 		secValues, diags := types.MapValueFrom(ctx, models.Secret{}.ModelType(), secrets)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 		} else {
	// 			secretsMap = secValues
	// 		}
	// 	}
	// }

	// networks := map[string]models.Network{}
	// nets := basetypes.NewMapNull(models.Network{}.ModelType())

	// if compProj.Networks != nil {
	// 	for k, net := range compProj.Networks {
	// 		nNet := models.Network{}
	// 		diags := nNet.FromComposeConfig(ctx, &net)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 			return
	// 		} else {
	// 			networks[k] = nNet
	// 		}
	// 	}

	// 	netValues, diags := types.MapValueFrom(ctx, models.Network{}.ModelType(), networks)
	// 	if diags.HasError() {
	// 		resp.Diagnostics.Append(diags...)
	// 	} else {
	// 		nets = netValues
	// 	}
	// }

	// configs := map[string]models.Config{}
	// configsMap := basetypes.NewMapNull(models.Config{}.ModelType())

	// if compProj.Configs != nil {

	// 	for k, cfg := range compProj.Configs {
	// 		nCfg := models.Config{}
	// 		diags := nCfg.FromComposeConfig(ctx, &cfg)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 			return
	// 		} else {
	// 			configs[k] = nCfg
	// 		}

	// 		cfgValues, diags := types.MapValueFrom(ctx, models.Config{}.ModelType(), configs)
	// 		if diags.HasError() {
	// 			resp.Diagnostics.Append(diags...)
	// 		} else {
	// 			configsMap = cfgValues
	// 		}
	// 	}
	// }

	servicePortalType := map[string]attr.Type{
		"enable":   types.BoolType,
		"name":     types.StringType,
		"port":     types.Int64Type,
		"protocol": types.StringType,
	}

	servicePortalValues := types.ObjectNull(servicePortalType)

	if res.EnableServicePortal || res.ServicePortalName != "" || res.ServicePortalPort != 0 || res.ServicePortalProtocol != "" {
		servicePortal := models.ServicePortal{
			Enable:   types.BoolValue(res.EnableServicePortal),
			Name:     types.StringValue(res.ServicePortalName),
			Port:     types.Int64Value(int64(res.ServicePortalPort)),
			Protocol: types.StringValue(res.ServicePortalProtocol),
		}

		svcPortalValues, diags := types.ObjectValueFrom(ctx, servicePortalType, servicePortal)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
		} else {
			servicePortalValues = svcPortalValues
		}
	}

	project := ProjectResourceModel{
		ID:        types.StringValue(res.ID),
		Content:   types.StringValue(res.Content),
		Status:    types.StringValue(res.Status),
		CreatedAt: timetypes.NewRFC3339TimeValue(res.CreatedAt),
		UpdatedAt: timetypes.NewRFC3339TimeValue(res.UpdatedAt),
		Name:      types.StringValue(res.Name),
		SharePath: types.StringValue(res.SharePath),
		Services:  types.MapValueMust(models.Service{}.ModelType(), map[string]attr.Value{}),
		Volumes:   types.MapValueMust(models.Volume{}.ModelType(), map[string]attr.Value{}),
		Secrets:   types.MapValueMust(models.Secret{}.ModelType(), map[string]attr.Value{}),
		Configs:   types.MapValueMust(models.Config{}.ModelType(), map[string]attr.Value{}),
		Networks:  types.MapValueMust(models.Network{}.ModelType(), map[string]attr.Value{}),
		// Secrets:   types.MapNull(models.Secret{}.ModelType()),
		// Configs:   types.MapNull(models.Config{}.ModelType()),
		// Networks:  types.MapNull(models.Network{}.ModelType()),
		Extensions: types.MapNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name": types.StringType,
			},
		}),
		Run:           types.BoolValue(false),
		Metadata:      types.MapNull(types.StringType),
		ServicePortal: servicePortalValues,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, project)...)
}
