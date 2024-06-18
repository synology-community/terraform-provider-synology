package container

import (
	"context"
	"fmt"

	"github.com/appkins/terraform-provider-synology/synology/provider/container/models"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/docker"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

type ProjectResource struct {
	client docker.Api
}

type ServicePortalModel struct {
	Enable   types.Bool   `tfsdk:"enable"`
	Name     types.String `tfsdk:"name"`
	Port     types.Int64  `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	SharePath     types.String `tfsdk:"share_path"`
	Services      types.Set    `tfsdk:"service"`
	Networks      types.Set    `tfsdk:"network"`
	Volumes       types.Set    `tfsdk:"volume"`
	Secrets       types.Set    `tfsdk:"secret"`
	Configs       types.Set    `tfsdk:"config"`
	Extensions    types.Set    `tfsdk:"extension"`
	Build         types.Bool   `tfsdk:"build"`
	State         types.String `tfsdk:"state"`
	ServicePortal types.Set    `tfsdk:"service_portal"`
	// ComposeFiles types.ListType `tfsdk:"compose_files"`
	// Environment  types.MapType  `tfsdk:"environment"`
}

func getProjectYaml(ctx context.Context, data ProjectResourceModel) (string, error) {
	project := composetypes.Project{}

	if !data.Services.IsNull() && !data.Services.IsUnknown() {

		elements := []models.Service{}
		diags := data.Services.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			return "", fmt.Errorf("Failed to read services")
		}

		project.Services = map[string]composetypes.ServiceConfig{}

		for _, v := range elements {

			service := v.AsComposeServiceConfig()

			project.Services[service.Name] = service
		}
	}

	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {

		elements := []models.Network{}
		diags := data.Networks.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			return "", fmt.Errorf("Failed to read networks")
		}

		project.Networks = map[string]composetypes.NetworkConfig{}

		for _, v := range elements {
			n := composetypes.NetworkConfig{}

			diags := v.AsComposeConfig(ctx, &n)
			if diags.HasError() {
				return "", fmt.Errorf("Failed to read networks")
			}

			project.Networks[n.Name] = n
		}
	}

	if !data.Volumes.IsNull() && !data.Volumes.IsUnknown() {

		elements := []models.Volume{}
		diags := data.Volumes.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			return "", fmt.Errorf("Failed to read volumes")
		}

		project.Volumes = map[string]composetypes.VolumeConfig{}

		for _, v := range elements {
			n := composetypes.VolumeConfig{}

			diags := v.AsComposeConfig(ctx, &n)
			if diags.HasError() {
				return "", fmt.Errorf("Failed to read volumes")
			}

			project.Volumes[n.Name] = n
		}
	}

	if !data.Configs.IsNull() && !data.Configs.IsUnknown() {

		elements := []models.Config{}
		diags := data.Configs.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			return "", fmt.Errorf("Failed to read configs")
		}

		project.Configs = map[string]composetypes.ConfigObjConfig{}

		for _, v := range elements {
			n := composetypes.ConfigObjConfig{}

			diags := v.AsComposeConfig(ctx, &n)
			if diags.HasError() {
				return "", fmt.Errorf("Failed to read configs")
			}

			project.Configs[n.Name] = n
		}
	}

	projectYAML, err := project.MarshalYAML()
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal docker-compose.yml")
	}

	return string(projectYAML), nil
}

// Create implements resource.Resource.
func (f *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectYAML, err := getProjectYaml(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal docker-compose.yml", err.Error())
		return
	}

	enableServicePortal := new(bool)
	servicePortalName := ""
	servicePortalPort := new(int64)
	servicePortalProtocol := ""

	if !data.ServicePortal.IsNull() && !data.ServicePortal.IsUnknown() {
		elements := []ServicePortalModel{}
		resp.Diagnostics.Append(data.ServicePortal.ElementsAs(ctx, &elements, true)...)

		if resp.Diagnostics.HasError() {
			return
		}

		if len(elements) > 0 {
			enableServicePortal = elements[0].Enable.ValueBoolPointer()
			servicePortalName = elements[0].Name.ValueString()
			servicePortalPort = elements[0].Port.ValueInt64Pointer()
			servicePortalProtocol = elements[0].Protocol.ValueString()
		}
	}

	var sharePath string

	if !data.SharePath.IsNull() && !data.SharePath.IsUnknown() {
		sharePath = data.SharePath.ValueString()
	} else {
		sharePath = fmt.Sprintf("/projects/%s", data.Name.ValueString())
	}

	shouldUpdate := false

	res, err := f.client.ProjectCreate(ctx, docker.ProjectCreateRequest{
		Name:                  data.Name.ValueString(),
		Content:               projectYAML,
		SharePath:             sharePath,
		EnableServicePortal:   enableServicePortal,
		ServicePortalName:     servicePortalName,
		ServicePortalPort:     servicePortalPort,
		ServicePortalProtocol: servicePortalProtocol,
	})

	if err != nil {
		errs, ok := err.(*multierror.Error)
		if !ok {
			resp.Diagnostics.AddError("Failed to create project", err.Error())
			return
		}

		if errs.Errors[0].Error() == "api response error code 2102: Project already exists" {
			shouldUpdate = true
		} else {
			for _, e := range errs.Errors {
				resp.Diagnostics.AddError("Failed to create project", e.Error())
			}
			return
		}
	}

	if shouldUpdate {
		status := ""

		listResult, err := f.client.ProjectList(ctx, docker.ProjectListRequest{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to list projects", err.Error())
			return
		}

		for k, p := range *listResult {
			if p.Name == data.Name.ValueString() {
				status = p.Status
				data.ID = types.StringValue(k)
				break
			}
		}

		if status == "RUNNING" {
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
		}

		_, err = f.client.ProjectUpdate(ctx, docker.ProjectUpdateRequest{
			ID:                    data.ID.ValueString(),
			Content:               projectYAML,
			EnableServicePortal:   enableServicePortal,
			ServicePortalName:     servicePortalName,
			ServicePortalPort:     servicePortalPort,
			ServicePortalProtocol: servicePortalProtocol,
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to update project", err.Error())
			return
		}

	} else {
		data.ID = types.StringValue(res.ID)
	}

	if !data.Build.IsNull() && !data.Build.IsUnknown() && data.Build.ValueBool() {
		_, err = f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to build project", err.Error())
			return
		}
	}

	proj, err := f.client.ProjectGet(ctx, docker.ProjectGetRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project", err.Error())
		return
	}

	data.State = types.StringValue(proj.Status)

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

	proj, err := f.client.ProjectGet(ctx, docker.ProjectGetRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project", err.Error())
		return
	}

	if proj.Status == "RUNNING" {
		_, err = f.client.ProjectStopStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to stop project", err.Error())
			return
		}
	}

	_, err = f.client.ProjectCleanStream(ctx, docker.ProjectStreamRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to clean project", err.Error())
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

	proj, err := f.client.ProjectGet(ctx, docker.ProjectGetRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		emessage := err.Error()
		projects, err := f.client.ProjectList(ctx, docker.ProjectListRequest{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to list projects", err.Error())
			return
		}
		found := false
		for k, p := range *projects {
			if p.Name == data.Name.ValueString() {
				data.ID = types.StringValue(k)
				found = true
			}
		}
		if !found {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get project", emessage)
		return
	}

	data.State = types.StringValue(proj.Status)

	if proj.Status != "RUNNING" && data.Build.ValueBool() {
		_, err = f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to build project", err.Error())
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements resource.Resource.
func (f *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enableServicePortal := new(bool)
	servicePortalName := ""
	servicePortalPort := new(int64)
	servicePortalProtocol := ""

	if !data.ServicePortal.IsNull() && !data.ServicePortal.IsUnknown() {
		elements := []ServicePortalModel{}
		resp.Diagnostics.Append(data.ServicePortal.ElementsAs(ctx, &elements, true)...)

		if resp.Diagnostics.HasError() {
			return
		}

		if len(elements) > 0 {
			enableServicePortal = elements[0].Enable.ValueBoolPointer()
			servicePortalName = elements[0].Name.ValueString()
			servicePortalPort = elements[0].Port.ValueInt64Pointer()
			servicePortalProtocol = elements[0].Protocol.ValueString()
		}
	}

	projectYAML, err := getProjectYaml(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal docker-compose.yml", err.Error())
		return
	}

	proj, err := f.client.ProjectGet(ctx, docker.ProjectGetRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project", err.Error())
		return
	}

	if proj.Content == projectYAML {
		tflog.Info(ctx, "No changes detected in project, skipping update")
		data.State = types.StringValue(proj.Status)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return

	}

	if proj.Status == "RUNNING" {
		_, err = f.client.ProjectStopStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to stop project", err.Error())
			return
		}
	}

	_, err = f.client.ProjectCleanStream(ctx, docker.ProjectStreamRequest{
		ID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to clean project", err.Error())
		return
	}

	_, err = f.client.ProjectUpdate(ctx, docker.ProjectUpdateRequest{
		ID:                    data.ID.ValueString(),
		Content:               projectYAML,
		EnableServicePortal:   enableServicePortal,
		ServicePortalName:     servicePortalName,
		ServicePortalPort:     servicePortalPort,
		ServicePortalProtocol: servicePortalProtocol,
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to update project", err.Error())
		return
	}

	if !data.Build.IsNull() && !data.Build.IsUnknown() && data.Build.ValueBool() {
		_, err = f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
			ID: data.ID.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Failed to build project", err.Error())
			return
		}
	}

	proj, err = f.client.ProjectGet(ctx, docker.ProjectGetRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to get project", err.Error())
		return
	}

	data.State = types.StringValue(proj.Status)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "project")
}

// Schema implements resource.Resource.
func (f *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Docker Compose project for the Container Manager Synology API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the guest.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"share_path": schema.StringAttribute{
				MarkdownDescription: "The share path of the project.",
				Required:            true,
			},
			"build": schema.BoolAttribute{
				MarkdownDescription: "Whether to build the project.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the project.",
				Computed:            true,
			},
			// "compose_files": schema.ListAttribute{
			// 	MarkdownDescription: "The list of compose files.",
			// 	ElementType:         types.StringType,
			// 	Optional:            true,
			// },
		},
		Blocks: map[string]schema.Block{
			"service_portal": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
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
			},
			"service": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the service.",
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
						"command": schema.ListAttribute{
							MarkdownDescription: "The command of the service.",
							Optional:            true,
							ElementType:         types.StringType,
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
						"environment": schema.MapAttribute{
							MarkdownDescription: "The environment of the service.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"image": schema.SetNestedBlock{
							MarkdownDescription: "The image of the service.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the image.",
										Required:            true,
									},
									"repository": schema.StringAttribute{
										MarkdownDescription: "The repository of the image. Default is `docker.io`.",
										Optional:            true,
									},
									"tag": schema.StringAttribute{
										MarkdownDescription: "The tag of the image. Default is `latest`.",
										Optional:            true,
									},
								},
							},
							Validators: []validator.Set{
								setvalidator.SizeBetween(1, 1),
							},
						},
						"port": schema.SetNestedBlock{
							MarkdownDescription: "The ports of the service.",
							NestedObject: schema.NestedBlockObject{
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
						"network": schema.SetNestedBlock{
							MarkdownDescription: "The networks of the service.",
							NestedObject: schema.NestedBlockObject{
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
						"logging": schema.SetNestedBlock{
							MarkdownDescription: "Logging configuration for the docker service.",
							NestedObject: schema.NestedBlockObject{
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
						},
						"health_check": schema.SetNestedBlock{
							MarkdownDescription: "Health check configuration.",
							NestedObject: schema.NestedBlockObject{
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
						},
						"volume": schema.SetNestedBlock{
							MarkdownDescription: "The volumes of the service.",
							NestedObject: schema.NestedBlockObject{
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
								},
								Blocks: map[string]schema.Block{
									"bind": schema.SetNestedBlock{
										MarkdownDescription: "The bind of the volume.",
										NestedObject: schema.NestedBlockObject{
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
						},
						"ulimit": schema.SetNestedBlock{
							MarkdownDescription: "The ulimits of the service.",
							NestedObject: schema.NestedBlockObject{
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
								// Validators: []validator.Object{
								// 	objectvalidator.ConflictsWith(path.MatchRelative().AtName("value"), path.MatchRelative().AtName("soft")),
								// 	objectvalidator.ConflictsWith(path.MatchRelative().AtName("value"), path.MatchRelative().AtName("hard")),
								// },
							},
						},
					},
				},
			},
			"network": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
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
					},
				},
			},
			"volume": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the volume.",
							Required:            true,
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
			"secret": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the secret.",
							Optional:            true,
						},
					},
				},
			},
			"config": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the config.",
							Required:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "The content of the config.",
							Optional:            true,
						},
						"file": schema.StringAttribute{
							MarkdownDescription: "The file of the config.",
							Optional:            true,
						},
					},
				},
			},
			"extension": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
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
}
