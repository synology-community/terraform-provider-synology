package filestation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/filestation"
	"github.com/synology-community/go-synology/pkg/util/form"
	"github.com/synology-community/terraform-provider-synology/synology/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IsoResource{}

func NewIsoResource() resource.Resource {
	return &IsoResource{}
}

type IsoResource struct {
	client filestation.Api
}

// IsoResourceModel describes the resource data model.
type IsoResourceModel struct {
	Path          types.String      `tfsdk:"path"`
	VolumeName    types.String      `tfsdk:"volume_name"`
	Files         types.List        `tfsdk:"files"`
	CreateParents types.Bool        `tfsdk:"create_parents"`
	Overwrite     types.Bool        `tfsdk:"overwrite"`
	AccessTime    timetypes.RFC3339 `tfsdk:"access_time"`
	ModifiedTime  timetypes.RFC3339 `tfsdk:"modified_time"`
	ChangeTime    timetypes.RFC3339 `tfsdk:"change_time"`
	CreateTime    timetypes.RFC3339 `tfsdk:"create_time"`
	RealPath      types.String      `tfsdk:"real_path"`
}

// Create implements resource.Resource.
func (f *IsoResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data IsoResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createParents := data.CreateParents.ValueBool()
	overwrite := data.Overwrite.ValueBool()
	path := data.Path.ValueString()
	fileName := filepath.Base(data.Path.ValueString())
	fileDir := filepath.Dir(data.Path.ValueString())

	dataFiles := []File{}
	resp.Diagnostics.Append(data.Files.ElementsAs(ctx, &dataFiles, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isoFiles := map[string]string{}
	for _, file := range dataFiles {
		if file.Path == "" {
			resp.Diagnostics.AddError("Invalid file path", "The file path must be set")
			return
		}
		if file.Content == "" {
			resp.Diagnostics.AddError("Invalid file content", "The file content must be set")
			return
		}
		isoFiles[file.Path] = file.Content
	}

	iso, err := util.IsoFromFiles(ctx, data.VolumeName.ValueString(), isoFiles)
	if err != nil {
		resp.Diagnostics.AddError("failed to create ISO", err.Error())
		return
	}

	// Upload the file
	_, err = f.client.Upload(ctx, fileDir, form.File{
		Name:    fileName,
		Content: iso,
	}, createParents, overwrite)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to upload file",
			fmt.Sprintf("Unable to upload file, got error: %s", err),
		)
		return
	}

	file, err := f.client.Get(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get file",
			fmt.Sprintf("Unable to get file, got error: %s", err),
		)
		return
	}
	data.ModifiedTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Mtime.Time)
	data.AccessTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Atime.Time)
	data.ChangeTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Ctime.Time)
	data.CreateTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Crtime.Time)
	data.RealPath = types.StringValue(file.Additional.RealPath)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *IsoResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data IsoResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	path := data.Path.ValueString()
	// Start Delete the file
	_, err := f.client.Delete(ctx, []string{path}, true)
	if err != nil {
		if e := errors.Unwrap(err); e != nil {
			resp.Diagnostics.AddError(
				"Failed to delete file",
				fmt.Sprintf("Unable to delete file, got error: %s", e),
			)
		} else {
			resp.Diagnostics.AddError("Failed to delete file", fmt.Sprintf("Unable to delete file, got error: %s", err))
		}
		return
	}
}

// Read implements resource.Resource.
func (f *IsoResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data IsoResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	path := data.Path.ValueString()
	file, err := f.client.Get(ctx, path)
	if err != nil {
		switch err.Error() {
		case "Result is empty":
			resp.State.RemoveResource(ctx)
			return
		default:
			resp.Diagnostics.AddError(
				"Failed to get file",
				fmt.Sprintf("Unable to get file, got error: %s", err),
			)
			return
		}
	}
	tflog.Info(ctx, fmt.Sprintf("File found: %s", file.Path))

	data.ModifiedTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Mtime.Time)
	data.AccessTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Atime.Time)
	data.ChangeTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Ctime.Time)
	data.CreateTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Crtime.Time)
	data.RealPath = types.StringValue(file.Additional.RealPath)
}

// Update implements resource.Resource.
func (f *IsoResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data IsoResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	path := data.Path.ValueString()
	fileName := filepath.Base(data.Path.ValueString())
	fileDir := filepath.Dir(data.Path.ValueString())

	dataFiles := []File{}
	resp.Diagnostics.Append(data.Files.ElementsAs(ctx, &dataFiles, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isoFiles := map[string]string{}
	for _, file := range dataFiles {
		if file.Path == "" {
			resp.Diagnostics.AddError("Invalid file path", "The file path must be set")
			return
		}
		if file.Content == "" {
			resp.Diagnostics.AddError("Invalid file content", "The file content must be set")
			return
		}
		isoFiles[file.Path] = file.Content
	}

	iso, err := util.IsoFromFiles(ctx, data.VolumeName.ValueString(), isoFiles)
	if err != nil {
		resp.Diagnostics.AddError("failed to create ISO", err.Error())
		return
	}

	// Upload the file
	_, err = f.client.Upload(ctx, fileDir, form.File{
		Name:    fileName,
		Content: iso,
	}, data.CreateParents.ValueBool(),
		true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to upload file",
			fmt.Sprintf("Unable to upload file, got error: %s", err),
		)
		return
	}

	file, err := f.client.Get(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get file",
			fmt.Sprintf("Unable to get file, got error: %s", err),
		)
		return
	}
	data.ModifiedTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Mtime.Time)
	data.AccessTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Atime.Time)
	data.ChangeTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Ctime.Time)
	data.CreateTime = timetypes.NewRFC3339TimeValue(file.Additional.Time.Crtime.Time)
	data.RealPath = types.StringValue(file.Additional.RealPath)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *IsoResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "iso")
}

// Schema implements resource.Resource.
func (f *IsoResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A file on the Synology NAS Filestation.",

		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "The path on the synology server to upload the ISO file.",
				Required:            true,
			},
			"volume_name": schema.StringAttribute{
				MarkdownDescription: "The name of the volume for the iso partition where the files are stored.",
				Required:            true,
			},
			"files": schema.ListNestedAttribute{
				MarkdownDescription: "A map of target file paths and the file content to add to the ISO file.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							MarkdownDescription: "The path of the file to be uploaded.",
							Required:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "The content of the file to be uploaded.",
							Required:            true,
						},
					},
				},
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
			"access_time": schema.StringAttribute{
				MarkdownDescription: "The time the file was last accessed.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"modified_time": schema.StringAttribute{
				MarkdownDescription: "The time the file was last modified.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"change_time": schema.StringAttribute{
				MarkdownDescription: "The time the file was last changed.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The time the file was created.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"real_path": schema.StringAttribute{
				MarkdownDescription: "The real path of the folder.",
				Computed:            true,
			},
		},
	}
}

func (f *IsoResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Api)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)

		return
	}

	f.client = client.FileStationAPI()
}

func (f *IsoResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	p := req.ID
	basedir := filepath.Dir(p)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), p)...)

	files, err := f.client.List(ctx, basedir)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list files",
			fmt.Sprintf("Unable to list files, got error: %s", err),
		)
		return
	}
	for _, file := range files.Files {
		if file.Path == p {
			tflog.Info(ctx, fmt.Sprintf("File found: %s", file.Path))

			resp.Diagnostics.Append(
				resp.State.SetAttribute(
					ctx,
					path.Root("modified_time"),
					file.Additional.Time.Mtime,
				)...)
			resp.Diagnostics.Append(
				resp.State.SetAttribute(
					ctx,
					path.Root("access_time"),
					file.Additional.Time.Atime,
				)...)
			resp.Diagnostics.Append(
				resp.State.SetAttribute(
					ctx,
					path.Root("change_time"),
					file.Additional.Time.Ctime,
				)...)
			resp.Diagnostics.Append(
				resp.State.SetAttribute(
					ctx,
					path.Root("create_time"),
					file.Additional.Time.Crtime,
				)...)
			continue
		}
	}
}
