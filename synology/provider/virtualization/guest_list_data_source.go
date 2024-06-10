package virtualization

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/pkg"
	"github.com/synology-community/synology-api/pkg/api/virtualization"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GuestListDataSource{}

func NewGuestListDataSource() datasource.DataSource {
	return &GuestListDataSource{}
}

type GuestListDataSource struct {
	client virtualization.Api
}

type GuestListDataSourceModel struct {
	Guest types.List `tfsdk:"guest"`
}

func (d *GuestListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "guest_list")
}

func (d *GuestListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Virtualization --- GuestList data source",
		MarkdownDescription: "Virtualization --- GuestList data source",

		Attributes: map[string]schema.Attribute{
			"guest": schema.ListAttribute{
				MarkdownDescription: "List of guestList.",
				Computed:            true,
				ElementType:         GuestDataSourceModel{}.ModelType(),
			},
		},
	}
}

func (d *GuestListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GuestListDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	clientResponse, err := d.client.GuestList(ctx)

	if err != nil {
		resp.Diagnostics.AddError("API request failed", fmt.Sprintf("Unable to read data source, got error: %s", err))
		return
	}

	guestList := []attr.Value{}

	for _, v := range clientResponse.Guests {
		m := GuestDataSourceModel{}
		if err := m.FromGuest(&v); err != nil {
			resp.Diagnostics.AddError("Failed to read guest data", err.Error())
			return
		}

		guestList = append(guestList, m.Value())
	}

	vv, _ := types.ListValue(GuestDataSourceModel{}.ModelType(), guestList)

	data.Guest = vv

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *GuestListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.VirtualizationAPI()
}
