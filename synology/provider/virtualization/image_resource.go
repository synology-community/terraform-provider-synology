package virtualization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/pkg"
	"github.com/synology-community/synology-api/pkg/api/virtualization"
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
	AutoClean   types.Bool   `tfsdk:"auto_clean"`
	ImageType   types.String `tfsdk:"image_type"`
	StorageID   types.String `tfsdk:"storage_id"`
	StorageName types.String `tfsdk:"storage_name"`
}

// Schema implements resource.Resource.
func (f *ImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Virtualization --- A image on the Synology NAS Imagestation.",

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
				MarkdownDescription: "The file on the DiskStation. Note: the path should begin with a shared folder.",
				Required:            true,
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
				MarkdownDescription: "ID of the storage device.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"storage_name": schema.StringAttribute{
				MarkdownDescription: "Name of the storage device.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
			},
		},
	}
}

// Create implements resource.Resource.
func (f *ImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ImageResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	image := virtualization.Image{
		Name:      data.Name.ValueString(),
		FilePath:  data.Path.ValueString(),
		AutoClean: data.AutoClean.ValueBool(),
		Type:      virtualization.ImageType(data.ImageType.ValueString()),
	}

	if !data.StorageID.IsUnknown() && !data.StorageID.IsNull() {
		image.Storages = append(image.Storages, virtualization.Storage{ID: data.StorageID.ValueString()})
	}

	if !data.StorageName.IsUnknown() && !data.StorageName.IsNull() {
		image.Storages = append(image.Storages, virtualization.Storage{Name: data.StorageName.ValueString()})
	}

	// Upload the image
	c, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	res, err := f.client.ImageCreate(c, image)
	if err != nil {

		if strings.Contains(err.Error(), "403") {
			img, err := f.getImage(ctx, data.Name.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Failed to list images", fmt.Sprintf("Unable to list images, got error: %s", err))
				return
			}
			if img.ID != "" {
				data.ID = types.StringValue(img.ID)
			}
		} else {
			resp.Diagnostics.AddError("failed to create guest image", fmt.Sprintf("unable to create guest image, got error: %s", err))
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *ImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ImageResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	imageName := data.Name.ValueString()

	// Start Delete the image
	if err := f.client.ImageDelete(ctx, imageName); err != nil {
		resp.Diagnostics.AddError("Failed to delete image", fmt.Sprintf("Unable to delete image, got error: %s", err))
		return
	}
}

// Read implements resource.Resource.
func (f *ImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ImageResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	image, err := f.getImage(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list images", fmt.Sprintf("Unable to list images, got error: %s", err))
		return
	}

	if image.ID != "" {
		data.ID = types.StringValue(image.ID)
	}

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
func (f *ImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// var data ImageResourceModel

	// Read Terraform configuration data into the model
	// resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Save data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *ImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "image")
}

func (f *ImageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	f.client = client.VirtualizationAPI()
}
