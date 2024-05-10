package filestation

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/package"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource = &FolderResource{}
)

func NewFolderResource() resource.Resource {
	return &FolderResource{}
}

type FolderResource struct {
	client client.SynologyClient
}

// FolderResourceModel describes the resource data model.
type FolderResourceModel struct {
	ConfigurableAttribute types.String `tfsdk:"configurable_attribute"`
	Defaulted             types.String `tfsdk:"defaulted"`
	Id                    types.String `tfsdk:"id"`
}

// Create implements resource.Resource.
func (f *FolderResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {
	panic("unimplemented")
}

// Delete implements resource.Resource.
func (f *FolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	panic("unimplemented")
}

// Read implements resource.Resource.
func (f *FolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	panic("unimplemented")
}

// Update implements resource.Resource.
func (f *FolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("unimplemented")
}

// Metadata implements resource.Resource.
func (f *FolderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "folder")
}

// Schema implements resource.Resource.
func (f *FolderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Filestation folder.",

		Attributes: map[string]schema.Attribute{
			"iterations": schema.Int64Attribute{
				MarkdownDescription: "Number of iterations.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(100000),
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "Output format; will additionally be base64 encoded.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("{{ printf \"%s:%s\" (b64enc .Salt) (b64enc .Key) }}"),
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password input to encrypt.",
				Required:            true,
				Sensitive:           true,
			},
			"hash_algorithm": schema.StringAttribute{
				MarkdownDescription: "The hash function to use.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("sha256"),
			},
			"salt": schema.StringAttribute{
				MarkdownDescription: "The generated salt value.",
				Computed:            true,
				Sensitive:           true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The generated key value.",
				Computed:            true,
				Sensitive:           true,
			},
			"result": schema.StringAttribute{
				MarkdownDescription: "The formatted key result.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}
