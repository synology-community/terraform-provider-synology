package filestation

import (
	"context"

	client "github.com/appkins/terraform-provider-synology/synology-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource = &FileResource{}
)

func NewFileResource() resource.Resource {
	return &FileResource{}
}

type FileResource struct {
	client client.Client
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	ConfigurableAttribute types.String `tfsdk:"configurable_attribute"`
	Defaulted             types.String `tfsdk:"defaulted"`
	Id                    types.String `tfsdk:"id"`
}

// Create implements resource.Resource.
func (f *FileResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {
	panic("unimplemented")
}

// Delete implements resource.Resource.
func (f *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	panic("unimplemented")
}

// Read implements resource.Resource.
func (f *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	panic("unimplemented")
}

// Update implements resource.Resource.
func (f *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("unimplemented")
}

// Metadata implements resource.Resource.
func (f *FileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "folder")
}

// Schema implements resource.Resource.
func (f *FileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A file on the Synology NAS Filestation.",

		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "A destination folder path starting with a shared folder to which files can be uploaded.",
				Required:            true,
			},
			"data": schema.StringAttribute{
				MarkdownDescription: "A destination folder path starting with a shared folder to which files can be uploaded.",
				Required:            true,
			},
			"create_parents": schema.BoolAttribute{
				MarkdownDescription: "Create parent folder(s) if none exist.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"overwrite": schema.BoolAttribute{
				MarkdownDescription: "Overwrite the destination file if one exists.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"md5": schema.StringAttribute{
				MarkdownDescription: "The MD5 checksum of the file.",
				Computed:            true,
			},
		},
	}
}
