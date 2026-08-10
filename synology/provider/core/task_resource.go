package core

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	Enable   types.Bool   `tfsdk:"enable"`

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

	// PLAT-522: unconditional, unlike the neighbouring attributes below.
	// resp.State starts out equal to the prior state (not the plan), so any
	// attribute left unwritten here keeps its old value; Terraform then sees
	// the apply's actual new value diverge from what the provider reported
	// and fails with an inconsistent-result error. Guarding this on
	// plan != state would still be correct, but unconditional is simpler and
	// costs nothing.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enable"), plan.Enable)...)

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
	task, err := p.client.TaskGet(ctx, taskID)
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// enable is DSM's own schedule state, refreshed unconditionally (not only
	// when null/unknown) so an out-of-band change -- e.g. toggling the task in
	// DSM's Task Scheduler UI -- produces a plan diff instead of staying
	// permanently invisible. TaskResult.Enable's `omitempty` json tag only
	// affects marshalling, not unmarshalling, so it does not suppress a
	// `false` value read back here.
	data.Enable = types.BoolValue(task.Enable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
				MarkdownDescription: "Schedule expressed in cron, mapped onto DSM's scheduler: a fixed time " +
					"(`17 3 * * *`) becomes a daily task, a day-of-week restriction " +
					"(`17 3 * * 1-5`) becomes a weekly one, and an even interval (`*/15 * * * *`, " +
					"`30 */6 * * *`) becomes DSM's repeat_min/repeat_hour. Schedules DSM cannot " +
					"represent are rejected rather than silently mis-stored: day-of-month or month " +
					"restrictions, bounded windows (`0 9-17 * * *`), and unevenly spaced lists " +
					"(`0,7,30`).",
				Optional: true,
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
			"enable": schema.BoolAttribute{
				MarkdownDescription: "Whether the task's schedule is enabled in DSM. DSM only " +
					"writes a crontab row for enabled tasks, so a disabled task never fires " +
					"regardless of `schedule`. This is the schedule's on/off state, and is " +
					"distinct from `run`: `run` triggers one immediate, one-shot execution at " +
					"`when` and has no effect on whether the schedule itself is active. " +
					"Defaults to `true`, since a scheduled task that does not schedule is not " +
					"a useful default.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
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

// DSM's Task Scheduler expresses a schedule with a small set of fields whose
// meanings were established by probing a live DSM 7.4:
//
//	repeat_date 1001                  daily at hour:minute
//	repeat_date 1002 + week_day       weekly, on those weekdays, at hour:minute
//	repeat_hour N  (hour = start)     every N hours, beginning at hour:minute
//	repeat_min  N  (minute = start)   every N minutes, beginning at :minute
//
// Two properties of that model drive everything below. First, hour and minute
// are LITERAL values -- DSM writes them straight into /etc/crontab. Second,
// DSM validates none of it: a schedule of hour=128 is stored verbatim and
// yields a task that is created, reported as enabled, and can never fire. All
// validation therefore has to happen here.
const (
	dsmRepeatDaily  = 1001
	dsmRepeatWeekly = 1002

	// util.ParseStandard sets the top bit of a field that contained "*".
	cronStarBit = uint64(1) << 63
)

// cronFieldValues returns the values a cron bitfield encodes, ascending.
func cronFieldValues(mask int64, max int64) []int64 {
	u := uint64(mask) &^ cronStarBit
	values := make([]int64, 0, 8)
	for i := int64(0); i <= max; i++ {
		if u&(uint64(1)<<uint(i)) != 0 {
			values = append(values, i)
		}
	}
	return values
}

// cronField is one parsed cron field expressed in DSM's terms: a starting
// value, plus an interval when the field repeats.
type cronField struct {
	start int64
	step  int64 // 0 when the field names a single point in time
}

// classifyCronField maps one cron field onto DSM's model.
//
// A single value ("17") becomes a start with no step. An even progression that
// runs to the end of the field ("*", "*/15", "5/15") becomes a start plus a
// step, which DSM stores as repeat_min/repeat_hour. Anything else -- a list
// ("0,30"), a bounded range ("9-17"), or a stepped range that stops early
// ("0-45/15") -- selects a set of times DSM has no way to represent, so it is
// rejected rather than approximated.
func classifyCronField(mask int64, max int64, name string) (cronField, error) {
	values := cronFieldValues(mask, max)

	switch len(values) {
	case 0:
		// Unset: how the "@every ..." descriptors arrive, carrying only the
		// repeat_* counters.
		return cronField{}, nil
	case 1:
		return cronField{start: values[0]}, nil
	}

	step := values[1] - values[0]
	for i := 2; i < len(values); i++ {
		if values[i]-values[i-1] != step {
			return cronField{}, fmt.Errorf(
				"schedule field %s: %q selects an uneven set of values; DSM can express a "+
					"single %s or an even interval, not an arbitrary list",
				name, cronFieldString(values), name)
		}
	}

	// An interval must run to the end of the field. A progression that stops
	// early is a bounded window ("0-45/15" fires at :00 :15 :30 :45 and then
	// waits), and DSM has no field for the window.
	if last := values[len(values)-1]; last+step <= max {
		return cronField{}, fmt.Errorf(
			"schedule field %s: %q stops at %d rather than repeating through %d; DSM "+
				"repeats an interval for the whole %s range",
			name, cronFieldString(values), last, max, name)
	}

	return cronField{start: values[0], step: step}, nil
}

// cronFieldString renders values for an error message without dragging in a
// formatting dependency.
func cronFieldString(values []int64) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatInt(v, 10))
	}
	return strings.Join(parts, ",")
}

// weekDayString renders DSM's week_day field.
func weekDayString(values []int64) string {
	return cronFieldString(values)
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

	// "@every ..." sets only the repeat_* counters and leaves every cron field
	// empty; pass it through untouched.
	if s.Minute == 0 && s.Hour == 0 && s.Dom == 0 && s.Month == 0 && s.Dow == 0 {
		t.RepeatDate = s.RepeatDate
		t.RepeatHour = s.RepeatHour
		t.RepeatMin = s.RepeatMin
		if t.DateType == 0 && t.RepeatDate == 0 {
			t.RepeatDate = dsmRepeatDaily
		}
		return t, nil
	}

	// DSM schedules recur daily or weekly. Restricting the day of the month or
	// the month has no representation, and silently ignoring the restriction
	// would run the task far more often than asked.
	if dom := cronFieldValues(s.Dom, 31); len(dom) != 31 {
		return res, fmt.Errorf(
			"schedule: day-of-month restrictions are not supported; DSM repeats daily or "+
				"weekly, so %q cannot be expressed", c)
	}
	if months := cronFieldValues(s.Month, 12); len(months) != 12 {
		return res, fmt.Errorf(
			"schedule: month restrictions are not supported; DSM repeats daily or weekly, "+
				"so %q cannot be expressed", c)
	}

	minute, err := classifyCronField(s.Minute, 59, "minute")
	if err != nil {
		return res, err
	}
	hour, err := classifyCronField(s.Hour, 23, "hour")
	if err != nil {
		return res, err
	}

	// DSM carries one interval per task, in either repeat_min or repeat_hour.
	// A minute interval repeats across the whole day, so the hour field has to
	// be unrestricted -- "every 15 minutes, but only during hour 3" has no
	// representation. An unrestricted hour is not itself an interval here: it
	// is the absence of a restriction.
	hourUnrestricted := len(cronFieldValues(s.Hour, 23)) == 24

	switch {
	case minute.step != 0:
		if !hourUnrestricted {
			return res, fmt.Errorf(
				"schedule: %q repeats every %d minutes but also restricts the hour; DSM "+
					"repeats a minute interval across the whole day",
				c, minute.step)
		}
		t.Minute = minute.start
		t.Hour = 0
		t.RepeatMin = minute.step
	case hour.step != 0:
		t.Minute = minute.start
		t.Hour = hour.start
		t.RepeatHour = hour.step
	default:
		t.Minute = minute.start
		t.Hour = hour.start
	}

	// Day of week: all seven days is DSM's daily mode; any subset is its weekly
	// mode. The provider used to discard this field entirely, which turned
	// "0 3 * * 1" into a task that ran every morning instead of on Mondays.
	weekDays := cronFieldValues(s.Dow, 6)
	if len(weekDays) == 7 {
		t.RepeatDate = dsmRepeatDaily
	} else {
		t.RepeatDate = dsmRepeatWeekly
		t.WeekDay = weekDayString(weekDays)
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
		Enable:    data.Enable.ValueBool(),
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
