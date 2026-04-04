package virtualization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
				MarkdownDescription: "Number of virtual CPUs. Set via the API `set` method after creation.",
				Optional:            true,
				Default:             int64default.StaticInt64(1),
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"vram_size": schema.Int64Attribute{
				MarkdownDescription: "Size of virtual RAM in MB.",
				Optional:            true,
				Default:             int64default.StaticInt64(1024),
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(256),
				},
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
							MarkdownDescription: "Size of the disk in MB. Must be at least 10240 (10 GB).",
							Default:             int64default.StaticInt64(20 * 1024),
							Optional:            true,
							Computed:            true,
							Validators: []validator.Int64{
								int64validator.AtLeast(10240),
							},
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Step 1: Resolve storage_name → storage_id via StorageList API.
	// The API accepts storage_id or storage_name, but the user-friendly
	// "default" name must be resolved to the actual VMM storage name/ID.
	storageID := data.StorageID.ValueString()
	if storageID == "" {
		storageName := data.StorageName.ValueString()
		if storageName == "" {
			storageName = "default"
		}
		resolvedID, resolvedName, err := f.resolveStorageID(c, storageName)
		if err != nil {
			resp.Diagnostics.AddError("Failed to resolve storage", err.Error())
			return
		}
		storageID = resolvedID
		data.StorageID = types.StringValue(resolvedID)
		data.StorageName = types.StringValue(resolvedName)
	}

	// Step 2: Build vdisks array from config.
	var vdisks virtualization.VDisks
	if !data.Disks.IsNull() && !data.Disks.IsUnknown() {
		var elements []VDiskModel
		diags := data.Disks.ElementsAs(ctx, &elements, true)
		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read disks", "Unable to read disk configuration")
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
			vdisks = append(vdisks, disk)
		}
	}

	// Step 3: Build vnics array from config, resolving network names.
	var vnics virtualization.VNICs
	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {
		var elements []models.VNic
		diags := data.Networks.ElementsAs(ctx, &elements, true)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Failed to read networks",
				"Unable to read network configuration",
			)
			return
		}

		// Pre-fetch the network list for name resolution.
		networks, err := f.client.NetworkList(c)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to list networks",
				fmt.Sprintf("Unable to list VMM networks for name resolution: %s", err),
			)
			return
		}

		for _, v := range elements {
			nic := virtualization.VNIC{
				ID:  v.ID.ValueString(),
				Mac: v.Mac.ValueString(),
			}

			// Resolve network name: "default" → first available, or exact/case-insensitive match.
			netName := v.Name.ValueString()
			if nic.ID == "" && netName != "" {
				resolvedID, resolvedName := resolveNetworkName(netName, networks.Networks)
				if resolvedID == "" {
					available := make([]string, 0, len(networks.Networks))
					for _, n := range networks.Networks {
						available = append(available, fmt.Sprintf("%q", n.Name))
					}
					resp.Diagnostics.AddError(
						"Network not found",
						fmt.Sprintf("Network %q not found. Available networks: %s",
							netName, strings.Join(available, ", ")),
					)
					return
				}
				nic.ID = resolvedID
				nic.Name = resolvedName
			} else if netName != "" {
				nic.Name = netName
			}

			vnics = append(vnics, nic)
		}
	}

	// Step 4: Create the guest with ONLY the documented create parameters.
	// Per the Synology VMM API Guide, the create method only accepts:
	// guest_name, storage_id/storage_name, vdisks, vnics, auto_clean_task.
	// Sending additional params like vcpu_num or vram_size causes error 401 (Bad parameter).
	createReq := virtualization.Guest{
		Name:      data.Name.ValueString(),
		StorageID: storageID,
		Disks:     vdisks,
		Networks:  vnics,
	}

	tflog.Debug(
		ctx,
		fmt.Sprintf(
			"GuestCreate request: guest_name=%s, storage_id=%s, vdisks_len=%d, vnics_len=%d",
			createReq.Name,
			createReq.StorageID,
			len(createReq.Disks),
			len(createReq.Networks),
		),
	)
	for i, d := range createReq.Disks {
		tflog.Debug(
			ctx,
			fmt.Sprintf("  vdisk[%d]: create_type=%d, size=%d, image_id=%s, image_name=%s",
				i, d.CreateType, d.Size, d.ImageID, d.ImageName),
		)
	}
	for i, n := range createReq.Networks {
		tflog.Debug(ctx, fmt.Sprintf("  vnic[%d]: id=%s, name=%s, mac=%s",
			i, n.ID, n.Name, n.Mac))
	}

	res, err := f.client.GuestCreate(c, createReq)
	if err != nil {
		// Error 403 = "Name conflict" — the guest already exists.
		if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Name conflict") {
			res, err = f.client.GuestGet(c, virtualization.Guest{Name: data.Name.ValueString()})
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to get existing guest",
					fmt.Sprintf(
						"Guest with name %q may already exist but could not be retrieved: %s",
						data.Name.ValueString(),
						err,
					),
				)
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"Failed to create guest",
				fmt.Sprintf("Unable to create guest %q: %s", data.Name.ValueString(), err),
			)
			return
		}
	}

	if res.ID == "" {
		resp.Diagnostics.AddError("Failed to create guest", "API returned empty guest ID")
		return
	}
	data.ID = types.StringValue(res.ID)

	// Step 5: Set vcpu_num and vram_size via the documented "set" method.
	// These are not create parameters per the API guide.
	vcpuNum := data.VcpuNum.ValueInt64()
	vramSize := data.VramSize.ValueInt64()
	if vcpuNum > 0 || vramSize > 0 {
		setReq := virtualization.GuestUpdate{
			ID: res.ID,
		}
		if vcpuNum > 0 {
			setReq.VcpuNum = vcpuNum
		}
		if vramSize > 0 {
			setReq.VramSize = vramSize
		}
		if err := f.client.GuestUpdate(c, setReq); err != nil {
			resp.Diagnostics.AddError(
				"Failed to configure guest CPU/RAM",
				fmt.Sprintf("Guest was created but CPU/RAM configuration failed: %s", err),
			)
			return
		}
	}

	// Step 6: Mount ISO images if configured (via the internal set API).
	isoImages := f.buildIsoImages(ctx, data)
	if len(isoImages) == 2 {
		err = f.client.GuestUpdate(c, virtualization.GuestUpdate{
			ID:        data.ID.ValueString(),
			IsoImages: isoImages,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to mount ISO images",
				fmt.Sprintf("Guest was created but ISO mounting failed: %s", err),
			)
			return
		}
	}

	// Step 7: Power on if requested.
	if data.Run.ValueBool() {
		if err := f.client.GuestPowerOn(
			c,
			virtualization.Guest{ID: data.ID.ValueString()},
		); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Guest created but failed to power on: %s", err))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, "Guest created")
}

// resolveStorageID resolves a user-friendly storage name (like "default") to
// the actual VMM storage ID and name by querying the StorageList API.
func (f *GuestResource) resolveStorageID(
	ctx context.Context,
	storageName string,
) (string, string, error) {
	storages, err := f.client.StorageList(ctx)
	if err != nil {
		return "", "", fmt.Errorf("unable to list storages: %s", err)
	}
	if len(storages.Storages) == 0 {
		return "", "", fmt.Errorf(
			"no storages found in VMM — ensure Virtual Machine Manager has at least one storage configured",
		)
	}

	// Try exact match first.
	for _, s := range storages.Storages {
		if s.Name == storageName {
			return s.ID, s.Name, nil
		}
	}

	// Try case-insensitive substring match (e.g. "default" matches "Synology - VM Storage 1"
	// only if that's the first/only storage, or matches "Default VM Storage").
	for _, s := range storages.Storages {
		if strings.EqualFold(s.Name, storageName) {
			return s.ID, s.Name, nil
		}
	}

	// If "default" was requested, fall back to first available storage.
	if strings.EqualFold(storageName, "default") && len(storages.Storages) > 0 {
		s := storages.Storages[0]
		return s.ID, s.Name, nil
	}

	available := make([]string, 0, len(storages.Storages))
	for _, s := range storages.Storages {
		available = append(available, fmt.Sprintf("%q (id: %s)", s.Name, s.ID))
	}
	return "", "", fmt.Errorf("storage %q not found. Available storages: %s",
		storageName, strings.Join(available, ", "))
}

// buildIsoImages converts the ISO block config into the ["image_id", "unmounted"] array
// expected by the GuestUpdate API. Returns nil if no ISOs configured.
func (f *GuestResource) buildIsoImages(ctx context.Context, data GuestResourceModel) []string {
	if data.IsoImages.IsNull() || data.IsoImages.IsUnknown() {
		return nil
	}

	var elements []GuestIsoModel
	d := data.IsoImages.ElementsAs(ctx, &elements, true)
	if d.HasError() {
		return nil
	}

	isoImages := []string{"unmounted", "unmounted"}
	for _, v := range elements {
		i := 1
		if !v.Boot.IsNull() && !v.Boot.IsUnknown() && v.Boot.ValueBool() {
			i = 0
		}
		isoImages[i] = v.ID.ValueString()
	}
	return isoImages
}

// resolveNetworkName resolves a user-friendly network name (like "default") to the
// actual VMM network ID and name from a pre-fetched list of networks.
func resolveNetworkName(name string, networks []virtualization.Network) (string, string) {
	// Exact match.
	for _, n := range networks {
		if n.Name == name {
			return n.ID, n.Name
		}
	}
	// Case-insensitive match.
	for _, n := range networks {
		if strings.EqualFold(n.Name, name) {
			return n.ID, n.Name
		}
	}
	// "default" → first available network.
	if strings.EqualFold(name, "default") && len(networks) > 0 {
		return networks[0].ID, networks[0].Name
	}
	return "", ""
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
			resp.Diagnostics.AddError(
				"Failed to list guests",
				fmt.Sprintf("Unable to list guests, got error: %s", err),
			)
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

// ValidateConfig validates the guest resource configuration at plan time.
func (f *GuestResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var data GuestResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Storage: at least one of storage_id or storage_name must be set.
	if data.StorageID.IsNull() && data.StorageName.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("storage_id"),
			"Missing storage configuration",
			"At least one of storage_id or storage_name must be set.",
		)
	}

	// ISO: maximum 2 ISO images (boot + secondary).
	if !data.IsoImages.IsNull() && !data.IsoImages.IsUnknown() {
		if len(data.IsoImages.Elements()) > 2 {
			resp.Diagnostics.AddAttributeError(
				path.Root("iso"),
				"Too many ISO images",
				"A maximum of 2 ISO images can be mounted on a virtual machine.",
			)
		}
	}

	// Disk validation: check create_type requirements.
	if !data.Disks.IsNull() && !data.Disks.IsUnknown() {
		var disks []VDiskModel
		diags := data.Disks.ElementsAs(ctx, &disks, true)
		if !diags.HasError() {
			for i, d := range disks {
				hasImage := (!d.ImageID.IsNull() && !d.ImageID.IsUnknown() && d.ImageID.ValueString() != "") ||
					(!d.ImageName.IsNull() && !d.ImageName.IsUnknown() && d.ImageName.ValueString() != "")

				if !hasImage {
					// create_type=0: empty disk — size must be > 0
					size := d.Size.ValueInt64()
					if !d.Size.IsNull() && !d.Size.IsUnknown() && size <= 0 {
						resp.Diagnostics.AddAttributeError(
							path.Root("disk"),
							fmt.Sprintf("Invalid disk size (disk %d)", i),
							"Disk size must be greater than 0 MB when creating an empty disk.",
						)
					}
				}
			}
		}
	}

	// Network validation: at least network_id or network_name per vnic.
	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {
		var nics []models.VNic
		diags := data.Networks.ElementsAs(ctx, &nics, true)
		if !diags.HasError() {
			for i, n := range nics {
				hasID := !n.ID.IsNull() && !n.ID.IsUnknown() && n.ID.ValueString() != ""
				hasName := !n.Name.IsNull() && !n.Name.IsUnknown() && n.Name.ValueString() != ""
				if !hasID && !hasName {
					resp.Diagnostics.AddAttributeError(
						path.Root("network"),
						fmt.Sprintf("Missing network identifier (network %d)", i),
						"Each network block must specify at least one of id or name.",
					)
				}
			}
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
