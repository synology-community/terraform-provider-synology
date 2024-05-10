package filestation

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/package"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &infoDataSource{}

func NewInfoDataSource() datasource.DataSource {
	return &infoDataSource{}
}

type infoDataSource struct {
	client client.SynologyClient
}

type infoDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Hostname               types.String `tfsdk:"hostname"`
	IsManager              types.Bool   `tfsdk:"is_manager"`
	SupportVirtualProtocol types.String `tfsdk:"support_virtual_protocol"`
	SupportSharing         types.Bool   `tfsdk:"support_sharing"`
}

func (d *infoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "info")
}

func (d *infoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Info data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for this data source.",
				Computed:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "Hostname of Synology station.",
				Computed:    true,
			},
			"is_manager": schema.BoolAttribute{
				Description: "Indicates whether current user is an administrator.",
				Computed:    true,
			},
			"support_virtual_protocol": schema.StringAttribute{
				Description: "Comma separated list of virtual file system, which current user is able to mount.",
				Computed:    true,
			},
			"support_sharing": schema.BoolAttribute{
				Description: "Indicates whether current user can share files/folders.",
				Computed:    true,
			},
		},
	}
}

func (d *infoDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *infoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// var data infoDataSourceModel

	// clientResponse := filestation.FileStationInfoResponse{}
	// clientRequest := filestation.NewFileStationInfoRequest(2)
	// if err := d.client.Get(clientRequest, &clientResponse); err != nil {
	// 	resp.Diagnostics.AddError("API request failed", fmt.Sprintf("Unable to read data source, got error: %s", err))
	// 	return
	// }
	// if !clientResponse.Success() {
	// 	resp.Diagnostics.AddError(
	// 		"Client error",
	// 		fmt.Sprintf("Unable to read data source, got error: %s", clientResponse.GetError()),
	// 	)
	// 	return
	// }

	// data.ID = types.StringValue(clientResponse.Hostname)
	// data.Hostname = types.StringValue(clientResponse.Hostname)
	// data.IsManager = types.BoolValue(clientResponse.IsManager)
	// data.SupportSharing = types.BoolValue(clientResponse.Supportsharing)
	// data.SupportVirtualProtocol = types.StringValue(clientResponse.SupportVirtualProtocol)

	// // Save data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
