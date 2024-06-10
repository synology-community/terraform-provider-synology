package container

import (
	"context"
	"fmt"

	"github.com/appkins/terraform-provider-synology/synology/provider/container/models"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/pkg"
	"github.com/synology-community/synology-api/pkg/api/docker"

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

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Services   types.Set    `tfsdk:"service"`
	Networks   types.Set    `tfsdk:"network"`
	Volumes    types.Set    `tfsdk:"volume"`
	Secrets    types.Set    `tfsdk:"secret"`
	Configs    types.Set    `tfsdk:"config"`
	Extensions types.Set    `tfsdk:"extension"`
	// ComposeFiles types.ListType `tfsdk:"compose_files"`
	// Environment  types.MapType  `tfsdk:"environment"`
}

// Create implements resource.Resource.
func (f *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := composetypes.Project{}

	if !data.Services.IsNull() && !data.Services.IsUnknown() {

		elements := []models.Service{}
		diags := data.Services.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read networks", "Unable to read networks")
			return
		}

		if project.Services == nil {
			project.Services = make(map[string]composetypes.ServiceConfig)
		}

		for _, v := range elements {

			service := v.AsComposeServiceConfig()

			project.Services[service.Name] = service
		}
	}

	projectYAML, err := project.MarshalYAML()
	if err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal docker-compose.yml", err.Error())
		return
	}

	res, err := f.client.ProjectCreate(ctx, docker.ProjectCreateRequest{
		Name:    data.Name.ValueString(),
		Content: string(projectYAML),
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to create project", err.Error())
		return
	}

	data.ID = types.StringValue(res.ID)

	// project := compose.Project{
	// 	Name: data.Name.ValueString(),
	// }

	// if !data.Networks.IsNull() && !data.Networks.IsUnknown() {

	// 	var elements []compose.NetworkConfig
	// 	diags := data.Networks.WithElementType()

	// }

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

}

// Read implements resource.Resource.
func (f *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

}

// Update implements resource.Resource.
func (f *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

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
		MarkdownDescription: "Docker --- A Docker Compose project for the Container Manager Synology API.",

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
			// "compose_files": schema.ListAttribute{
			// 	MarkdownDescription: "The list of compose files.",
			// 	ElementType:         types.StringType,
			// 	Optional:            true,
			// },
		},
		Blocks: map[string]schema.Block{
			"service": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the service.",
							Optional:            true,
						},
						"image": schema.StringAttribute{
							MarkdownDescription: "The image of the service.",
							Optional:            true,
						},
						"replicas": schema.Int64Attribute{
							MarkdownDescription: "The number of replicas.",
							Optional:            true,
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
					},
				},
			},
			"volume": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the volume.",
							Optional:            true,
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

	client, ok := req.ProviderData.(client.SynologyClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	f.client = client.DockerAPI()
}
