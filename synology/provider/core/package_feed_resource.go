package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

type PackageFeedResourceModel struct {
	Name types.String `tfsdk:"name"`
	URL  types.String `tfsdk:"url"`
}

var _ resource.Resource = &PackageFeedResource{}

func NewPackageFeedResource() resource.Resource {
	return &PackageFeedResource{}
}

type PackageFeedResource struct {
	client core.Api
}

// Create implements resource.Resource.
func (p *PackageFeedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PackageFeedResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	feedName := data.Name.ValueString()
	feedURL := data.URL.ValueString()

	if data.Name.IsNull() || data.Name.IsUnknown() {
		resp.Diagnostics.AddError("Name is required", "Name is required")
		return
	}

	err := p.client.PackageFeedAdd(ctx, core.PackageFeedAddRequest{
		List: core.PackageFeedItem{
			Name: feedName,
			Feed: feedURL,
		},
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to create package feed", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete implements resource.Resource.
func (p *PackageFeedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PackageFeedResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	feedURL := data.URL.ValueString()

	err := p.client.PackageFeedDelete(ctx, core.PackageFeedDeleteRequest{
		List: core.PackageFeeds{feedURL},
	})

	if err != nil {
		resp.Diagnostics.AddError("Failed to delete package feed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

// Metadata implements resource.Resource.
func (p *PackageFeedResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "package_feed")
}

// Read implements resource.Resource.
func (p *PackageFeedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PackageFeedResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	name := data.Name.ValueString()
	url := data.URL.ValueString()

	feeds, err := p.client.PackageFeedList(ctx)

	if err != nil {
		resp.Diagnostics.AddError("Failed to list package feeds", err.Error())
		return
	}

	found := false
	foundURL := ""
	for _, feed := range feeds.Items {
		if feed.Name == name {
			found = true
			foundURL = feed.Feed
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Package feed not found", fmt.Sprintf("Package feed %s not found", name))
		return
	}

	if foundURL != url {
		resp.Diagnostics.AddError("Package feed URL does not match", fmt.Sprintf("Package feed URL does not match. Expected: %s, got: %s", url, foundURL))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schema implements resource.Resource.
func (p *PackageFeedResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A resource for managing package feeds.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the package feed.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to the package feed.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Update implements resource.Resource.
func (p *PackageFeedResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {

}

func (f *PackageFeedResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(synology.Api)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	f.client = client.CoreAPI()
}
