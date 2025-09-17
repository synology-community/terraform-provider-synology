package container

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/docker"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.ResourceWithIdentity    = &NetworkResource{}
	_ resource.ResourceWithImportState = &NetworkResource{}
)

func NewNetworkResource() resource.Resource {
	return &NetworkResource{}
}

type NetworkResource struct {
	client docker.Api
}

// NetworkResourceModel describes the resource data model.
type NetworkResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Driver            types.String `tfsdk:"driver"`
	Subnet            types.String `tfsdk:"subnet"`
	IPRange           types.String `tfsdk:"ip_range"`
	Gateway           types.String `tfsdk:"gateway"`
	EnableIPv6        types.Bool   `tfsdk:"enable_ipv6"`
	IPv6Subnet        types.String `tfsdk:"ipv6_subnet"`
	IPv6Gateway       types.String `tfsdk:"ipv6_gateway"`
	IPv6IPRange       types.String `tfsdk:"ipv6_ip_range"`
	DisableMasquerade types.Bool   `tfsdk:"disable_masquerade"`
}

type NetworkResourceIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *NetworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_container_network"
	resp.ResourceBehavior = resource.ResourceBehavior{
		MutableIdentity: true,
	}
}

func (r *NetworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages a Container Manager network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Network identifier",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Network name",
				Required:            true,
			},
			"driver": schema.StringAttribute{
				MarkdownDescription: "Network driver (bridge, overlay, etc.)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("bridge"),
				Validators: []validator.String{
					stringvalidator.OneOf(
						"bridge",
						"overlay",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "Subnet for the network in CIDR format.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ip_range": schema.StringAttribute{
				MarkdownDescription: "IP range for the network in CIDR format.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"gateway": schema.StringAttribute{
				MarkdownDescription: "Gateway for the network.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"disable_masquerade": schema.BoolAttribute{
				MarkdownDescription: "Disable masquerading for the network. This is optional and can be set later.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			// IPv6 related attributes
			// These are optional and can be set later
			"enable_ipv6": schema.BoolAttribute{
				MarkdownDescription: "Enable IPv6 for the network.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_subnet": schema.StringAttribute{
				MarkdownDescription: "IPv6 subnet for the network in CIDR format. This is optional and can be set later.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_gateway": schema.StringAttribute{
				MarkdownDescription: "IPv6 gateway for the network. This is optional and can be set later.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_ip_range": schema.StringAttribute{
				MarkdownDescription: "IPv6 IP range for the network in CIDR format. This is optional and can be set later.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r NetworkResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Network identifier",
				OptionalForImport: true,
			},
			"name": identityschema.StringAttribute{
				Description:       "Network name",
				RequiredForImport: true,
			},
		},
	}
}

func (r *NetworkResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Api)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected docker.Api, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.client = client.DockerAPI()
}

func (r *NetworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data NetworkResourceModel
	var identityData NetworkResourceIdentityModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// Don't get identity data during create - it might be null
	// resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating network", map[string]any{
		"name": data.Name.ValueString(),
	})

	// Create the network using data from the plan (not identity)
	if err := r.client.NetworkCreate(ctx, docker.Network{
		Name:       data.Name.ValueString(),
		Driver:     data.Driver.ValueString(),
		Subnet:     data.Subnet.ValueString(),
		IPRange:    data.IPRange.ValueString(),
		Gateway:    data.Gateway.ValueString(),
		EnableIPv6: data.EnableIPv6.ValueBool(),
		IPv6Subnet: data.IPv6Subnet.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create network, got error: %s", err),
		)
		return
	}

	network, err := r.client.NetworkGetByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read network after creation, got error: %s", err),
		)
		return
	}

	// Set the ID and other computed fields
	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Driver = types.StringValue(network.Driver)
	data.Subnet = types.StringValue(network.Subnet)
	data.IPRange = types.StringValue(network.IPRange)
	data.Gateway = types.StringValue(network.Gateway)
	data.EnableIPv6 = types.BoolValue(network.EnableIPv6)

	// Set identity data for this resource
	identityData.ID = types.StringValue(network.ID)
	identityData.Name = types.StringValue(network.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &identityData)...)

	tflog.Debug(ctx, "Created network", map[string]any{
		"id":          data.ID.ValueString(),
		"name":        data.Name.ValueString(),
		"driver":      data.Driver.ValueString(),
		"subnet":      data.Subnet.ValueString(),
		"ip_range":    data.IPRange.ValueString(),
		"gateway":     data.Gateway.ValueString(),
		"enable_ipv6": data.EnableIPv6.ValueBool(),
	})
}

func (r *NetworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data NetworkResourceModel
	var identityData NetworkResourceIdentityModel

	// Handle imports: only one of the models will be populated on import
	// Check which model exists before attempting to get it
	var stateGetDiags, identityGetDiags diag.Diagnostics

	stateGetDiags = req.State.Get(ctx, &data)
	identityGetDiags = req.Identity.Get(ctx, &identityData)

	// Only fail if both have errors (meaning neither model is populated)
	if stateGetDiags.HasError() && identityGetDiags.HasError() {
		resp.Diagnostics.Append(stateGetDiags...)
		resp.Diagnostics.Append(identityGetDiags...)
		return
	}

	// Determine which model has data and populate the empty one
	var networkID, networkName string

	// Check if identity data is populated (has non-null/unknown values and no get errors)
	identityHasData := !identityGetDiags.HasError() &&
		((!identityData.ID.IsNull() && !identityData.ID.IsUnknown()) ||
			(!identityData.Name.IsNull() && !identityData.Name.IsUnknown()))
	stateHasData := !stateGetDiags.HasError() &&
		((!data.ID.IsNull() && !data.ID.IsUnknown()) ||
			(!data.Name.IsNull() && !data.Name.IsUnknown()))

	if identityHasData && !identityData.ID.IsNull() && !identityData.ID.IsUnknown() {
		networkID = identityData.ID.ValueString()
		networkName = identityData.Name.ValueString()
	} else if identityHasData && !identityData.Name.IsNull() && !identityData.Name.IsUnknown() {
		networkName = identityData.Name.ValueString()
	} else if stateHasData && !data.ID.IsNull() && !data.ID.IsUnknown() {
		networkID = data.ID.ValueString()
		networkName = data.Name.ValueString()
		// Fill the identity model from state data
		identityData.ID = data.ID
		identityData.Name = data.Name
	} else if stateHasData && !data.Name.IsNull() && !data.Name.IsUnknown() {
		networkName = data.Name.ValueString()
		// Fill the identity model from state data
		identityData.Name = data.Name
	}

	tflog.Debug(ctx, "Reading network", map[string]any{
		"id":   networkID,
		"name": networkName,
	})

	var network *docker.Network
	var err error

	// Try to get by ID first, fall back to name
	if networkID != "" {
		network, err = r.client.NetworkGetByID(ctx, networkID)
	} else if networkName != "" {
		network, err = r.client.NetworkGetByName(ctx, networkName)
	} else {
		resp.Diagnostics.AddError(
			"Invalid State",
			"Neither network ID nor name is available for reading network",
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read network, got error: %s", err),
		)
		return
	}

	// Update both models with the network data
	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Driver = types.StringValue(network.Driver)
	data.Subnet = types.StringValue(network.Subnet)
	data.IPRange = types.StringValue(network.IPRange)
	data.Gateway = types.StringValue(network.Gateway)
	data.EnableIPv6 = types.BoolValue(network.EnableIPv6)
	data.IPv6Subnet = types.StringValue(network.IPv6Subnet)
	data.IPv6Gateway = types.StringValue(network.IPv6Gateway)
	data.IPv6IPRange = types.StringValue(network.IPv6IPRange)
	data.DisableMasquerade = types.BoolValue(network.DisableMasquerade)

	identityData.ID = types.StringValue(network.ID)
	identityData.Name = types.StringValue(network.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &identityData)...)
}

func (r *NetworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data NetworkResourceModel
	var identityData NetworkResourceIdentityModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating network", map[string]any{
		"id": identityData.ID.ValueString(),
	})

	network, err := r.client.NetworkGetByID(ctx, identityData.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read network before deletion, got error: %s", err),
		)
		return
	}

	if network == nil {
		tflog.Debug(ctx, "Network already deleted", map[string]any{
			"id": identityData.ID.ValueString(),
		})
		return
	}

	// Delete the network
	tflog.Debug(ctx, "Deleting network by ID", map[string]any{
		"id": identityData.ID.ValueString(),
	})

	// Use the ID from the identity data to delete the network
	// This ensures we delete the correct network even if the name has changed
	if identityData.ID.IsNull() || identityData.ID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Invalid Network ID",
			"Network ID is required for deletion but is null or unknown.",
		)
		return
	}

	containers := network.Containers
	// If the network has existing containers, first remove them
	// Store the container names for re-creation in this method
	if len(containers) > 0 {
		tflog.Debug(
			ctx,
			"Network has existing containers, removing them before deletion",
			map[string]any{
				"id":         identityData.ID.ValueString(),
				"containers": network.Containers,
			},
		)
		if err := r.client.NetworkUpdate(ctx, docker.NetworkUpdateRequest{
			Name:       identityData.Name.ValueString(),
			Containers: []string{}, // Clear containers to remove them from the network
		}); err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to update network before deletion, got error: %s", err),
			)
			return
		}
		network.Containers = nil // Clear the containers from the network object
		tflog.Debug(ctx, "Removed containers from network", map[string]any{
			"id": identityData.ID.ValueString(),
		})
	}

	err = r.client.NetworkDelete(ctx, *network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete network, got error: %s", err),
		)
		return
	}

	tflog.Debug(ctx, "Deleted network", map[string]any{
		"id": identityData.ID.ValueString(),
	})

	// Create the network with updated values
	if err := r.client.NetworkCreate(ctx, *network); err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create network after deletion, got error: %s", err),
		)
		return
	}

	if err := r.client.NetworkUpdate(ctx, docker.NetworkUpdateRequest{
		Name:       identityData.Name.ValueString(),
		Containers: containers, // Re-add containers to the network
	}); err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update network after creation, got error: %s", err),
		)
		return
	}

	network, err = r.client.NetworkGetByName(ctx, identityData.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read network after update, got error: %s", err),
		)
		return
	}

	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Driver = types.StringValue(network.Driver)
	data.Subnet = types.StringValue(network.Subnet)
	data.IPRange = types.StringValue(network.IPRange)
	data.Gateway = types.StringValue(network.Gateway)
	data.EnableIPv6 = types.BoolValue(network.EnableIPv6)
	data.IPv6Subnet = types.StringValue(network.IPv6Subnet)
	data.IPv6Gateway = types.StringValue(network.IPv6Gateway)
	data.IPv6IPRange = types.StringValue(network.IPv6IPRange)
	data.DisableMasquerade = types.BoolValue(network.DisableMasquerade)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	identityData.ID = types.StringValue(network.ID)
	identityData.Name = types.StringValue(network.Name)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, &identityData)...)

	tflog.Debug(ctx, "Updated network", map[string]any{
		"id":                 identityData.ID.ValueString(),
		"name":               identityData.Name.ValueString(),
		"driver":             data.Driver.ValueString(),
		"subnet":             data.Subnet.ValueString(),
		"ip_range":           data.IPRange.ValueString(),
		"gateway":            data.Gateway.ValueString(),
		"enable_ipv6":        data.EnableIPv6.ValueBool(),
		"ipv6_subnet":        data.IPv6Subnet.ValueString(),
		"ipv6_gateway":       data.IPv6Gateway.ValueString(),
		"ipv6_ip_range":      data.IPv6IPRange.ValueString(),
		"disable_masquerade": data.DisableMasquerade.ValueBool(),
	})
}

func (r *NetworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine which model has the network ID
	var networkID string

	tflog.Debug(ctx, "Deleting network", map[string]any{
		"id": networkID,
	})

	network, err := r.client.NetworkGetByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read network before deletion, got error: %s", err),
		)
		return
	}

	if network == nil {
		tflog.Debug(ctx, "Network already deleted", map[string]any{
			"id": data.ID.ValueString(),
		})
		return
	}

	// Delete the network
	tflog.Debug(ctx, "Deleting network by ID", map[string]any{
		"id": data.ID.ValueString(),
	})

	// Use the ID from the identity data to delete the network
	// This ensures we delete the correct network even if the name has changed
	if data.ID.IsNull() || data.ID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Invalid Network ID",
			"Network ID is required for deletion but is null or unknown.",
		)
		return
	}

	containers := network.Containers

	// If the network has existing containers, first remove them
	// Store the container names for re-creation in this method
	if len(containers) > 0 {
		tflog.Debug(
			ctx,
			"Network has existing containers, removing them before deletion",
			map[string]any{
				"id":         data.ID.ValueString(),
				"containers": network.Containers,
			},
		)
		if err := r.client.NetworkUpdate(ctx, docker.NetworkUpdateRequest{
			Name:       data.Name.ValueString(),
			Containers: []string{}, // Clear containers to remove them from the network
		}); err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to update network before deletion, got error: %s", err),
			)
			return
		}
		network.Containers = nil // Clear the containers from the network object
		tflog.Debug(ctx, "Removed containers from network", map[string]any{
			"id": data.ID.ValueString(),
		})
	}

	err = r.client.NetworkDelete(ctx, *network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete network, got error: %s", err),
		)
		return
	}

	tflog.Debug(ctx, "Deleted network", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.State.RemoveResource(ctx)
}

func (r *NetworkResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughWithIdentity(
		ctx,
		path.Root("name"),
		path.Root("name"),
		req,
		resp,
	)
}
