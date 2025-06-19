package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NetworkDataSource{}

func NewNetworkDataSource() datasource.DataSource {
	return &NetworkDataSource{}
}

type NetworkDataSource struct {
	client core.Api
}

// NetworkDataSourceModel is the data source model for network information.
type NetworkDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ArpIgnore              types.Bool   `tfsdk:"arp_ignore"`
	DNSManual              types.Bool   `tfsdk:"dns_manual"`
	DNSPrimary             types.String `tfsdk:"dns_primary"`
	DNSSecondary           types.String `tfsdk:"dns_secondary"`
	EnableIPConflictDetect types.Bool   `tfsdk:"enable_ip_conflict_detect"`
	EnableWindomain        types.Bool   `tfsdk:"enable_windomain"`
	Gateway                types.String `tfsdk:"gateway"`
	GatewayInfo            types.Object `tfsdk:"gateway_info"`
	IPv4First              types.Bool   `tfsdk:"ipv4_first"`
	MultiGateway           types.Bool   `tfsdk:"multi_gateway"`
	ServerName             types.String `tfsdk:"server_name"`
	UseDHCPDomain          types.Bool   `tfsdk:"use_dhcp_domain"`
	V6Gateway              types.String `tfsdk:"v6gateway"`
}

// GatewayInfoModel represents the gateway information.
type GatewayInfoModel struct {
	Interface types.String `tfsdk:"interface"`
	IP        types.String `tfsdk:"ip"`
	Mask      types.String `tfsdk:"mask"`
	Status    types.String `tfsdk:"status"`
	Type      types.String `tfsdk:"type"`
	UseDHCP   types.Bool   `tfsdk:"use_dhcp"`
}

func (m GatewayInfoModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"interface": types.StringType,
		"ip":        types.StringType,
		"mask":      types.StringType,
		"status":    types.StringType,
		"type":      types.StringType,
		"use_dhcp":  types.BoolType,
	}
}

func (m GatewayInfoModel) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"interface": m.Interface,
		"ip":        m.IP,
		"mask":      m.Mask,
		"status":    m.Status,
		"type":      m.Type,
		"use_dhcp":  m.UseDHCP,
	})
}

func (m NetworkDataSourceModel) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m NetworkDataSourceModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                        types.StringType,
		"arp_ignore":                types.BoolType,
		"dns_manual":                types.BoolType,
		"dns_primary":               types.StringType,
		"dns_secondary":             types.StringType,
		"enable_ip_conflict_detect": types.BoolType,
		"enable_windomain":          types.BoolType,
		"gateway":                   types.StringType,
		"gateway_info":              types.ObjectType{AttrTypes: GatewayInfoModel{}.AttrType()},
		"ipv4_first":                types.BoolType,
		"multi_gateway":             types.BoolType,
		"server_name":               types.StringType,
		"use_dhcp_domain":           types.BoolType,
		"v6gateway":                 types.StringType,
	}
}

func (m *NetworkDataSourceModel) FromNetworkResponse(response *core.NetworkConfig) error {
	m.ID = types.StringValue("network")
	m.ArpIgnore = types.BoolValue(response.ArpIgnore)
	m.DNSManual = types.BoolValue(response.DNSManual)
	m.DNSPrimary = types.StringValue(response.DNSPrimary)
	m.DNSSecondary = types.StringValue(response.DNSSecondary)
	m.EnableIPConflictDetect = types.BoolValue(response.EnableIPConflictDetect)
	m.EnableWindomain = types.BoolValue(response.EnableWindomain)
	m.Gateway = types.StringValue(response.Gateway)
	m.IPv4First = types.BoolValue(response.IPv4First)
	m.MultiGateway = types.BoolValue(response.MultiGateway)
	m.ServerName = types.StringValue(response.ServerName)
	m.UseDHCPDomain = types.BoolValue(response.UseDHCPDomain)
	m.V6Gateway = types.StringValue(response.V6Gateway)

	// Convert GatewayInfo
	gatewayInfo := GatewayInfoModel{
		Interface: types.StringValue(response.GatewayInfo.Interface),
		IP:        types.StringValue(response.GatewayInfo.IP),
		Mask:      types.StringValue(response.GatewayInfo.Mask),
		Status:    types.StringValue(response.GatewayInfo.Status),
		Type:      types.StringValue(response.GatewayInfo.Type),
		UseDHCP:   types.BoolValue(response.GatewayInfo.UseDHCP),
	}
	gatewayInfoObj, diags := types.ObjectValue(gatewayInfo.AttrType(), map[string]attr.Value{
		"interface": gatewayInfo.Interface,
		"ip":        gatewayInfo.IP,
		"mask":      gatewayInfo.Mask,
		"status":    gatewayInfo.Status,
		"type":      gatewayInfo.Type,
		"use_dhcp":  gatewayInfo.UseDHCP,
	})
	if diags.HasError() {
		return fmt.Errorf("failed to convert gateway info: %s", diags.Errors())
	}
	m.GatewayInfo = gatewayInfoObj

	return nil
}

func (d *NetworkDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "network")
}

func (d *NetworkDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Network data source to retrieve network configuration information from Synology DSM",
		MarkdownDescription: "Network data source to retrieve network configuration information from Synology DSM",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for this data source.",
				Computed:            true,
			},
			"arp_ignore": schema.BoolAttribute{
				MarkdownDescription: "Whether ARP ignore is enabled.",
				Computed:            true,
			},
			"dns_manual": schema.BoolAttribute{
				MarkdownDescription: "Whether DNS is manually configured.",
				Computed:            true,
			},
			"dns_primary": schema.StringAttribute{
				MarkdownDescription: "The primary DNS server configured on the Synology device.",
				Computed:            true,
			},
			"dns_secondary": schema.StringAttribute{
				MarkdownDescription: "The secondary DNS server configured on the Synology device.",
				Computed:            true,
			},
			"enable_ip_conflict_detect": schema.BoolAttribute{
				MarkdownDescription: "Whether IP conflict detection is enabled.",
				Computed:            true,
			},
			"enable_windomain": schema.BoolAttribute{
				MarkdownDescription: "Whether Windows domain is enabled.",
				Computed:            true,
			},
			"gateway": schema.StringAttribute{
				MarkdownDescription: "The default gateway configured on the Synology device.",
				Computed:            true,
			},
			"gateway_info": schema.SingleNestedAttribute{
				MarkdownDescription: "Gateway information details.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"interface": schema.StringAttribute{
						MarkdownDescription: "Gateway interface name.",
						Computed:            true,
					},
					"ip": schema.StringAttribute{
						MarkdownDescription: "Gateway IP address.",
						Computed:            true,
					},
					"mask": schema.StringAttribute{
						MarkdownDescription: "Gateway subnet mask.",
						Computed:            true,
					},
					"status": schema.StringAttribute{
						MarkdownDescription: "Gateway status.",
						Computed:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "Gateway type.",
						Computed:            true,
					},
					"use_dhcp": schema.BoolAttribute{
						MarkdownDescription: "Whether DHCP is used for the gateway.",
						Computed:            true,
					},
				},
			},
			"ipv4_first": schema.BoolAttribute{
				MarkdownDescription: "Whether IPv4 is prioritized over IPv6.",
				Computed:            true,
			},
			"multi_gateway": schema.BoolAttribute{
				MarkdownDescription: "Whether multiple gateways are configured.",
				Computed:            true,
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "The server name configured on the Synology device.",
				Computed:            true,
			},
			"use_dhcp_domain": schema.BoolAttribute{
				MarkdownDescription: "Whether DHCP domain is used.",
				Computed:            true,
			},
			"v6gateway": schema.StringAttribute{
				MarkdownDescription: "The IPv6 gateway configured on the Synology device.",
				Computed:            true,
			},
		},
	}
}

func (d *NetworkDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	if client, ok := req.ProviderData.(client.Api); ok {
		d.client = client.CoreAPI()
	} else {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}
}

func (d *NetworkDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data NetworkDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call the Core Network API
	networkResponse, err := d.client.NetworkGet(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"API request failed",
			fmt.Sprintf("Unable to read network data source, got error: %s", err),
		)
		return
	}

	if err := data.FromNetworkResponse(networkResponse); err != nil {
		resp.Diagnostics.AddError("Failed to read network data", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
