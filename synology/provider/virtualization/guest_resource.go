package virtualization

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	Disks       types.Set    `tfsdk:"disks"`
	Networks    types.Set    `tfsdk:"networks"`
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
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the guest.",
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"autorun": schema.Int64Attribute{
				MarkdownDescription: "Determine whether to automatically clean task info when the task finishes. It will be automatically cleaned in a minute after task finishes.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
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
				Default:             int64default.StaticInt64(512),
			},
			"disks": schema.SetAttribute{
				MarkdownDescription: "List of virtual disks.",
				Computed:            true,
				Optional:            true,
				ElementType:         VDiskModel{}.ModelType(),
			},
			"networks": schema.SetAttribute{
				MarkdownDescription: "List of networks.",
				Computed:            true,
				Optional:            true,
				ElementType:         VNicModel{}.ModelType(),
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

	if !data.Description.IsUnknown() && !data.Description.IsNull() {
		guest.Description = data.Description.ValueString()
	}

	if !data.Status.IsUnknown() && !data.Status.IsNull() {
		guest.Status = data.Status.ValueString()
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
