package virtualization

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/synology-api/pkg"
	"github.com/synology-community/synology-api/pkg/api/virtualization"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GuestResource{}

func NewGuestResource() resource.Resource {
	return &GuestResource{}
}

type GuestResource struct {
	client virtualization.VirtualizationAPI
}

func (d GuestResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("storage_id"),
			path.MatchRoot("storage_name"),
		),
	}
}

// GuestResourceModel describes the resource data model.
type GuestResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	StorageID   types.String `tfsdk:"storage_id"`
	StorageName types.String `tfsdk:"storage_name"`
	AutoRun     types.Int64  `tfsdk:"autorun"`
	VcpuNum     types.Int64  `tfsdk:"vcpu_num"`
	VramSize    types.Int64  `tfsdk:"vram_size"`
	Disks       types.Set    `tfsdk:"disk"`
	Networks    types.Set    `tfsdk:"network"`
}

// Schema implements resource.Resource.
func (f *GuestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A guest on the Synology NAS Gueststation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the guest.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the guest to upload to the Synology DSM.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the guest.",
				Computed:            true,
				Optional:            true,
			},
			"autorun": schema.Int64Attribute{
				MarkdownDescription: "Determine whether to automatically clean task info when the task finishes. It will be automatically cleaned in a minute after task finishes.",
				Optional:            true,
				Computed:            true,
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
			"vcpu_num": schema.Int64Attribute{
				MarkdownDescription: "Number of virtual CPUs.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(4),
			},
			"vram_size": schema.Int64Attribute{
				MarkdownDescription: "Size of virtual RAM.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(4096),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the guest.",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"disk": schema.SetNestedBlock{
				MarkdownDescription: "Disks of the guest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the network.",
							Computed:            true,
							Default:             stringdefault.StaticString(""),
						},
						"create_type": schema.Int64Attribute{
							MarkdownDescription: "Type of the disk.",
							Computed:            true,
							Optional:            true,
						},
						"size": schema.Int64Attribute{
							MarkdownDescription: "Size of the disk in MB.",
							Computed:            true,
							Default:             int64default.StaticInt64(20 * 1000),
							Optional:            true,
						},
						"image_id": schema.StringAttribute{
							MarkdownDescription: "ID of the image.",
							Computed:            true,
							Optional:            true,
							Default:             stringdefault.StaticString(""),
						},
						"image_name": schema.StringAttribute{
							MarkdownDescription: "Name of the image.",
							Computed:            true,
							Optional:            true,
							Default:             stringdefault.StaticString(""),
						},
						"unmap": schema.BoolAttribute{
							MarkdownDescription: "Unmap the disk.",
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"controller": schema.Int64Attribute{
							MarkdownDescription: "Controller of the disk.",
							Computed:            true,
							Default:             int64default.StaticInt64(0),
						},
					},
				},
			},
			"network": schema.SetNestedBlock{
				MarkdownDescription: "Networks of the guest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the network.",
							Computed:            true,
							Optional:            true,
							Default:             stringdefault.StaticString(""),
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the network.",
							Computed:            true,
							Optional:            true,
							Default:             stringdefault.StaticString(""),
						},
						"mac": schema.StringAttribute{
							MarkdownDescription: "MAC address.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(""),
						},
						"model": schema.Int64Attribute{
							MarkdownDescription: "Model of the network.",
							Computed:            true,
							Default:             int64default.StaticInt64(0),
						},
						"vnic_id": schema.StringAttribute{
							MarkdownDescription: "Virtual NIC ID.",
							Computed:            true,
							Default:             stringdefault.StaticString(""),
						},
					},
				},
			},
		},
	}
}

// Create implements resource.Resource.
func (f *GuestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {

		var elements []VNicModel
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

			disk := virtualization.VDisk{
				// ID:         v.ID.ValueString(),
				CreateType: v.CreateType.ValueInt64(),
				Size:       v.Size.ValueInt64(),
				ImageID:    v.ImageID.ValueString(),
				ImageName:  v.ImageName.ValueString(),
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
		resp.Diagnostics.AddError("failed to create guest guest", fmt.Sprintf("unable to create guest guest, got error: %s", err))
		return
	}

	if res.ID != "" {
		data.ID = types.StringValue(res.ID)
	} else {
		resp.Diagnostics.AddError("Failed to upload guest", "Unable to get guest ID")
		return
	}

	guestrefresh, err := f.client.GuestGet(ctx, virtualization.Guest{Name: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list guests", fmt.Sprintf("Unable to list guests, got error: %s", err))
		return
	}

	data.Status = types.StringValue(guestrefresh.Status)
	data.AutoRun = types.Int64Value(guestrefresh.AutoRun)
	data.Description = types.StringValue(guestrefresh.Description)

	if !data.Networks.IsNull() && !data.Networks.IsUnknown() {

		var elements []VNicModel
		diags := data.Networks.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read networks", "Unable to read networks")
			return
		}

		if len(elements) <= len(guestrefresh.Networks) {
			for i, v := range elements {
				newDisk := guestrefresh.Networks[i]
				v.ID = types.StringValue(newDisk.ID)
				v.Mac = types.StringValue(newDisk.Mac)
				v.Model = types.Int64Value(newDisk.Model)
				v.VNicID = types.StringValue(newDisk.VnicID)

			}
		}

		setValue, diags := types.SetValueFrom(ctx, VNicModel{}.ModelType(), elements)
		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to set disks", "Unable to set disks")
			return
		}
		data.Networks = setValue
	}

	if !data.Disks.IsNull() && !data.Disks.IsUnknown() {

		var elements []VDiskModel
		diags := data.Disks.ElementsAs(ctx, &elements, true)

		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to read networks", "Unable to read networks")
			return
		}

		if len(elements) <= len(guestrefresh.Disks) {
			for i, v := range elements {
				newDisk := guestrefresh.Disks[i]
				v.ID = types.StringValue(newDisk.ID)
				v.Unmap = types.BoolValue(newDisk.Unmap)
				v.Controller = types.Int64Value(newDisk.Controller)
				v.ImageID = types.StringValue(newDisk.ImageID)
				v.ImageName = types.StringValue(newDisk.ImageName)
			}
		}

		setValue, diags := types.SetValueFrom(ctx, VDiskModel{}.ModelType(), elements)
		if diags.HasError() {
			resp.Diagnostics.AddError("Failed to set disks", "Unable to set disks")
			return
		}
		data.Disks = setValue
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (f *GuestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GuestResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Start Delete the guest
	if err := f.client.GuestDelete(ctx, virtualization.Guest{
		Name: data.Name.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to delete guest", fmt.Sprintf("Unable to delete guest, got error: %s", err))
		return
	}
}

// Read implements resource.Resource.
func (f *GuestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GuestResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	guest, err := f.client.GuestGet(ctx, virtualization.Guest{Name: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list guests", fmt.Sprintf("Unable to list guests, got error: %s", err))
		return
	}

	data.ID = types.StringValue(guest.ID)

	resp.State.Set(ctx, &data)
}

// Update implements resource.Resource.
func (f *GuestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// var data GuestResourceModel

	// Read Terraform configuration data into the model
	// resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Save data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (f *GuestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "guest")
}

func (f *GuestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
