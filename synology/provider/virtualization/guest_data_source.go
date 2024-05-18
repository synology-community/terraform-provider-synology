package virtualization

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	client "github.com/synology-community/synology-api/pkg"
	"github.com/synology-community/synology-api/pkg/api/virtualization"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GuestDataSource{}

func NewGuestDataSource() datasource.DataSource {
	return &GuestDataSource{}
}

type GuestDataSource struct {
	client client.SynologyClient
}

type VDiskDataModel struct {
	ID         types.String `tfsdk:"id"`
	Size       types.Int64  `tfsdk:"size"`
	Controller types.Int64  `tfsdk:"controller"`
	Unmap      types.Bool   `tfsdk:"unmap"`
}

func (m VDiskDataModel) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VDiskDataModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         types.StringType,
		"size":       types.Int64Type,
		"controller": types.Int64Type,
		"unmap":      types.BoolType,
	}
}

func (m VDiskDataModel) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"id":         types.StringValue(m.ID.ValueString()),
		"size":       types.Int64Value(m.Size.ValueInt64()),
		"controller": types.Int64Value(m.Controller.ValueInt64()),
		"unmap":      types.BoolValue(m.Unmap.ValueBool()),
	})
}

type VNicDataModel struct {
	ID     types.String `tfsdk:"id"`
	Mac    types.String `tfsdk:"mac"`
	Model  types.Int64  `tfsdk:"model"`
	Name   types.String `tfsdk:"name"`
	VNicID types.String `tfsdk:"vnic_id"`
}

func (m VNicDataModel) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m VNicDataModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":      types.StringType,
		"name":    types.StringType,
		"mac":     types.StringType,
		"model":   types.Int64Type,
		"vnic_id": types.StringType,
	}
}

func (m VNicDataModel) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"id":      types.StringValue(m.ID.ValueString()),
		"name":    types.StringValue(m.Name.ValueString()),
		"mac":     types.StringValue(m.Mac.ValueString()),
		"model":   types.Int64Value(m.Model.ValueInt64()),
		"vnic_id": types.StringValue(m.VNicID.ValueString()),
	})
}

type GuestDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	StorageID   types.String `tfsdk:"storage_id"`
	StorageName types.String `tfsdk:"storage_name"`
	Autorun     types.Int64  `tfsdk:"autorun"`
	VcpuNum     types.Int64  `tfsdk:"vcpu_num"`
	VramSize    types.Int64  `tfsdk:"vram_size"`
	Disks       types.Set    `tfsdk:"disks"`
	Networks    types.Set    `tfsdk:"networks"`
}

func (m GuestDataSourceModel) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m GuestDataSourceModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":           types.StringType,
		"name":         types.StringType,
		"description":  types.StringType,
		"status":       types.StringType,
		"storage_id":   types.StringType,
		"storage_name": types.StringType,
		"autorun":      types.Int64Type,
		"vcpu_num":     types.Int64Type,
		"vram_size":    types.Int64Type,
		"disks":        types.SetType{ElemType: VDiskDataModel{}.ModelType()},
		"networks":     types.SetType{ElemType: VNicDataModel{}.ModelType()},
	}
}

func (m GuestDataSourceModel) Value() attr.Value {

	var networks attr.Value
	if m.Networks.IsNull() {
		networks = basetypes.NewSetNull(VNicDataModel{}.ModelType())
	} else {
		networks = basetypes.NewSetValueMust(VNicDataModel{}.ModelType(), m.Networks.Elements())
	}

	var disks attr.Value
	if m.Networks.IsNull() {
		disks = basetypes.NewSetNull(VDiskDataModel{}.ModelType())
	} else {
		disks = basetypes.NewSetValueMust(VDiskDataModel{}.ModelType(), m.Disks.Elements())
	}

	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"id":           types.StringValue(m.ID.ValueString()),
		"name":         types.StringValue(m.Name.ValueString()),
		"description":  types.StringValue(m.Description.ValueString()),
		"status":       types.StringValue(m.Status.ValueString()),
		"storage_id":   types.StringValue(m.StorageID.ValueString()),
		"storage_name": types.StringValue(m.StorageName.ValueString()),
		"autorun":      types.Int64Value(m.Autorun.ValueInt64()),
		"vcpu_num":     types.Int64Value(m.VcpuNum.ValueInt64()),
		"vram_size":    types.Int64Value(m.VramSize.ValueInt64()),
		"disks":        disks,
		"networks":     networks,
	})
}

func (m *GuestDataSourceModel) FromGuest(v *virtualization.Guest) error {
	m.ID = types.StringValue(v.ID)

	if m.Name.IsNull() {
		m.Name = types.StringValue(v.Name)
	}

	m.Description = types.StringValue(v.Description)
	m.Status = types.StringValue(v.Status)
	m.StorageID = types.StringValue(v.StorageID)
	m.StorageName = types.StringValue(v.StorageName)
	m.Autorun = types.Int64Value(v.AutoRun)
	m.VcpuNum = types.Int64Value(v.VcpuNum)
	m.VramSize = types.Int64Value(v.VramSize)

	disks := []attr.Value{}
	for _, d := range v.Disks {
		disk := VDiskDataModel{
			ID:         types.StringValue(d.ID),
			Size:       types.Int64Value(d.Size),
			Controller: types.Int64Value(d.Controller),
			Unmap:      types.BoolValue(d.Unmap),
		}.Value()

		disks = append(disks, disk)
	}
	if diskst, err := types.SetValue(VNicDataModel{}.ModelType(), disks); err == nil {
		m.Disks = diskst
	}

	nets := []attr.Value{}
	for _, n := range v.Networks {
		m := VNicDataModel{
			ID:    types.StringValue(n.ID),
			Mac:   types.StringValue(n.Mac),
			Name:  types.StringValue(n.Name),
			Model: types.Int64Value(n.Model),
		}.Value()
		nets = append(nets, m)
	}

	if netst, err := types.SetValue(VNicDataModel{}.ModelType(), nets); err == nil {
		m.Networks = netst
	}

	return nil
}

func (d *GuestDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "guest")
}

func (d *GuestDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Virtualization --- Guest data source",
		MarkdownDescription: "Virtualization --- Guest data source",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the guest.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for this data source.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the guest.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the guest.",
				Computed:            true,
			},
			"storage_id": schema.StringAttribute{
				MarkdownDescription: "Storage ID of the guest.",
				Computed:            true,
			},
			"storage_name": schema.StringAttribute{
				MarkdownDescription: "Storage name of the guest.",
				Computed:            true,
			},
			"autorun": schema.Int64Attribute{
				MarkdownDescription: "Hostname of Synology station.",
				Computed:            true,
			},
			"vcpu_num": schema.Int64Attribute{
				MarkdownDescription: "Number of virtual CPUs.",
				Computed:            true,
			},
			"vram_size": schema.Int64Attribute{
				MarkdownDescription: "Size of virtual RAM.",
				Computed:            true,
			},
			"disks": schema.SetAttribute{
				MarkdownDescription: "List of virtual disks.",
				Computed:            true,
				ElementType:         VDiskDataModel{}.ModelType(),
			},
			"networks": schema.SetAttribute{
				MarkdownDescription: "List of networks.",
				Computed:            true,
				ElementType:         VNicDataModel{}.ModelType(),
			},
		},
	}
}

func (d *GuestDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *GuestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GuestDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	name := data.Name.ValueString()

	clientResponse, err := d.client.VirtualizationAPI().GuestGet(ctx, virtualization.Guest{
		Name: name,
	})

	if err != nil {
		resp.Diagnostics.AddError("API request failed", fmt.Sprintf("Unable to read data source, got error: %s", err))
		return
	}

	if err := data.FromGuest(clientResponse); err != nil {
		resp.Diagnostics.AddError("Failed to read guest data", err.Error())
		return
	}

	if data.ID.IsNull() {
		resp.Diagnostics.AddError("Guest not found", "Guest not found")
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
