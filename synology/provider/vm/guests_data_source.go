package vm

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/package"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GuestsDataSource{}

func NewGuestsDataSource() datasource.DataSource {
	return &GuestsDataSource{}
}

type GuestsDataSource struct {
	client client.SynologyClient
}

type GuestsDataSourceModel struct {
	Guest types.List `tfsdk:"guest"`
}

func (d *GuestsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guests"
}

func (d *GuestsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Guests data source",

		Attributes: map[string]schema.Attribute{
			"guest": schema.ListAttribute{
				MarkdownDescription: "List of guests.",
				Computed:            true,
				ElementType:         GuestDataSourceModel{}.ModelType(),
			},
		},
	}
}

func (d *GuestsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GuestsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GuestsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	clientResponse, err := d.client.VirtualizationAPI().ListGuests()

	if err != nil {
		resp.Diagnostics.AddError("API request failed", fmt.Sprintf("Unable to read data source, got error: %s", err))
		return
	}

	guests := []attr.Value{}

	for _, v := range clientResponse.Guests {
		m := GuestDataSourceModel{}
		if err := m.FromGuest(&v); err != nil {
			resp.Diagnostics.AddError("Failed to read guest data", err.Error())
			return
		}

		guests = append(guests, m.Value())
	}

	vv, _ := types.ListValue(GuestDataSourceModel{}.ModelType(), guests)

	data.Guest = vv

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
