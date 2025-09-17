package core

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

type EventResourceModel struct {
	Name types.String `tfsdk:"name"`

	Script types.String `tfsdk:"script"`

	User  types.String `tfsdk:"user"`
	Event types.String `tfsdk:"event"`

	Run  types.Bool `tfsdk:"run"`
	When types.List `tfsdk:"when"`
}

var (
	_ resource.Resource                = &EventResource{}
	_ resource.ResourceWithImportState = &EventResource{}
)

func NewEventResource() resource.Resource {
	return &EventResource{}
}

type EventResource struct {
	client core.Api
}

// Create implements resource.Resource.
func (p *EventResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data EventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() {
		resp.Diagnostics.AddError("Name is required", "Name is required")
		return
	}

	eventReq := getEventRequest(data)

	var eventCreate func(ctx context.Context, req core.EventRequest) (*core.EventResult, error)

	// Check if the user is root
	isRoot := false
	for _, user := range eventReq.Owner {
		if user == "root" {
			isRoot = true
		}
	}

	if isRoot {
		eventCreate = p.client.RootEventCreate
	} else {
		eventCreate = p.client.EventCreate
	}

	_, err := eventCreate(ctx, eventReq)
	if err != nil {
		resp.Diagnostics.AddError("Event install failed", err.Error())
		return
	}

	_, err = p.client.EventGet(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to find event", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	when := []string{}
	resp.Diagnostics.Append(data.When.ElementsAs(ctx, &when, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Run.ValueBool() && slices.Contains(when, "apply") {
		err := p.client.EventRun(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run event", err.Error())
			return
		}
	}
}

// Update implements resource.Resource.
func (p *EventResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state EventResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	eventReq := getEventRequest(plan)

	var eventUpdate func(ctx context.Context, req core.EventRequest) (*core.EventResult, error)

	// Check if the user is root
	isRoot := false
	for _, user := range eventReq.Owner {
		if user == "root" {
			isRoot = true
		}
	}

	if isRoot {
		eventUpdate = p.client.RootEventUpdate
	} else {
		eventUpdate = p.client.EventUpdate
	}

	_, err := eventUpdate(ctx, eventReq)
	if err != nil {
		resp.Diagnostics.AddError("Event install failed", err.Error())
		return
	}

	if plan.Run.ValueBool() != state.Run.ValueBool() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("run"), plan.Run)...)
	}

	if !plan.When.Equal(state.When) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("when"), plan.When)...)
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), plan.Name)...)
	}

	if !plan.Script.Equal(state.Script) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("script"), plan.Script)...)
	}
	if !plan.User.Equal(state.User) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user"), plan.User)...)
	}
	if !plan.Event.Equal(state.Event) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("event"), plan.Event)...)
	}

	when := []string{}
	resp.Diagnostics.Append(plan.When.ElementsAs(ctx, &when, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Run.ValueBool() && slices.Contains(when, "upgrade") {
		err := p.client.EventRun(ctx, plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run event", err.Error())
			return
		}
	}
}

// Delete implements resource.Resource.
func (p *EventResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data EventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	when := []string{}
	resp.Diagnostics.Append(data.When.ElementsAs(ctx, &when, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Run.ValueBool() && slices.Contains(when, "destroy") {
		err := p.client.EventRun(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run event", err.Error())
			return
		}
	}

	err := p.client.EventDelete(ctx, core.EventRequest{Name: data.Name.ValueString()})
	if err != nil {
		event, err := p.client.EventGet(ctx, data.Name.ValueString())
		// Success, event not found
		if err != nil && event == nil {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError("Failed to uninstall event", err.Error())
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

// Metadata implements resource.Resource.
func (p *EventResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "event")
}

// Read implements resource.Resource.
func (p *EventResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data EventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	event, err := p.client.EventGet(ctx, data.Name.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(event.Name)
	data.Script = types.StringValue(event.Operation)
	for _, owner := range event.Owner {
		if owner == "root" {
			data.User = types.StringValue("root")
			break
		}
	}
	data.Event = types.StringValue(event.Event)
	if data.When.IsNull() || data.When.IsUnknown() {
		data.When = types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("apply"),
		})
	}
	if data.Run.IsNull() || data.Run.IsUnknown() {
		data.Run = types.BoolValue(false)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schema implements resource.Resource.
func (p *EventResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Generic API Resource for making calls to the Synology DSM API.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the event to install.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9 ]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$"),
						"Task name can include only English characters, numbers, and spaces; it cannot start/end with spaces.",
					),
				},
			},
			"script": schema.StringAttribute{
				MarkdownDescription: "Script content to run in the event.",
				Required:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "The user that will execute the event.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("root"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"event": schema.StringAttribute{
				MarkdownDescription: "Event trigger to run script. One of `bootup` or `shutdown`",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("bootup", "shutdown"),
				},
				Computed: true,
				Default:  stringdefault.StaticString("bootup"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Whether to run the event after creation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"when": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "When to run the event. Valid values are `apply` and `destroy`.",
				Optional:            true,
				Computed:            true,
				Default: listdefault.StaticValue(
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("apply"),
					}),
				),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("apply", "destroy", "upgrade"),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (f *EventResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(synology.Api); ok {
		f.client = client.CoreAPI()
	} else {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
	}
}

func (p *EventResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func getEventRequest(data EventResourceModel) core.EventRequest {
	event := data.Event.ValueString()
	if event == "" {
		event = "bootup"
	}

	user := data.User.ValueString()

	return core.EventRequest{
		Name:               data.Name.ValueString(),
		Owner:              map[string]string{"0": user},
		Operation:          data.Script.ValueString(),
		OperationType:      "script",
		Event:              event,
		Enable:             true,
		NotifyEnabled:      false,
		NotifyIfError:      false,
		NotifyMail:         "",
		SynoConfirmPWToken: "",
	}
}
