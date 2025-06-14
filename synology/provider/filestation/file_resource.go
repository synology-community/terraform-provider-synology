package filestation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/filestation"
	"github.com/synology-community/go-synology/pkg/util/form"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FileResource{}

func NewFileResource() resource.Resource {
	return &FileResource{}
}

type FileResource struct {
	client filestation.Api
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	Path          types.String      `tfsdk:"path"`
	Content       types.String      `tfsdk:"content"`
	Url           types.String      `tfsdk:"url"`
	CreateParents types.Bool        `tfsdk:"create_parents"`
	Overwrite     types.Bool        `tfsdk:"overwrite"`
	AccessTime    timetypes.RFC3339 `tfsdk:"access_time"`
	ModifiedTime  timetypes.RFC3339 `tfsdk:"modified_time"`
	ChangeTime    timetypes.RFC3339 `tfsdk:"change_time"`
	CreateTime    timetypes.RFC3339 `tfsdk:"create_time"`
	RealPath      types.String      `tfsdk:"real_path"`
	MD5           types.String      `tfsdk:"md5"`
}

// Create implements resource.Resource.
func (f *FileResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
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

	if (!data.Url.IsNull() && !data.Url.IsUnknown()) &&
		(data.Content.IsNull() || data.Content.IsUnknown()) {

		dresp, err := retryablehttp.NewClient().Get(data.Url.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to download file",
				fmt.Sprintf("Unable to download file, got error: %s", err),
			)
			return
		}

		defer func() {
			_ = dresp.Body.Close()
		}()

		dbody, err := io.ReadAll(dresp.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to read file",
				fmt.Sprintf("Unable to read file, got error: %s", err),
			)
			return
		}
		fileContent = string(dbody)
	}

	dctx, cancel := context.WithTimeout(ctx, 120*time.Minute)
	defer cancel()

	// Upload the file
	_, err := f.client.Upload(dctx, fileDir, form.File{
		Name:    fileName,
		Content: fileContent,
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

	if md5, err := f.client.MD5(ctx, path); err == nil {
		data.MD5 = types.StringValue(md5.MD5)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *FileResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data FileResourceModel
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
func (f *FileResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	fPath := data.Path.ValueString()
	file, err := f.client.Get(ctx, fPath)
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

	if data.MD5.IsNull() || data.MD5.IsUnknown() || data.MD5.ValueString() == "" {
		if md5, err := f.client.MD5(ctx, fPath); err == nil {
			data.MD5 = types.StringValue(md5.MD5)
			resp.State.SetAttribute(ctx, path.Root("md5"), data.MD5)
		}
	}
}

// Update implements resource.Resource.
func (f *FileResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data FileResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	path := data.Path.ValueString()
	fileName := filepath.Base(data.Path.ValueString())
	fileDir := filepath.Dir(data.Path.ValueString())
	var fileContent string

	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		fileContent = data.Content.ValueString()
	}

	if (!data.Url.IsNull() && !data.Url.IsUnknown()) &&
		(data.Content.IsNull() || data.Content.IsUnknown()) {

		dresp, err := retryablehttp.NewClient().Get(data.Url.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to download file",
				fmt.Sprintf("Unable to download file, got error: %s", err),
			)
			return
		}

		dbody, err := io.ReadAll(dresp.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to read file",
				fmt.Sprintf("Unable to read file, got error: %s", err),
			)
			return
		}
		fileContent = string(dbody)
	}

	// Upload the file
	_, err := f.client.Upload(
		ctx,
		fileDir,
		form.File{
			Name:    fileName,
			Content: fileContent,
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

	if md5, err := f.client.MD5(ctx, path); err == nil {
		data.MD5 = types.StringValue(md5.MD5)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *FileResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "file")
}

// Schema implements resource.Resource.
func (f *FileResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A file on the Synology NAS Filestation.",

		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "A destination folder path starting with a shared folder to which files can be uploaded.",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The raw file contents to add to the Synology NAS.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("url"),
						path.MatchRoot("content")),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "A file url to download and add to the Synology NAS.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("url"),
						path.MatchRoot("content")),
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
			"md5": schema.StringAttribute{
				MarkdownDescription: "The MD5 hash of the file.",
				Computed:            true,
			},
		},
	}
}

func (f *FileResource) Configure(
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
