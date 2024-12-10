package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api/core"
)

type EventResourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	Service types.String `tfsdk:"service"`
	Script  types.String `tfsdk:"script"`

	Schedule types.String `tfsdk:"schedule"`
	User     types.String `tfsdk:"user"`

	Run  types.Bool   `tfsdk:"run"`
	When types.String `tfsdk:"when"`
}

var _ resource.Resource = &EventResource{}

func NewEventResource() resource.Resource {
	return &EventResource{}
}

type EventResource struct {
	client core.Api
}

// Create implements resource.Resource.
func (p *EventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() {
		resp.Diagnostics.AddError("Name is required", "Name is required")
		return
	}

	eventReq, err := getEventRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create event", err.Error())
		return
	}

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

	_, err = eventCreate(ctx, eventReq)
	if err != nil {
		resp.Diagnostics.AddError("Event install failed", err.Error())
		return
	}

	task, err := p.client.EventGet(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to find event", err.Error())
		return
	}

	data.ID = types.Int64PointerValue(task.ID)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Run.ValueBool() && data.When.ValueString() == "apply" {
		err := p.client.EventRun(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run event", err.Error())
			return
		}
	}
}

// Update implements resource.Resource.
func (p *EventResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EventResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	eventReq, err := getEventRequest(plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create event", err.Error())
		return
	}

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

	_, err = eventUpdate(ctx, eventReq)
	if err != nil {
		resp.Diagnostics.AddError("Event install failed", err.Error())
		return
	}

	if plan.Run.ValueBool() != state.Run.ValueBool() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("run"), plan.Run)...)
	}

	if plan.When.ValueString() != state.When.ValueString() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("when"), plan.When)...)
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), plan.Name)...)
	}

	if (!plan.Service.IsNull() && !plan.Service.IsUnknown()) && (plan.Service.ValueString() != state.Service.ValueString()) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), plan.Service)...)
	}

	if plan.Run.ValueBool() && plan.When.ValueString() == "upgrade" {
		err := p.client.EventRun(ctx, plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run event", err.Error())
			return
		}
	}
}

// Delete implements resource.Resource.
func (p *EventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Run.ValueBool() && data.When.ValueString() == "destroy" {
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
func (p *EventResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = buildName(req.ProviderTypeName, "event")
}

// Read implements resource.Resource.
func (p *EventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := p.client.EventGet(ctx, data.Name.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
	}

	//resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schema implements resource.Resource.
func (p *EventResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Generic API Resource for making calls to the Synology DSM API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the event to install.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the event to install.",
				Required:            true,
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
			},
			"event": schema.StringAttribute{
				MarkdownDescription: "Event trigger to run script. One of `bootup` or `shutdown`",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("bootup", "shutdown"),
				},
				Computed: true,
				Default:  stringdefault.StaticString("bootup"),
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Whether to run the event after creation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"when": schema.StringAttribute{
				MarkdownDescription: "When to run the event. Valid values are `apply` and `destroy`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("apply"),
				Validators: []validator.String{
					stringvalidator.OneOf("apply", "destroy", "upgrade"),
				},
			},
		},
	}
}

func (f *EventResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(synology.Api)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	f.client = client.CoreAPI()
}

func (p *EventResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID

	event, err := p.client.EventGet(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find event", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), event.ID)...)
}

func getEventRequest(data EventResourceModel) (eventReq core.EventRequest, err error) {
	event := "bootup"

	if !data.Script.IsNull() && !data.Script.IsUnknown() && data.Script.ValueString() != "" {
		event = "bootup"
	}

	user := data.User.ValueString()

	eventReq = core.EventRequest{
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

	return eventReq, nil
}
