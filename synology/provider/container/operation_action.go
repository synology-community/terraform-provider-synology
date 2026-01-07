package container

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/docker"
)

// Ensure the implementation satisfies framework interfaces.
var (
	_ action.Action              = &containerOperationAction{}
	_ action.ActionWithConfigure = &containerOperationAction{}
)

// NewContainerOperationAction returns a new instance of the container operation action.
func NewContainerOperationAction() action.Action {
	return &containerOperationAction{}
}

type containerOperationAction struct {
	client docker.Api
}

// containerOperationActionModel describes the action request and response data model.
type containerOperationActionModel struct {
	Name      types.String `tfsdk:"name"`
	Operation types.String `tfsdk:"operation"`
}

func (a *containerOperationAction) Metadata(
	ctx context.Context,
	req action.MetadataRequest,
	resp *action.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_container_operation"
}

func (a *containerOperationAction) Schema(
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
			},
		},
	}
}

func (a *containerOperationAction) Configure(
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

func (a *containerOperationAction) Invoke(
	ctx context.Context,
	req action.InvokeRequest,
	resp *action.InvokeResponse,
) {
	var config containerOperationActionModel

	// Read the action configuration
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate operation
	operation := config.Operation.ValueString()
	validOperations := []string{"start", "stop", "restart"}
	if !slices.Contains(validOperations, operation) {
		resp.Diagnostics.AddError(
			"Invalid Operation",
			fmt.Sprintf(
				"Operation must be one of: start, stop, restart. Got: %s",
				operation,
			),
		)
		return
	}

	containerName := config.Name.ValueString()

	// Perform the requested operation
	var err error
	switch operation {
	case "start":
		_, err = a.client.ContainerStart(ctx, docker.ContainerStartRequest{
			Name: containerName,
		})
	case "stop":
		_, err = a.client.ContainerStop(ctx, docker.ContainerStopRequest{
			Name: containerName,
		})
	case "restart":
		_, err = a.client.ContainerRestart(ctx, docker.ContainerRestartRequest{
			Name: containerName,
		})
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

	// Action completed successfully - no result data to return for actions
}
