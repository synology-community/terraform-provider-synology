package filestation

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/package"
	"github.com/synology-community/synology-api/package/util/form"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource = &FileResource{}
)

func NewFileResource() resource.Resource {
	return &FileResource{}
}

type FileResource struct {
	client client.SynologyClient
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	Path          types.String `tfsdk:"path"`
	CreateParents types.Bool   `tfsdk:"create_parents"`
	Overwrite     types.Bool   `tfsdk:"overwrite"`
	Name          types.String `tfsdk:"name"`
	Content       types.String `tfsdk:"content"`
	Id            types.String `tfsdk:"id"`
	MD5           types.String `tfsdk:"md5"`
}

// Create implements resource.Resource.
func (f *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	file := form.File{
		Name:    data.Name.ValueString(),
		Content: data.Content.ValueString(),
	}

	// Check if the file exists

	// If it exists, check the MD5 checksum

	// If the checksums match, return

	// Upload the file
	_, err := f.client.FileStationAPI().Upload(data.Path.ValueString(), &file, data.CreateParents.ValueBool(), data.Overwrite.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
		return
	}

	// Get the file's MD5 checksum
	// md5, err := f.client.FileStationAPI().GetMD5(data.Path.ValueString(), data.Name.ValueString())

	// Store the MD5 checksum in the state

}

// Delete implements resource.Resource.
func (f *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	fileName := filepath.Join(data.Path.ValueString(), data.Name.ValueString())

	// Start Delete the file
	rdel, err := f.client.FileStationAPI().DeleteStart([]string{fileName}, true)

	if err != nil {
		resp.Diagnostics.AddError("Failed to delete file", fmt.Sprintf("Unable to delete file, got error: %s", err))
		return
	}

	waitUntil := time.Now().Add(5 * time.Minute)
	completed := false
	for !completed {
		// Check the status of the delete operation
		rstat, err := f.client.FileStationAPI().DeleteStatus(rdel.TaskID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to delete file", fmt.Sprintf("Unable to delete file, got error: %s", err))
			return
		}

		if rstat.Finished {
			completed = true
		}

		if time.Now().After(waitUntil) {
			resp.Diagnostics.AddError("Failed to delete file", "Timeout waiting for file to be deleted")
			break
		}

		time.Sleep(5 * time.Second)
	}
}

// Read implements resource.Resource.
func (f *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	fileName := filepath.Join(data.Path.ValueString(), data.Name.ValueString())

	// Start Delete the file
	rdel, err := f.client.FileStationAPI().MD5Start(fileName)

	if err != nil {
		resp.Diagnostics.AddError("Failed to delete file", fmt.Sprintf("Unable to delete file, got error: %s", err))
		return
	}

	retry := 0
	completed := false
	for !completed {
		// Check the status of the delete operation
		hstat, err := f.client.FileStationAPI().MD5Status(rdel.TaskID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get file hash", fmt.Sprintf("Unable to get file hash, got error: %s", err))
			return
		}

		if hstat.Finished {
			if hstat.MD5 != "" {
				data.MD5 = types.StringValue(hstat.MD5)
			}

			completed = true
		}

		if retry > 2 {
			completed = true
			continue
		}
		retry++
		time.Sleep(2 * time.Second)
	}

	resp.State.Set(ctx, &data)
}

// Update implements resource.Resource.
func (f *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	file := form.File{
		Name:    data.Name.ValueString(),
		Content: data.Content.ValueString(),
	}

	// Check if the file exists

	// If it exists, check the MD5 checksum

	// If the checksums match, return

	// Upload the file
	_, err := f.client.FileStationAPI().Upload(
		data.Path.ValueString(),
		&file, data.CreateParents.ValueBool(),
		true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
		return
	}

	// Get the file's MD5 checksum
	// md5, err := f.client.FileStationAPI().GetMD5(data.Path.ValueString(), data.Name.ValueString())

	// Store the MD5 checksum in the state
}

// Metadata implements resource.Resource.
func (f *FileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "file")
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
			"content": schema.StringAttribute{
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
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the file to upload to the Synology DSM.",
				Optional:            true,
				Computed:            true,
			},
			"md5": schema.StringAttribute{
				MarkdownDescription: "The MD5 checksum of the file.",
				Computed:            true,
			},
		},
	}
}
