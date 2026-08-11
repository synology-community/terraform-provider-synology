package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

var (
	_ resource.Resource                = &FirewallResource{}
	_ resource.ResourceWithImportState = &FirewallResource{}
)

func NewFirewallResource() resource.Resource {
	return &FirewallResource{}
}

type FirewallResource struct {
	client core.Api
}

// FirewallResourceModel is the singleton firewall enablement + active profile + conf.
type FirewallResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Enable          types.Bool   `tfsdk:"enable"`
	ProfileName     types.String `tfsdk:"profile_name"`
	EnablePortCheck types.Bool   `tfsdk:"enable_port_check"`
}

func (r *FirewallResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "firewall")
}

func (r *FirewallResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages DSM firewall enablement, the active profile name, and " +
			"port-check conf (`SYNO.Core.Security.Firewall` + `.Conf`).\n\n" +
			"**Break-glass:** enabling the firewall with an empty / default-deny profile locks " +
			"out DSM (including SSH). Create allow rules on the profile *before* enabling, and " +
			"prefer importing the live config rather than inventing rules from scratch. " +
			"DS1621+ Mode 1 reset restores network settings if locked out.\n\n" +
			"Some service accounts receive DSM error 114 on `Firewall.set` even when " +
			"`Firewall.get` works; profile rules still manage via " +
			"`synology_core_firewall_profile`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Singleton id (`firewall`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enable": schema.BoolAttribute{
				MarkdownDescription: "Whether the firewall is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"profile_name": schema.StringAttribute{
				MarkdownDescription: "Active firewall profile name (must already exist).",
				Required:            true,
			},
			"enable_port_check": schema.BoolAttribute{
				MarkdownDescription: "Whether DSM firewall port-check is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *FirewallResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(synology.Api)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected synology.Api, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client.CoreAPI()
}

func (r *FirewallResource) refresh(ctx context.Context, data *FirewallResourceModel) error {
	fw, err := r.client.FirewallGet(ctx)
	if err != nil {
		return fmt.Errorf("firewall get: %w", err)
	}
	conf, err := r.client.FirewallConfGet(ctx)
	if err != nil {
		return fmt.Errorf("firewall conf get: %w", err)
	}
	data.ID = types.StringValue("firewall")
	data.Enable = types.BoolValue(fw.EnableFirewall)
	data.ProfileName = types.StringValue(fw.ProfileName)
	data.EnablePortCheck = types.BoolValue(conf.EnablePortCheck)
	return nil
}

func (r *FirewallResource) apply(ctx context.Context, data FirewallResourceModel) error {
	if err := r.client.FirewallSet(ctx, core.FirewallSetRequest{
		EnableFirewall: data.Enable.ValueBool(),
		ProfileName:    data.ProfileName.ValueString(),
	}); err != nil {
		return fmt.Errorf("firewall set: %w", err)
	}
	if err := r.client.FirewallConfSet(ctx, core.FirewallConfSetRequest{
		EnablePortCheck: data.EnablePortCheck.ValueBool(),
	}); err != nil {
		return fmt.Errorf("firewall conf set: %w", err)
	}
	return nil
}

func (r *FirewallResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan FirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Failed to configure firewall", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read firewall after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state FirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.refresh(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Failed to read firewall", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FirewallResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan FirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Failed to update firewall", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read firewall after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallResource) Delete(
	_ context.Context,
	_ resource.DeleteRequest,
	_ *resource.DeleteResponse,
) {
	// Intentionally a no-op: removing the resource from state must not disable
	// the live firewall (that would open the box). Use enable=false explicitly.
}

func (r *FirewallResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Accept any id; singleton is always "firewall".
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "firewall")...)
}
