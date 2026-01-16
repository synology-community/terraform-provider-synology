package core

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

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
	"github.com/synology-community/terraform-provider-synology/synology/util"
)

type TaskResourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	Service types.String `tfsdk:"service"`
	Script  types.String `tfsdk:"script"`

	Schedule types.String `tfsdk:"schedule"`
	User     types.String `tfsdk:"user"`

	Run  types.Bool   `tfsdk:"run"`
	When types.String `tfsdk:"when"`
}

var _ resource.Resource = &TaskResource{}

func NewTaskResource() resource.Resource {
	return &TaskResource{}
}

type TaskResource struct {
	client core.Api
}

// Create implements resource.Resource.
func (p *TaskResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data TaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() {
		resp.Diagnostics.AddError("Name is required", "Name is required")
		return
	}

	taskReq, err := getTaskRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create task", err.Error())
		return
	}

	var taskCreate func(ctx context.Context, req core.TaskRequest) (*core.TaskResult, error)
	if taskReq.Owner == "root" {
		taskCreate = p.client.RootTaskCreate
	} else {
		taskCreate = p.client.TaskCreate
	}

	res, err := taskCreate(ctx, taskReq)
	if err != nil {
		resp.Diagnostics.AddError("Task install failed", err.Error())
		return
	}

	data.ID = types.Int64PointerValue(res.ID)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Run.ValueBool() && data.When.ValueString() == "apply" {
		err := p.client.TaskRun(ctx, data.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run task", err.Error())
			return
		}
	}
}

// Update implements resource.Resource.
func (p *TaskResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state TaskResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	taskReq, err := getTaskRequest(plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create task", err.Error())
		return
	}

	taskReq.ID = state.ID.ValueInt64Pointer()

	var taskUpdate func(ctx context.Context, req core.TaskRequest) (*core.TaskResult, error)
	if taskReq.Owner == "root" {
		taskUpdate = p.client.RootTaskUpdate
	} else {
		taskUpdate = p.client.TaskUpdate
	}

	_, err = taskUpdate(ctx, taskReq)
	if err != nil {
		resp.Diagnostics.AddError("Task install failed", err.Error())
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

	if (!plan.Service.IsNull() && !plan.Service.IsUnknown()) &&
		(plan.Service.ValueString() != state.Service.ValueString()) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), plan.Service)...)
	}

	if plan.Run.ValueBool() && plan.When.ValueString() == "upgrade" {
		err := p.client.TaskRun(ctx, plan.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run task", err.Error())
			return
		}
	}
}

// Delete implements resource.Resource.
func (p *TaskResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data TaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Run.ValueBool() && data.When.ValueString() == "destroy" {
		err := p.client.TaskRun(ctx, data.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Failed to run task", err.Error())
			return
		}
	}

	taskID := data.ID.ValueInt64()
	err := p.client.TaskDelete(ctx, taskID)
	if err != nil {
		task, err := p.client.TaskGet(ctx, taskID)
		// Success, task not found
		if err != nil && task == nil {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError("Failed to uninstall task", err.Error())
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

// Metadata implements resource.Resource.
func (p *TaskResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "task")
}

// Read implements resource.Resource.
func (p *TaskResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data TaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID := data.ID.ValueInt64()
	_, err := p.client.TaskGet(ctx, taskID)
	if err != nil {
		resp.State.RemoveResource(ctx)
	}

	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schema implements resource.Resource.
func (p *TaskResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Generic API Resource for making calls to the Synology DSM API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the task to install.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the task to install.",
				Required:            true,
			},
			"schedule": schema.StringAttribute{
				MarkdownDescription: "Schedule expressed in cron.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(
							`(@(annually|yearly|monthly|weekly|daily|hourly|reboot))|(@every (\d+(ns|us|µs|ms|s|m|h))+)|((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5,7})`,
						),
						"value must contain a valid cron expression",
					),
				},
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "Systemctl service to change state.",
				Optional:            true,
			},
			"script": schema.StringAttribute{
				MarkdownDescription: "Script content to run in the task.",
				Optional:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "The user that will execute the task.",
				Required:            true,
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Whether to run the task after creation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"when": schema.StringAttribute{
				MarkdownDescription: "When to run the task. Valid values are `apply` and `destroy`.",
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

func (f *TaskResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(synology.Api)

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

	f.client = client.CoreAPI()
}

func (p *TaskResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse ID", err.Error())
		return
	}

	task, err := p.client.TaskGet(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find task", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), task.ID)...)
}

func newTaskSchedule() core.TaskSchedule {
	return core.TaskSchedule{
		WeekDay:              "0,1,2,3,4,5,6",
		MonthlyWeek:          []string{},
		RepeatMinStoreConfig: []int64{1, 5, 10, 15, 20, 30},
		RepeatHourStoreConfig: []int64{
			1,
			2,
			3,
			4,
			5,
			6,
			7,
			8,
			9,
			10,
			11,
			12,
			13,
			14,
			15,
			16,
			17,
			18,
			19,
			20,
			21,
			22,
			23,
		},
	}
}

func parseSchedule(c string) (res core.TaskSchedule, err error) {
	if c == "" {
		return
	}

	s, err := util.ParseStandard(c)
	if err != nil {
		return
	}

	t := newTaskSchedule()

	t.Minute = s.Minute
	t.Hour = s.Hour
	t.RepeatDate = s.RepeatDate
	t.RepeatHour = s.RepeatHour
	t.RepeatMin = s.RepeatMin

	if t.DateType == 0 && t.RepeatDate == 0 {
		t.RepeatDate = 1001
	}
	return t, nil
}

func getTaskRequest(data TaskResourceModel) (taskReq core.TaskRequest, err error) {
	taskType := "script"

	if !data.Script.IsNull() && !data.Script.IsUnknown() && data.Script.ValueString() != "" {
		taskType = "script"
	}

	user := data.User.ValueString()

	taskReq = core.TaskRequest{
		Name:      data.Name.ValueString(),
		RealOwner: "root",
		Owner:     user,
		Type:      taskType,
		Extra: core.TaskExtra{
			Script: data.Script.ValueString(),
		},
	}

	if !data.Schedule.IsNull() && !data.Schedule.IsUnknown() && data.Schedule.ValueString() != "" {
		schedule, e := parseSchedule(data.Schedule.ValueString())
		if e != nil {
			err = e
			return
		}
		taskReq.Schedule = schedule
	} else {
		t := newTaskSchedule()
		pkgRunTime := time.Now().Local().Add(-time.Minute * 5)
		t.Date = fmt.Sprintf(
			"%d-%02d-%02d",
			pkgRunTime.Year(),
			pkgRunTime.Month(),
			pkgRunTime.Day(),
		)
		taskReq.Schedule = t
	}

	return taskReq, nil
}
