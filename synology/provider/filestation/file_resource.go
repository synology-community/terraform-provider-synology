package filestation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/synology-api/pkg"
	"github.com/synology-community/synology-api/pkg/api/filestation"
	"github.com/synology-community/synology-api/pkg/util/form"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FileResource{}

func NewFileResource() resource.Resource {
	return &FileResource{}
}

type FileResource struct {
	client filestation.FileStationApi
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	Path          types.String `tfsdk:"path"`
	CreateParents types.Bool   `tfsdk:"create_parents"`
	Overwrite     types.Bool   `tfsdk:"overwrite"`
	Content       types.String `tfsdk:"content"`
	AccessTime    types.Int64  `tfsdk:"access_time"`
	ModifiedTime  types.Int64  `tfsdk:"modified_time"`
	ChangeTime    types.Int64  `tfsdk:"change_time"`
	CreateTime    types.Int64  `tfsdk:"create_time"`
}

// Create implements resource.Resource.
func (f *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createParents := data.CreateParents.ValueBool()
	overwrite := data.Overwrite.ValueBool()
	path := data.Path.ValueString()
	fileContent := data.Content.ValueString()
	fileName := filepath.Base(data.Path.ValueString())
	fileDir := filepath.Dir(data.Path.ValueString())

	// Upload the file
	_, err := f.client.Upload(ctx, fileDir, form.File{
		Name:    fileName,
		Content: fileContent,
	}, createParents, overwrite)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
		return
	}

	files, err := f.client.List(ctx, fileDir)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list files", fmt.Sprintf("Unable to list files, got error: %s", err))
		return
	}
	for _, file := range files.Files {
		if file.Path == path {
			tflog.Info(ctx, fmt.Sprintf("File found: %s", file.Path))

			data.ModifiedTime = types.Int64Value(file.Additional.Time.Mtime.Unix())
			data.AccessTime = types.Int64Value(file.Additional.Time.Atime.Unix())
			data.ChangeTime = types.Int64Value(file.Additional.Time.Ctime.Unix())
			data.CreateTime = types.Int64Value(file.Additional.Time.Crtime.Unix())
			continue
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel
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
func (f *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	path := data.Path.ValueString()
	basedir := filepath.Dir(path)
	files, err := f.client.List(ctx, basedir)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list files", fmt.Sprintf("Unable to list files, got error: %s", err))
		return
	}
	for _, file := range files.Files {
		if file.Path == path {
			tflog.Info(ctx, fmt.Sprintf("File found: %s", file.Path))

			data.ModifiedTime = types.Int64Value(file.Additional.Time.Mtime.Unix())
			data.AccessTime = types.Int64Value(file.Additional.Time.Atime.Unix())
			data.ChangeTime = types.Int64Value(file.Additional.Time.Ctime.Unix())
			data.CreateTime = types.Int64Value(file.Additional.Time.Crtime.Unix())
			continue
		}
	}
}

// Update implements resource.Resource.
func (f *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	path := data.Path.ValueString()
	fileName := filepath.Base(data.Path.ValueString())
	fileDir := filepath.Dir(data.Path.ValueString())

	file := form.File{
		Name:    fileName,
		Content: data.Content.ValueString(),
	}

	// Upload the file
	_, err := f.client.Upload(
		ctx,
		fileDir,
		file, data.CreateParents.ValueBool(),
		true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload file", fmt.Sprintf("Unable to upload file, got error: %s", err))
		return
	}

	basedir := filepath.Dir(path)
	files, err := f.client.List(ctx, basedir)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list files", fmt.Sprintf("Unable to list files, got error: %s", err))
		return
	}
	for _, file := range files.Files {
		if file.Path == path {
			tflog.Info(ctx, fmt.Sprintf("File found: %s", file.Path))

			data.ModifiedTime = types.Int64Value(file.Additional.Time.Mtime.Unix())
			data.AccessTime = types.Int64Value(file.Additional.Time.Atime.Unix())
			data.ChangeTime = types.Int64Value(file.Additional.Time.Ctime.Unix())
			data.CreateTime = types.Int64Value(file.Additional.Time.Crtime.Unix())
			continue
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *FileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "file")
}

// Schema implements resource.Resource.
func (f *FileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "FileStation --- A file on the Synology NAS Filestation.",

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
			"access_time": schema.Int64Attribute{
				MarkdownDescription: "The time the file was last accessed.",
				Computed:            true,
			},
			"modified_time": schema.Int64Attribute{
				MarkdownDescription: "The time the file was last modified.",
				Computed:            true,
			},
			"change_time": schema.Int64Attribute{
				MarkdownDescription: "The time the file was last changed.",
				Computed:            true,
			},
			"create_time": schema.Int64Attribute{
				MarkdownDescription: "The time the file was created.",
				Computed:            true,
			},
		},
	}
}

func (f *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	f.client = client.FileStationAPI()
}
