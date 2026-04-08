package virtualization

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
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
  
  vcpu_num  = 2
  vram_size = 2048
  
  network {}
  
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
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("storage_name")),
				},
			},
			"storage_name": schema.StringAttribute{
				MarkdownDescription: "Name of the storage device.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("storage_id")),
				},
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
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, 8),
				},
			},
			"network": schema.SetNestedBlock{
				MarkdownDescription: "Networks of the guest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the network.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the network.",
							Optional:            true,
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"mac": schema.StringAttribute{
							MarkdownDescription: "MAC address.",
							Optional:            true,
						},
					},
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, 8),
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
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Set{
					setvalidator.SizeAtMost(2),
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
	var plan GuestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Step 1: Resolve storage_name → storage_id via StorageList API.
	storages, err := f.client.StorageList(c)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list storages",
			fmt.Sprintf("Unable to list storages, got error: %s", err),
		)
		return
	}
	storage, diags := resolveStorage(
		storages.Storages,
		plan.StorageID.ValueString(),
		plan.StorageName.ValueString(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.StorageID = types.StringValue(storage.ID)
	plan.StorageName = types.StringValue(storage.Name)
	storageID := storage.ID

	// Step 2: Build vdisks array from config.
	var vdisks virtualization.VDisks
	if !plan.Disks.IsNull() && !plan.Disks.IsUnknown() {
		var elements []VDiskModel
		diags := plan.Disks.ElementsAs(ctx, &elements, true)
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
	if !plan.Networks.IsNull() && !plan.Networks.IsUnknown() {
		var elements []models.VNic
		diags := plan.Networks.ElementsAs(ctx, &elements, true)
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

		var resolvedNics []models.VNic
		for _, v := range elements {
			nic := virtualization.VNIC{
				ID:  v.ID.ValueString(),
				Mac: v.Mac.ValueString(),
			}

			// Resolve network name: "default" → first available, or exact/case-insensitive match.
			netName := v.Name.ValueString()
			if nic.ID == "" {
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
			resolvedNics = append(resolvedNics, models.VNic{
				ID:   types.StringValue(nic.ID),
				Name: types.StringValue(nic.Name),
				Mac:  v.Mac,
			})
		}

		// Write resolved network values back into the plan so state has known values.
		resolvedSet, d := types.SetValueFrom(ctx, plan.Networks.ElementType(ctx), resolvedNics)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Networks = resolvedSet
	}

	// Step 4: Create the guest with ONLY the documented create parameters.
	// Per the Synology VMM API Guide, the create method only accepts:
	// guest_name, storage_id/storage_name, vdisks, vnics, auto_clean_task.
	// Sending additional params like vcpu_num or vram_size causes error 401 (Bad parameter).
	createReq := virtualization.Guest{
		Name:      plan.Name.ValueString(),
		StorageID: storageID,
		Disks:     vdisks,
		Networks:  vnics,
	}

	res, err := f.client.GuestCreate(c, createReq)
	if err != nil {
		// Error 403 = "Name conflict" — the guest already exists.
		if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Name conflict") {
			res, err = f.client.GuestGet(c, virtualization.Guest{Name: plan.Name.ValueString()})
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to get existing guest",
					fmt.Sprintf(
						"Guest with name %q may already exist but could not be retrieved: %s",
						plan.Name.ValueString(),
						err,
					),
				)
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"Failed to create guest",
				fmt.Sprintf("Unable to create guest %q: %s", plan.Name.ValueString(), err),
			)
			return
		}
	}

	if res.ID == "" {
		resp.Diagnostics.AddError("Failed to create guest", "API returned empty guest ID")
		return
	}
	plan.ID = types.StringValue(res.ID)

	// Step 5: Set vcpu_num and vram_size via the documented "set" method.
	// These are not create parameters per the API guide.
	vcpuNum := plan.VcpuNum.ValueInt64()
	vramSize := plan.VramSize.ValueInt64()
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
	isoImages := f.buildIsoImages(ctx, plan)
	if len(isoImages) == 2 {
		err = f.client.GuestUpdate(c, virtualization.GuestUpdate{
			ID:        plan.ID.ValueString(),
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
	if plan.Run.ValueBool() {
		if err := f.client.GuestPowerOn(
			c,
			virtualization.Guest{ID: plan.ID.ValueString()},
		); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Guest created but failed to power on: %s", err))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "Guest created")
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
	// Case-insensitive match.
	if name != "" {
		i := slices.IndexFunc(networks, func(n virtualization.Network) bool {
			return strings.EqualFold(n.Name, name)
		})
		if i != -1 {
			return networks[i].ID, networks[i].Name
		}
	}
	// first available network
	if name == "" && len(networks) > 0 {
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
