package container

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/appkins/terraform-provider-synology/synology/provider/container/models"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
	"github.com/synology-community/go-synology/pkg/api/docker"
	"github.com/synology-community/go-synology/pkg/api/filestation"
	"github.com/synology-community/go-synology/pkg/util/form"

	composetypes "github.com/compose-spec/compose-go/v2/types"
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
	Services      types.Set    `tfsdk:"service"`
	Networks      types.Set    `tfsdk:"network"`
	Volumes       types.Set    `tfsdk:"volume"`
	Secrets       types.Set    `tfsdk:"secret"`
	Configs       types.Set    `tfsdk:"config"`
	Extensions    types.Set    `tfsdk:"extension"`
	Run           types.Bool   `tfsdk:"run"`
	State         types.String `tfsdk:"state"`
	ServicePortal types.Set    `tfsdk:"service_portal"`
	// ComposeFiles types.ListType `tfsdk:"compose_files"`
	// Environment  types.MapType  `tfsdk:"environment"`
}

const projectDescription = `A Docker Compose project for the Container Manager Synology API.

> **Note:** Synology creates a shared folder for each project. The shared folder is created in the ` + "`/projects`" + ` directory by default. The shared folder is named after the project name. The shared folder is used to store the project files and data. The shared folder is mounted to the ` + "`/volume1/projects`" + ` directory on the Synology NAS.

`

func getProjectYaml(ctx context.Context, data ProjectResourceModel, projYaml *string) (diags diag.Diagnostics) {
	diags = []diag.Diagnostic{}
	project := composetypes.Project{}

	if !data.Services.IsNull() && !data.Services.IsUnknown() {

		elements := []models.Service{}
		diags.Append(data.Services.ElementsAs(ctx, &elements, true)...)

		if diags.HasError() {
			return
		}

		project.Services = map[string]composetypes.ServiceConfig{}

		for _, v := range elements {

			service := composetypes.ServiceConfig{}
			diags.Append(v.AsComposeConfig(ctx, &service)...)
			if diags.HasError() {
				return
			}

			project.Services[service.Name] = service
		}
	}

	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {

		elements := []models.Network{}
		diags.Append(data.Networks.ElementsAs(ctx, &elements, true)...)

		if diags.HasError() {
			return
		}

		project.Networks = map[string]composetypes.NetworkConfig{}

		for _, v := range elements {
			n := composetypes.NetworkConfig{}

			diags.Append(v.AsComposeConfig(ctx, &n)...)
			if diags.HasError() {
				return
			}

			project.Networks[n.Name] = n
		}
	}

	if !data.Volumes.IsNull() && !data.Volumes.IsUnknown() {

		elements := []models.Volume{}
		diags.Append(data.Volumes.ElementsAs(ctx, &elements, true)...)

		if diags.HasError() {
			return
		}

		project.Volumes = map[string]composetypes.VolumeConfig{}

		for _, v := range elements {
			n := composetypes.VolumeConfig{}

			diags.Append(v.AsComposeConfig(ctx, &n)...)
			if diags.HasError() {
				return
			}

			project.Volumes[n.Name] = n
		}
	}

	if !data.Configs.IsNull() && !data.Configs.IsUnknown() {

		elements := []models.Config{}
		diags.Append(data.Configs.ElementsAs(ctx, &elements, true)...)

		if diags.HasError() {
			return
		}

		project.Configs = map[string]composetypes.ConfigObjConfig{}

		for _, v := range elements {
			n := composetypes.ConfigObjConfig{}

			diags.Append(v.AsComposeConfig(ctx, &n)...)
			if diags.HasError() {
				return
			}

			project.Configs[n.Name] = n
		}
	}

	projectYAML, err := project.MarshalYAML()
	if err != nil {
		diags.Append(diag.NewErrorDiagnostic("Failed to marshal docker-compose.yml", err.Error()))
		return
	}
	pyaml := string(projectYAML)
	*projYaml = pyaml

	return
}

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

	if data.SharePath.IsNull() || data.SharePath.IsUnknown() {
		data.SharePath = types.StringValue(fmt.Sprintf("/projects/%s", data.Name.ValueString()))
	}

	err := f.ensureProjectShare(ctx, data.SharePath.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Failed to create project share", err.Error())
		return
	}

	// Set the file values where content is specified
	if !data.Configs.IsNull() && !data.Configs.IsUnknown() {
		elements := []models.Config{}
		resp.Diagnostics.Append(data.Configs.ElementsAs(ctx, &elements, true)...)
		if resp.Diagnostics.HasError() {
			return
		}
		changed := false
		for i, v := range elements {
			if !v.Content.IsNull() || !v.Content.IsUnknown() {
				fileName := fmt.Sprintf("config_%s", v.Name.ValueString())
				fileContent := v.Content.ValueString()
				v.File = types.StringValue(fileName)
				elements[i] = v
				changed = true

				// Upload the file
				_, err := f.fsClient.Upload(
					ctx,
					data.SharePath.ValueString(),
					form.File{
						Name:    fileName,
						Content: fileContent,
					}, false,
					true)
				if err != nil {
					resp.Diagnostics.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
					return
				}
			}
		}
		if changed {
			var elementValues []attr.Value
			for _, v := range elements {
				elementValues = append(elementValues, v.Value())
			}
			data.Configs = types.SetValueMust(models.Config{}.ModelType(), elementValues)
		}
	}

	projectYAML := ""
	resp.Diagnostics.Append(getProjectYaml(ctx, data, &projectYAML)...)
	if resp.Diagnostics.HasError() {
		return
	}

	servicePortal := models.ServicePortal{}
	resp.Diagnostics.Append(servicePortal.First(ctx, data.ServicePortal)...)

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
	}

	if shouldUpdate {
		p, err := f.client.ProjectGetByName(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get project", err.Error())
			return
		}

		data.ID = types.StringValue(p.ID)
		status := p.Status

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
			resp.Diagnostics.AddError("Failed to build project", err.Error())
			return
		}
	}

	proj, err := f.client.ProjectGet(ctx, data.ID.ValueString())

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

	proj, err := f.client.ProjectGet(ctx, data.ID.ValueString())

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

	id := data.ID.ValueString()

	proj, err := f.client.ProjectGet(ctx, data.ID.ValueString())

	if err != nil {
		if proj, err = f.client.ProjectGetByName(ctx, data.Name.ValueString()); err != nil {
			switch err.(type) {
			case docker.ProjectNotFoundError:
				resp.State.RemoveResource(ctx)
				return
			default:
				resp.Diagnostics.AddError("Failed to get project", err.Error())
				return
			}
		} else if data.ID.IsNull() || data.ID.IsUnknown() || data.ID.ValueString() != proj.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(proj.ID))...)
			if resp.Diagnostics.HasError() {
				return
			}
			id = proj.ID
		}
	}

	if !proj.IsRunning() && data.Run.ValueBool() {
		_, err = f.client.ProjectBuildStream(ctx, docker.ProjectStreamRequest{
			ID: id,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to build project", err.Error())
			return
		}
		proj, err = f.client.ProjectGet(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get project", err.Error())
			return
		}
	}

	if data.State.ValueString() != proj.Status {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("state"), types.StringValue(proj.Status))...)
	}
}

// Update implements resource.Resource.
func (f *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel

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
	resp.Diagnostics.Append(servicePortal.First(ctx, plan.ServicePortal)...)

	if configChanged {
		// Set the file values where content is specified
		if !plan.Configs.IsNull() && !plan.Configs.IsUnknown() {
			elements := []models.Config{}
			stateElements := []models.Config{}
			resp.Diagnostics.Append(plan.Configs.ElementsAs(ctx, &elements, true)...)
			resp.Diagnostics.Append(state.Configs.ElementsAs(ctx, &stateElements, true)...)
			if resp.Diagnostics.HasError() {
				return
			}
			changed := false
			for i, v := range elements {
				if !(v.Content.IsNull() || v.Content.IsUnknown()) {
					fileName := fmt.Sprintf("config_%s", v.Name.ValueString())
					fileContent := v.Content.ValueString()
					v.File = types.StringValue(fileName)
					elements[i] = v
					changed = true

					// Upload the file
					_, err := f.fsClient.Upload(
						ctx,
						plan.SharePath.ValueString(),
						form.File{
							Name:    fileName,
							Content: fileContent,
						}, false,
						true)
					if err != nil {
						resp.Diagnostics.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
						return
					}
				} else {
					if len(stateElements) != len(elements) {
						servicesChanged = true
					} else {
						sv := stateElements[i]
						if sv.File.ValueString() != v.File.ValueString() {
							servicesChanged = true
						}
					}
				}
			}
			if changed {
				var elementValues []attr.Value
				for _, v := range elements {
					elementValues = append(elementValues, v.Value())
				}
				plan.Configs = types.SetValueMust(models.Config{}.ModelType(), elementValues)
			}
		}
	}

	if servicesChanged {
		projectYAML := ""
		resp.Diagnostics.Append(getProjectYaml(ctx, plan, &projectYAML)...)
		if resp.Diagnostics.HasError() {
			return
		}

		proj, err := f.client.ProjectGet(ctx, plan.ID.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Failed to get project", err.Error())
			return
		}

		if proj.Content == projectYAML {
			tflog.Info(ctx, "No changes detected in project, skipping update")
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("state"), types.StringValue(proj.Status))...)
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
		resp.Diagnostics.AddError("Failed to get project", err.Error())
		return
	}

	plan.State = types.StringValue(proj.Status)

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
			"share_path": schema.StringAttribute{
				MarkdownDescription: "The share path of the project.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				MarkdownDescription: "Synology Web Station configuration for the docker compose project.",
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
				MarkdownDescription: "Docker compose services.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the service.",
							Optional:            true,
						},
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
						"depends_on": schema.SetNestedBlock{
							MarkdownDescription: "The dependencies of the service.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the dependency.",
										Required:            true,
									},
									"condition": schema.StringAttribute{
										MarkdownDescription: "The condition of the dependency.",
										Optional:            true,
									},
									"restart": schema.BoolAttribute{
										MarkdownDescription: "Whether to restart.",
										Optional:            true,
									},
									"required": schema.BoolAttribute{
										MarkdownDescription: "Whether the dependency is required.",
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
						"config": schema.SetNestedBlock{
							MarkdownDescription: "The configs of the service.",
							NestedObject: schema.NestedBlockObject{
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
			"network": schema.SetNestedBlock{
				MarkdownDescription: "Docker compose networks.",
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
					Blocks: map[string]schema.Block{
						"ipam": schema.SetNestedBlock{
							MarkdownDescription: "The IPAM of the network.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"driver": schema.StringAttribute{
										MarkdownDescription: "The driver of the IPAM.",
										Optional:            true,
									},
								},
								Blocks: map[string]schema.Block{
									"config": schema.SetNestedBlock{
										MarkdownDescription: "The config of the IPAM.",
										NestedObject: schema.NestedBlockObject{
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
												"aux_address": schema.MapAttribute{
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
			},
			"volume": schema.SetNestedBlock{
				MarkdownDescription: "Docker compose volumes.",
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
				MarkdownDescription: "Docker compose secrets.",
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
				MarkdownDescription: "Docker compose configs.",
				NestedObject: schema.NestedBlockObject{
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
	f.fsClient = client.FileStationAPI()
	f.coreClient = client.CoreAPI()
}
