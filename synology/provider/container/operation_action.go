package container

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/docker"
)

// Ensure the implementation satisfies framework interfaces.
var (
	_ action.Action              = &ContainerOperationAction{}
	_ action.ActionWithConfigure = &ContainerOperationAction{}
)

// NewContainerOperationAction returns a new instance of the container operation action.
func NewContainerOperationAction() action.Action {
	return &ContainerOperationAction{}
}

type ContainerOperationAction struct {
	client docker.Api
}

// containerOperationActionModel describes the action request and response data model.
type containerOperationActionModel struct {
	Name      types.String `tfsdk:"name"`
	Operation types.String `tfsdk:"operation"`
}

func (a *ContainerOperationAction) Metadata(
	ctx context.Context,
	req action.MetadataRequest,
	resp *action.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_container_operation"
}

func (a *ContainerOperationAction) Schema(
	ctx context.Context,
	req action.SchemaRequest,
	resp *action.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs an operation on a Synology Docker container, allowing start, stop, or restart actions.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the container to perform the operation on.",
				Required:            true,
			},
			"operation": schema.StringAttribute{
				MarkdownDescription: "Operation to perform on the container. Valid values are `start`, `stop`, and `restart`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("start", "stop", "restart"),
				},
			},
		},
	}
}

func (a *ContainerOperationAction) Configure(
	ctx context.Context,
	req action.ConfigureRequest,
	resp *action.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Api)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf(
				"Expected client.Api, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	a.client = client.DockerAPI()
}

func (a *ContainerOperationAction) Invoke(
	ctx context.Context,
	req action.InvokeRequest,
	resp *action.InvokeResponse,
) {
	var config containerOperationActionModel

	// Read the action configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Name.IsNull() || config.Name.IsUnknown() {
		return
	}

	if config.Operation.IsNull() || config.Operation.IsUnknown() {
		return
	}

	// Validate operation
	operation := config.Operation.ValueString()

	containerName := config.Name.ValueString()

	request := docker.ContainerOperationRequest{
		Name: containerName,
	}

	tflog.Info(ctx, "running container action")

	// Perform the requested operation
	var err error
	switch operation {
	case "start":
		_, err = a.client.ContainerStart(ctx, request)
	case "stop":
		_, err = a.client.ContainerStop(ctx, request)
	case "restart":
		_, err = a.client.ContainerRestart(ctx, request)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Performing Container Operation",
			fmt.Sprintf(
				"Could not %s container %s: %s",
				operation,
				containerName,
				err.Error(),
			),
		)
		return
	}
}
