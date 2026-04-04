package virtualization

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/virtualization"
	"github.com/synology-community/go-synology/pkg/util/form"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ImageResource{}

func NewImageResource() resource.Resource {
	return &ImageResource{}
}

type ImageResource struct {
	client virtualization.Api
}

// ImageResourceModel describes the resource data model.
type ImageResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	Content     types.String `tfsdk:"content"`
	Url         types.String `tfsdk:"url"`
	AutoClean   types.Bool   `tfsdk:"auto_clean"`
	ImageType   types.String `tfsdk:"image_type"`
	StorageID   types.String `tfsdk:"storage_id"`
	StorageName types.String `tfsdk:"storage_name"`
	UsedSize    types.Int64  `tfsdk:"used_size"`
}

// Schema implements resource.Resource.
func (f *ImageResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A image on the Synology NAS Imagestation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the image.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the image to upload to the Synology DSM.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The file on the DiskStation. Note: the path should begin with a shared folder. Use this to create an image from an existing file on the NAS.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("path"),
						path.MatchRoot("content"),
						path.MatchRoot("url"),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The raw file contents to upload as a guest image.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "A URL to download and upload as a guest image.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_type": schema.StringAttribute{
				MarkdownDescription: "The image type. (disk/vdsm/iso)",
				Required:            true,
			},
			"auto_clean": schema.BoolAttribute{
				MarkdownDescription: "Determine whether to automatically clean task info when the task finishes. It will be automatically cleaned in a minute after task finishes.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"storage_id": schema.StringAttribute{
				MarkdownDescription: "ID of the storage device. If not specified, it will be resolved from storage_name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"storage_name": schema.StringAttribute{
				MarkdownDescription: "Name of the storage device.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
			},
			"used_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the image in bytes as reported by the server. Changes to this value indicate the image needs replacing.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create implements resource.Resource.
func (f *ImageResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data ImageResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	c, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	isUpload := (!data.Content.IsNull() && !data.Content.IsUnknown()) ||
		(!data.Url.IsNull() && !data.Url.IsUnknown())

	if isUpload {
		// Upload mode: upload file content to create a new guest image
		var fileContent string

		if !data.Content.IsNull() && !data.Content.IsUnknown() {
			fileContent = data.Content.ValueString()
		} else {
			dresp, err := retryablehttp.NewClient().Get(data.Url.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to download file",
					fmt.Sprintf("Unable to download file from URL, got error: %s", err),
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
					fmt.Sprintf("Unable to read downloaded file, got error: %s", err),
				)
				return
			}
			fileContent = string(dbody)
		}

		fileName := data.Name.ValueString() + "." + data.ImageType.ValueString()

		var imageRepos []string
		if !data.StorageID.IsNull() && !data.StorageID.IsUnknown() &&
			data.StorageID.ValueString() != "" {
			imageRepos = []string{data.StorageID.ValueString()}
		} else {
			// Resolve storage name to storage ID
			storageName := "default"
			if !data.StorageName.IsNull() && !data.StorageName.IsUnknown() &&
				data.StorageName.ValueString() != "" {
				storageName = data.StorageName.ValueString()
			}
			storages, err := f.client.StorageList(ctx)
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to list storages",
					fmt.Sprintf(
						"Unable to list storages to resolve storage name, got error: %s",
						err,
					),
				)
				return
			}
			for _, s := range storages.Storages {
				if s.Name == storageName {
					imageRepos = []string{s.ID}
					data.StorageID = types.StringValue(s.ID)
					break
				}
			}
			if len(imageRepos) == 0 && len(storages.Storages) > 0 {
				// Fall back to first available storage
				s := storages.Storages[0]
				imageRepos = []string{s.ID}
				data.StorageID = types.StringValue(s.ID)
			}
			if len(imageRepos) == 0 {
				resp.Diagnostics.AddError(
					"Storage not found",
					fmt.Sprintf(
						"Unable to find storage. Specify storage_id directly or ensure VMM has configured storages.",
					),
				)
				return
			}
		}

		res, err := f.client.ImageUploadAndCreate(c, form.File{
			Name:    fileName,
			Content: fileContent,
		}, imageRepos, data.ImageType.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to upload and create guest image",
				fmt.Sprintf("Unable to upload and create guest image, got error: %s", err),
			)
			return
		}

		if res.TaskInfo.ImageID != "" {
			data.ID = types.StringValue(res.TaskInfo.ImageID)
		} else {
			// Task completed but image_id not in task info; look up by name
			img, err := f.getImage(ctx, data.Name.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to find uploaded image",
					fmt.Sprintf(
						"Image upload task completed but image not found by name, got error: %s",
						err,
					),
				)
				return
			}
			data.ID = types.StringValue(img.ID)
		}
	} else {
		// Existing file mode: create image from a file already on the DiskStation
		image := virtualization.Image{
			Name:      data.Name.ValueString(),
			FilePath:  data.Path.ValueString(),
			AutoClean: data.AutoClean.ValueBool(),
			Type:      virtualization.ImageType(data.ImageType.ValueString()),
		}

		if !data.StorageID.IsUnknown() && !data.StorageID.IsNull() {
			image.Storages = append(
				image.Storages,
				virtualization.Storage{ID: data.StorageID.ValueString()},
			)
		}

		if !data.StorageName.IsUnknown() && !data.StorageName.IsNull() {
			image.Storages = append(
				image.Storages,
				virtualization.Storage{Name: data.StorageName.ValueString()},
			)
		}

		res, err := f.client.ImageCreate(c, image)
		if err != nil {
			if strings.Contains(err.Error(), "403") {
				img, err := f.getImage(ctx, data.Name.ValueString())
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to list images",
						fmt.Sprintf("Unable to list images, got error: %s", err),
					)
					return
				}
				if img.ID != "" {
					data.ID = types.StringValue(img.ID)
				}
			} else {
				resp.Diagnostics.AddError(
					"failed to create guest image",
					fmt.Sprintf("unable to create guest image, got error: %s", err),
				)
				return
			}
		} else {
			if res.TaskInfo.ImageID != "" {
				data.ID = types.StringValue(res.TaskInfo.ImageID)
			} else {
				resp.Diagnostics.AddError("Failed to upload image", "Unable to get image ID")
				return
			}
		}
	}

	// Read back the image to get used_size
	img, err := f.getImage(ctx, data.Name.ValueString())
	if err == nil && img != nil {
		data.UsedSize = types.Int64Value(img.UsedSize)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *ImageResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ImageResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	imageName := data.Name.ValueString()

	// Start Delete the image
	if err := f.client.ImageDelete(ctx, imageName); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete image",
			fmt.Sprintf("Unable to delete image, got error: %s", err),
		)
		return
	}
}

// Read implements resource.Resource.
func (f *ImageResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data ImageResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	image, err := f.getImage(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list images",
			fmt.Sprintf("Unable to list images, got error: %s", err),
		)
		return
	}

	if image.ID != "" {
		data.ID = types.StringValue(image.ID)
	}

	data.UsedSize = types.Int64Value(image.UsedSize)

	resp.State.Set(ctx, &data)
}

func (f *ImageResource) getImage(ctx context.Context, name string) (*virtualization.Image, error) {
	images, err := f.client.ImageList(ctx)
	if err != nil {
		return nil, err
	}

	for _, image := range images.Images {
		if image.Name == name {
			return &image, nil
		}
	}

	return nil, fmt.Errorf("image %s not found", name)
}

// Update implements resource.Resource.
func (f *ImageResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	// var data ImageResourceModel

	// Read Terraform configuration data into the model
	// resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Save data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *ImageResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "image")
}

func (f *ImageResource) Configure(
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

	f.client = client.VirtualizationAPI()
}
