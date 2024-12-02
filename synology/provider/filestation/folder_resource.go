package filestation

import (
	"context"
	"errors"
	"fmt"
	filepath "path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/filestation"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FolderResource{}

func NewFolderResource() resource.Resource {
	return &FolderResource{}
}

type FolderResource struct {
	client filestation.Api
}

// FolderResourceModel describes the resource data model.
type FolderResourceModel struct {
	Name          types.String `tfsdk:"name"`
	Path          types.String `tfsdk:"path"`
	CreateParents types.Bool   `tfsdk:"create_parents"`
	RealPath      types.String `tfsdk:"real_path"`
}

// Create implements resource.Resource.
func (f *FolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FolderResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createParents := true

	if !data.CreateParents.IsNull() || !data.CreateParents.IsUnknown() {
		createParents = data.CreateParents.ValueBool()
	}

	data.CreateParents = types.BoolValue(createParents)

	path := data.Path.ValueString()
	name := data.Name.ValueString()

	flist, err := f.client.CreateFolder(ctx, []string{path}, []string{name}, createParents)

	if err != nil {
		resp.Diagnostics.AddError("Failed to create folder", fmt.Sprintf("Failed to create folder, got error: %s", err))
		return
	}

	if flist == nil {
		resp.Diagnostics.AddError("Failed to create folder", fmt.Sprintf("Failed to create folder, got error: %s", err))
		return
	}

	file, err := f.client.Get(ctx, path)
	if err != nil {
		if _, ok := err.(filestation.FileNotFoundError); ok {
			resp.Diagnostics.AddError("Error finding file after create", fmt.Sprintf("Unable to get file, got error: %s", err))
			return
		} else {
			resp.Diagnostics.AddError("Failed to get file", fmt.Sprintf("Unable to get file, got error: %s", err))
			return
		}
	}

	if file == nil {
		resp.Diagnostics.AddError("Error finding file after create", fmt.Sprintf("Unable to get file, got error: %s", err))
		return
	}

	data.RealPath = types.StringValue(file.Additional.RealPath)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *FolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FolderResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	path := data.Path.ValueString()
	// Start Delete the file
	_, err := f.client.Delete(ctx, []string{path}, true)

	if err != nil {
		if e := errors.Unwrap(err); e != nil {
			resp.Diagnostics.AddError("Failed to delete file", fmt.Sprintf("Unable to delete file, got error: %s", e))
		} else {
			resp.Diagnostics.AddError("Failed to delete file", fmt.Sprintf("Unable to delete file, got error: %s", err))
		}
		return
	}
}

// Read implements resource.Resource.
func (f *FolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FolderResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	path := data.Path.ValueString()

	parts := filepath.SplitList(path)
	if len(parts) == 0 {
		resp.Diagnostics.AddError("Failed to list files", fmt.Sprintf("Unable to list files, got error: %s", "path is empty"))
		return
	}

	file, err := f.client.Get(ctx, path)
	if err != nil {
		if _, ok := err.(filestation.FileNotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError("Failed to get file", fmt.Sprintf("Unable to get file, got error: %s", err))
			return
		}
	}

	if file == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if file.Additional.RealPath != data.RealPath.ValueString() {
		data.RealPath = types.StringValue(file.Additional.RealPath)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

// Update implements resource.Resource.
func (f *FolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FolderResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.CreateParents.IsNull() || data.CreateParents.IsUnknown() {
		data.CreateParents = types.BoolValue(true)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *FolderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "folder")
}

// Schema implements resource.Resource.
func (f *FolderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A file on the Synology NAS Folderstation.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the folder to be created.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "A destination folder path starting with a shared folder to which files can be uploaded.",
				Required:            true,
			},
			"create_parents": schema.BoolAttribute{
				MarkdownDescription: "If true, create parent directories if they do not exist.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"real_path": schema.StringAttribute{
				MarkdownDescription: "The real path of the folder.",
				Computed:            true,
			},
		},
	}
}

func (f *FolderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	f.client = client.FileStationAPI()
}

func (f *FolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	p := req.ID
	basedir := filepath.Dir(p)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), basedir)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), filepath.Base(p))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("create_parents"), true)...)

	files, err := f.client.List(ctx, basedir)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list files", fmt.Sprintf("Unable to list files, got error: %s", err))
		return
	}
	for _, file := range files.Files {
		if file.IsDir && file.Path == p {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("real_path"), file.Additional.RealPath)...)
			continue
		}
	}
}
