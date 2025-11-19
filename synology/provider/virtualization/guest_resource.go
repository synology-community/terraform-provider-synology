package virtualization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/virtualization"
	"github.com/synology-community/terraform-provider-synology/synology/provider/virtualization/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GuestResource{}

func NewGuestResource() resource.Resource {
	return &GuestResource{}
}

type GuestResource struct {
	client virtualization.Api
}

type GuestIsoModel struct {
	ID   types.String `tfsdk:"image_id"`
	Boot types.Bool   `tfsdk:"boot"`
}

// GuestResourceModel describes the resource data model.
type GuestResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	// Description types.String `tfsdk:"description"`
	// Status      types.String `tfsdk:"status"`
	StorageID   types.String `tfsdk:"storage_id"`
	StorageName types.String `tfsdk:"storage_name"`
	// AutoRun     types.Int64  `tfsdk:"autorun"`
	VcpuNum   types.Int64 `tfsdk:"vcpu_num"`
	VramSize  types.Int64 `tfsdk:"vram_size"`
	Disks     types.Set   `tfsdk:"disk"`
	Networks  types.Set   `tfsdk:"network"`
	IsoImages types.Set   `tfsdk:"iso"`
	Run       types.Bool  `tfsdk:"run"`
}

// Schema implements resource.Resource.
func (f *GuestResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages virtual machines on Synology Virtual Machine Manager.

Create and configure virtual machines with custom CPU, memory, disk, and network settings. Supports ISO mounting for installation and cloud-init.

## Example Usage

` + "```hcl" + `
resource "synology_virtualization_guest" "ubuntu_vm" {
  name         = "ubuntu-server"
  storage_name = "default"
  
  vcpu_num  = 2
  vram_size = 2048
  
  network {
    name = "default"
  }
  
  disk {
    size = 20000  # 20GB
  }
  
  iso {
    image_id = synology_virtualization_image.ubuntu_iso.id
    boot     = true
  }
  
  run = true
}
` + "```" + `

See [examples/resources/synology_virtualization_guest](https://github.com/synology-community/terraform-provider-synology/tree/main/examples/resources/synology_virtualization_guest) for more examples.
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the guest.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the guest to upload to the Synology DSM.",
				Required:            true,
			},
			"storage_id": schema.StringAttribute{
				MarkdownDescription: "ID of the storage device.",
				Optional:            true,
			},
			"storage_name": schema.StringAttribute{
				MarkdownDescription: "Name of the storage device.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
			},
			"vcpu_num": schema.Int64Attribute{
				MarkdownDescription: "Number of virtual CPUs.",
				Optional:            true,
				Default:             int64default.StaticInt64(4),
				Computed:            true,
			},
			"vram_size": schema.Int64Attribute{
				MarkdownDescription: "Size of virtual RAM.",
				Optional:            true,
				Default:             int64default.StaticInt64(4096),
				Computed:            true,
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Run the guest.",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"disk": schema.SetNestedBlock{
				MarkdownDescription: "Disks of the guest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							MarkdownDescription: "Size of the disk in MB.",
							Default:             int64default.StaticInt64(20 * 1000),
							Optional:            true,
							Computed:            true,
						},
						"image_id": schema.StringAttribute{
							MarkdownDescription: "ID of the image.",
							Optional:            true,
						},
						"image_name": schema.StringAttribute{
							MarkdownDescription: "Name of the image.",
							Optional:            true,
						},
					},
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"network": schema.SetNestedBlock{
				MarkdownDescription: "Networks of the guest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the network.",
							Optional:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the network.",
							Optional:            true,
							Default:             stringdefault.StaticString("default"),
							Computed:            true,
						},
						"mac": schema.StringAttribute{
							MarkdownDescription: "MAC address.",
							Optional:            true,
						},
					},
				},
			},
			"iso": schema.SetNestedBlock{
				MarkdownDescription: "Mounted ISO files for guest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"image_id": schema.StringAttribute{
							MarkdownDescription: "Image ID for the iso.",
							Required:            true,
						},
						"boot": schema.BoolAttribute{
							MarkdownDescription: "Boot from this iso.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

// Create implements resource.Resource.
func (f *GuestResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data GuestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	guest := virtualization.Guest{
		Name:     data.Name.ValueString(),
		VcpuNum:  data.VcpuNum.ValueInt64(),
		VramSize: data.VramSize.ValueInt64(),
	}

	isoImages := []string{}

	if !data.IsoImages.IsNull() && !data.IsoImages.IsUnknown() {
		isoImages = []string{"unmounted", "unmounted"}
		var elements []GuestIsoModel
		diags := data.IsoImages.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read iso_images", "Unable to read iso_images")
			return
		}

		for _, v := range elements {
			i := 1
			if !v.Boot.IsNull() && !v.Boot.IsUnknown() && v.Boot.ValueBool() {
				i = 0
			}
			isoImages[i] = v.ID.ValueString()
		}
	}

	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {

		var elements []models.VNic
		diags := data.Networks.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read networks", "Unable to read networks")
			return
		}

		for _, v := range elements {
			network := virtualization.VNIC{
				ID:   v.ID.ValueString(),
				Name: v.Name.ValueString(),
				Mac:  v.Mac.ValueString(),
			}

			guest.Networks = append(guest.Networks, network)
		}
	}

	if !data.Disks.IsNull() && !data.Disks.IsUnknown() {

		var elements []VDiskModel
		diags := data.Disks.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read networks", "Unable to read networks")
			return
		}

		for _, v := range elements {

			disk := virtualization.VDisk{}

			if (v.ImageID.IsNull() || v.ImageID.IsUnknown()) &&
				(v.ImageName.IsNull() || v.ImageName.IsUnknown()) {
				disk.CreateType = 0
				disk.Size = v.Size.ValueInt64()
			} else {
				disk.CreateType = 1
				disk.ImageID = v.ImageID.ValueString()
				disk.ImageName = v.ImageName.ValueString()
			}

			guest.Disks = append(guest.Disks, disk)
		}
	}

	if !data.StorageID.IsUnknown() && !data.StorageID.IsNull() {
		guest.StorageID = data.StorageID.ValueString()
	}

	if !data.StorageName.IsUnknown() && !data.StorageName.IsNull() {
		guest.StorageName = data.StorageName.ValueString()
	}

	// Upload the guest
	c, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	res, err := f.client.GuestCreate(c, guest)
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			res, err = f.client.GuestGet(c, virtualization.Guest{Name: data.Name.ValueString()})
			if err != nil {
				resp.Diagnostics.AddError(
					"failed to get guest",
					fmt.Sprintf("unable to get guest, got error: %s", err),
				)
				return
			}
		} else {
			resp.Diagnostics.AddError("failed to create guest guest", fmt.Sprintf("unable to create guest guest, got error: %s", err))
			return
		}
	}

	if res.ID != "" {
		data.ID = types.StringValue(res.ID)
	} else {
		resp.Diagnostics.AddError("Failed to upload guest", "Unable to get guest ID")
		return
	}

	if len(isoImages) == 2 {

		err = f.client.GuestUpdate(c, virtualization.GuestUpdate{
			ID:        data.ID.ValueString(),
			IsoImages: isoImages,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to update guest",
				fmt.Sprintf("Unable to update guest, got error: %s", err),
			)
			return
		}
	}

	if data.Run.ValueBool() {
		_ = f.client.GuestPowerOn(c, virtualization.Guest{Name: data.Name.ValueString()})
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "Guest created")
}

// Delete implements resource.Resource.
func (f *GuestResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data GuestResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_ = f.client.GuestPowerOff(ctx, virtualization.Guest{
		Name: data.Name.ValueString(),
	})

	time.Sleep(5 * time.Second)

	// Start Delete the guest
	if err := f.client.GuestDelete(ctx, virtualization.Guest{
		Name: data.Name.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete guest",
			fmt.Sprintf("Unable to delete guest, got error: %s", err),
		)
		return
	}
}

// Read implements resource.Resource.
func (f *GuestResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data GuestResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	guest, err := f.client.GuestGet(ctx, virtualization.Guest{Name: data.Name.ValueString()})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError("Failed to list guests", fmt.Sprintf("Unable to list guests, got error: %s", err))
			return
		}
	}

	if guest.ID == "" {
		resp.Diagnostics.AddError("Failed to list guests", "Unable to list guests, got empty ID")
		return
	}

	// data.ID = types.StringValue(guest.ID)

	// if guest.ID != "" {
	// 	data.ID = types.StringValue(guest.ID)
	// }

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements resource.Resource.
func (f *GuestResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data GuestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var isoImages []string

	if !data.IsoImages.IsNull() && !data.IsoImages.IsUnknown() {
		isoImages = []string{"unmounted", "unmounted"}
		var elements []GuestIsoModel
		diags := data.IsoImages.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read iso_images", "Unable to read iso_images")
			return
		}

		if len(elements) > 2 {
			resp.Diagnostics.AddError("iso length", "iso length must not be greater than 2")
			return
		}

		for _, v := range elements {
			i := 1
			if !v.Boot.IsNull() && !v.Boot.IsUnknown() && v.Boot.ValueBool() {
				i = 0
			}

			isoImages[i] = v.ID.ValueString()
		}
	}

	err := f.client.GuestUpdate(ctx, virtualization.GuestUpdate{
		ID:        data.ID.ValueString(),
		Name:      data.Name.ValueString(),
		IsoImages: isoImages,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update guest",
			fmt.Sprintf("Unable to update guest, got error: %s", err),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *GuestResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "guest")
}

func (f *GuestResource) Configure(
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

// ValidateConfig.
func (f *GuestResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var data GuestResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Only validate if both values are explicitly null (not set)
	// Allow unknown values to pass validation as they will be resolved during planning
	if data.StorageID.IsNull() && data.StorageName.IsNull() {
		resp.Diagnostics.AddError(
			"At least one of storage_id or storage_name must be set",
			"At least one of storage_id or storage_name must be set",
		)
	}

	if !data.IsoImages.IsNull() && !data.IsoImages.IsUnknown() {
		if len(data.IsoImages.Elements()) > 2 {
			resp.Diagnostics.AddError("iso length", "iso length must not be greater than 2")
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (f *GuestResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	guestName := req.ID

	guest, err := f.client.GuestGet(ctx, virtualization.Guest{Name: guestName})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list guests",
			fmt.Sprintf("Unable to list guests, got error: %s", err),
		)
		return
	}

	id := guest.ID
	storageID := guest.StorageID
	storageName := guest.StorageName

	if id == "" {
		resp.Diagnostics.AddError("Failed to list guests", "Unable to list guests, got empty ID")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), guest.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("storage_id"), storageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("storage_name"), storageName)...)
}
