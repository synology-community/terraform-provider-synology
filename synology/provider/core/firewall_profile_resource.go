package core

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

var (
	_ resource.Resource                = &FirewallProfileResource{}
	_ resource.ResourceWithImportState = &FirewallProfileResource{}
)

func NewFirewallProfileResource() resource.Resource {
	return &FirewallProfileResource{}
}

type FirewallProfileResource struct {
	client core.Api
}

type FirewallProfileResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Apply   types.Bool   `tfsdk:"apply"`
	Adapter types.List   `tfsdk:"adapter"`
}

type firewallAdapterModel struct {
	Name   types.String `tfsdk:"name"`
	Policy types.String `tfsdk:"policy"`
	Rule   types.List   `tfsdk:"rule"`
}

type firewallRuleModel struct {
	Enable        types.Bool   `tfsdk:"enable"`
	Log           types.Bool   `tfsdk:"log"`
	Name          types.String `tfsdk:"name"`
	Policy        types.String `tfsdk:"policy"`
	PortDirection types.String `tfsdk:"port_direction"`
	PortGroup     types.String `tfsdk:"port_group"`
	Ports         types.String `tfsdk:"ports"`
	Protocol      types.String `tfsdk:"protocol"`
	SourceIP      types.String `tfsdk:"source_ip"`
	SourceIPGroup types.String `tfsdk:"source_ip_group"`
}

func firewallRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enable":          types.BoolType,
		"log":             types.BoolType,
		"name":            types.StringType,
		"policy":          types.StringType,
		"port_direction":  types.StringType,
		"port_group":      types.StringType,
		"ports":           types.StringType,
		"protocol":        types.StringType,
		"source_ip":       types.StringType,
		"source_ip_group": types.StringType,
	}
}

func firewallAdapterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"policy": types.StringType,
		"rule":   types.ListType{ElemType: types.ObjectType{AttrTypes: firewallRuleAttrTypes()}},
	}
}

func (r *FirewallProfileResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "firewall_profile")
}

func (r *FirewallProfileResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	ruleAttrs := map[string]schema.Attribute{
		"enable": schema.BoolAttribute{
			MarkdownDescription: "Whether the rule is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"log": schema.BoolAttribute{
			MarkdownDescription: "Whether matching traffic is logged.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Optional rule display name.",
			Optional:            true,
			Computed:            true,
		},
		"policy": schema.StringAttribute{
			MarkdownDescription: "Rule action: `allow` or `drop`.",
			Required:            true,
		},
		"port_direction": schema.StringAttribute{
			MarkdownDescription: "Port direction (`destination` / `source`).",
			Optional:            true,
			Computed:            true,
		},
		"port_group": schema.StringAttribute{
			MarkdownDescription: "Port group (`custom`, built-in app lists, etc.).",
			Optional:            true,
			Computed:            true,
		},
		"ports": schema.StringAttribute{
			MarkdownDescription: "Comma-separated ports or ranges when port_group is custom.",
			Optional:            true,
			Computed:            true,
		},
		"protocol": schema.StringAttribute{
			MarkdownDescription: "Protocol (`tcp`, `udp`, `all`, …).",
			Optional:            true,
			Computed:            true,
		},
		"source_ip": schema.StringAttribute{
			MarkdownDescription: "Source address (`all`, IP, or network).",
			Optional:            true,
			Computed:            true,
		},
		"source_ip_group": schema.StringAttribute{
			MarkdownDescription: "Source type (`all`, `netmask`, `ip`, `geoip`, …).",
			Optional:            true,
			Computed:            true,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DSM firewall profile document (adapters + ordered rules) " +
			"via `SYNO.Core.Security.Firewall.Profile` set and optional " +
			"`Profile.Apply` two-phase commit.\n\n" +
			"**Break-glass:** never apply a default-drop profile with no allow rules — that locks " +
			"out DSM/SSH. Import the live profile first. New profile *names* may need to be " +
			"created once in Control Panel (DSM returns 117 for some create paths); afterward " +
			"set/apply work for updates.\n\n" +
			"DS1621+ Mode 1 reset restores network settings if locked out.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Profile name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"apply": schema.BoolAttribute{
				MarkdownDescription: "When true (default), Profile.Apply runs after set so rules " +
					"become live nftables. Set false to stage the profile file only.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"adapter": schema.ListNestedBlock{
				MarkdownDescription: "Per-adapter policy and ordered rules. Use `global` for " +
					"all-interface rules (typical).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Adapter name (`global`, `eth0`, …).",
							Required:            true,
						},
						"policy": schema.StringAttribute{
							MarkdownDescription: "Default policy for the adapter (`none`, `allow`, `drop`).",
							Required:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"rule": schema.ListNestedBlock{
							MarkdownDescription: "Ordered firewall rules (first match wins).",
							NestedObject: schema.NestedBlockObject{
								Attributes: ruleAttrs,
							},
						},
					},
				},
			},
		},
	}
}

func (r *FirewallProfileResource) Configure(
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

func (r *FirewallProfileResource) toProfile(
	ctx context.Context,
	data FirewallProfileResourceModel,
) (core.FirewallProfile, error) {
	profile := core.FirewallProfile{
		Name:     data.Name.ValueString(),
		Adapters: map[string]core.FirewallAdapterRules{},
	}
	if data.Adapter.IsNull() || data.Adapter.IsUnknown() {
		return profile, nil
	}
	var adapters []firewallAdapterModel
	if diags := data.Adapter.ElementsAs(ctx, &adapters, false); diags.HasError() {
		return profile, fmt.Errorf("adapter decode: %s", diags.Errors())
	}
	for _, a := range adapters {
		block := core.FirewallAdapterRules{
			Policy: a.Policy.ValueString(),
			Rules:  []core.FirewallRule{},
		}
		if !a.Rule.IsNull() && !a.Rule.IsUnknown() {
			var rules []firewallRuleModel
			if diags := a.Rule.ElementsAs(ctx, &rules, false); diags.HasError() {
				return profile, fmt.Errorf("rule decode: %s", diags.Errors())
			}
			for _, rule := range rules {
				block.Rules = append(block.Rules, core.FirewallRule{
					Enable:        rule.Enable.ValueBool(),
					Log:           rule.Log.ValueBool(),
					Name:          rule.Name.ValueString(),
					Policy:        rule.Policy.ValueString(),
					PortDirection: rule.PortDirection.ValueString(),
					PortGroup:     rule.PortGroup.ValueString(),
					Ports:         rule.Ports.ValueString(),
					Protocol:      rule.Protocol.ValueString(),
					SourceIP:      rule.SourceIP.ValueString(),
					SourceIPGroup: rule.SourceIPGroup.ValueString(),
				})
			}
		}
		profile.Adapters[a.Name.ValueString()] = block
	}
	return profile, nil
}

func (r *FirewallProfileResource) fromProfile(
	ctx context.Context,
	data *FirewallProfileResourceModel,
	profile *core.FirewallProfile,
) error {
	data.Name = types.StringValue(profile.Name)

	adapterObjs := make([]attr.Value, 0, len(profile.Adapters))
	// Stable order: global first, then remaining keys alphabetically.
	keys := make([]string, 0, len(profile.Adapters))
	for k := range profile.Adapters {
		if k != "global" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if _, ok := profile.Adapters["global"]; ok {
		keys = append([]string{"global"}, keys...)
	}

	for _, name := range keys {
		block := profile.Adapters[name]
		ruleObjs := make([]attr.Value, 0, len(block.Rules))
		for _, rule := range block.Rules {
			obj, diags := types.ObjectValue(firewallRuleAttrTypes(), map[string]attr.Value{
				"enable":          types.BoolValue(rule.Enable),
				"log":             types.BoolValue(rule.Log),
				"name":            types.StringValue(rule.Name),
				"policy":          types.StringValue(rule.Policy),
				"port_direction":  types.StringValue(rule.PortDirection),
				"port_group":      types.StringValue(rule.PortGroup),
				"ports":           types.StringValue(rule.Ports),
				"protocol":        types.StringValue(rule.Protocol),
				"source_ip":       types.StringValue(rule.SourceIP),
				"source_ip_group": types.StringValue(rule.SourceIPGroup),
			})
			if diags.HasError() {
				return fmt.Errorf("rule object: %s", diags.Errors())
			}
			ruleObjs = append(ruleObjs, obj)
		}
		ruleList, diags := types.ListValue(
			types.ObjectType{AttrTypes: firewallRuleAttrTypes()},
			ruleObjs,
		)
		if diags.HasError() {
			return fmt.Errorf("rule list: %s", diags.Errors())
		}
		adapterObj, diags := types.ObjectValue(firewallAdapterAttrTypes(), map[string]attr.Value{
			"name":   types.StringValue(name),
			"policy": types.StringValue(block.Policy),
			"rule":   ruleList,
		})
		if diags.HasError() {
			return fmt.Errorf("adapter object: %s", diags.Errors())
		}
		adapterObjs = append(adapterObjs, adapterObj)
	}
	list, diags := types.ListValue(
		types.ObjectType{AttrTypes: firewallAdapterAttrTypes()},
		adapterObjs,
	)
	if diags.HasError() {
		return fmt.Errorf("adapter list: %s", diags.Errors())
	}
	data.Adapter = list
	_ = ctx
	return nil
}

func (r *FirewallProfileResource) push(
	ctx context.Context,
	data FirewallProfileResourceModel,
) error {
	profile, err := r.toProfile(ctx, data)
	if err != nil {
		return err
	}
	if err := r.client.FirewallProfileSet(ctx, profile, false); err != nil {
		return fmt.Errorf("profile set: %w", err)
	}
	if data.Apply.ValueBool() {
		if err := r.client.FirewallProfileApply(ctx, profile.Name); err != nil {
			return fmt.Errorf("profile apply: %w", err)
		}
	}
	return nil
}

func (r *FirewallProfileResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan FirewallProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.push(ctx, plan); err != nil {
		resp.Diagnostics.AddError(
			"Failed to create firewall profile",
			err.Error()+"\n\nIf DSM returned 117, create an empty profile with this name in "+
				"Control Panel once, then re-apply or import.",
		)
		return
	}
	got, err := r.client.FirewallProfileGet(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read profile after create", err.Error())
		return
	}
	if err := r.fromProfile(ctx, &plan, got); err != nil {
		resp.Diagnostics.AddError("Failed to map profile after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallProfileResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state FirewallProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.FirewallProfileGet(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read firewall profile", err.Error())
		return
	}
	if err := r.fromProfile(ctx, &state, got); err != nil {
		resp.Diagnostics.AddError("Failed to map firewall profile", err.Error())
		return
	}
	// apply is config-only; preserve state/default
	if state.Apply.IsNull() || state.Apply.IsUnknown() {
		state.Apply = types.BoolValue(true)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FirewallProfileResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan FirewallProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.push(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Failed to update firewall profile", err.Error())
		return
	}
	got, err := r.client.FirewallProfileGet(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read profile after update", err.Error())
		return
	}
	if err := r.fromProfile(ctx, &plan, got); err != nil {
		resp.Diagnostics.AddError("Failed to map profile after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallProfileResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state FirewallProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Refuse to delete the active profile — that is an easy lockout path.
	fw, err := r.client.FirewallGet(ctx)
	if err == nil && fw != nil && fw.ProfileName == state.Name.ValueString() {
		resp.Diagnostics.AddError(
			"Refusing to delete active firewall profile",
			fmt.Sprintf(
				"profile %q is the active firewall profile; switch active profile first",
				state.Name.ValueString(),
			),
		)
		return
	}
	if err := r.client.FirewallProfileDelete(ctx, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete firewall profile", err.Error())
		return
	}
}

func (r *FirewallProfileResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
