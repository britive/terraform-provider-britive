package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ScheduleScanResource manages a single scheduled scan ("task") under a resource type's scan
// task-service. The task service itself is a resource-type-scoped singleton auto-created by
// the API on the first task created for that resource type - this resource never creates it
// directly, and never tracks its ID in state, instead re-resolving it from resource_type_id on
// every Read/Update/Delete/Import via resolveTaskServiceID. There's no single-item GET for a
// task (only a list), so reads list-and-filter by task_id.
//
// The whole task service's enabled/disabled toggle (POST .../enabled-statuses,
// .../disabled-statuses) is resource-type-wide, not scoped to any one schedule, so it is
// deliberately NOT part of this resource - see scan_enabled on ResourceTypeResource instead.
type ScheduleScanResource struct {
	client *britive.Client
}

// ScheduleScanResourceModel describes the resource data model.
type ScheduleScanResourceModel struct {
	ID             types.String         `tfsdk:"id"`
	ResourceTypeID types.String         `tfsdk:"resource_type_id"`
	TaskID         types.String         `tfsdk:"task_id"`
	Name           types.String         `tfsdk:"name"`
	Description    types.String         `tfsdk:"description"`
	FrequencyType  types.String         `tfsdk:"frequency_type"`
	DayOfWeek      types.String         `tfsdk:"day_of_week"`
	DayOfMonth     types.Int64          `tfsdk:"day_of_month"`
	StartTime      types.String         `tfsdk:"start_time"`
	ResourceLabels []ResourceLabelModel `tfsdk:"resource_labels"`
	CreatedBy      types.String         `tfsdk:"created_by"`
	Created        types.Int64          `tfsdk:"created"`
	Modified       types.Int64          `tfsdk:"modified"`
	ModifiedBy     types.String         `tfsdk:"modified_by"`
	NextRun        types.Int64          `tfsdk:"next_run"`
}

// weekdayToInterval maps a day_of_week value (case-insensitive, full name or abbreviation)
// to the wire's frequencyInterval for Weekly schedules. Not evidenced by any capture (the
// captured UI flow only ever exercised Weekly with an already-fixed interval of 1) - this
// mapping is taken from the business rule as specified, and should be smoke-tested against
// the live API before relying on it.
var weekdayToInterval = map[string]int{
	"sunday": 1, "sun": 1,
	"monday": 2, "mon": 2,
	"tuesday": 3, "tue": 3,
	"wednesday": 4, "wed": 4,
	"thursday": 5, "thu": 5,
	"friday": 6, "fri": 6,
	"saturday": 7, "sat": 7,
}

// intervalToWeekday is weekdayToInterval's inverse, used to render the wire's
// frequencyInterval back into a canonical day name on Read/Import.
var intervalToWeekday = map[int]string{
	1: "Sunday", 2: "Monday", 3: "Tuesday", 4: "Wednesday", 5: "Thursday", 6: "Friday", 7: "Saturday",
}

// NewScheduleScanResource is a helper function to simplify the provider implementation.
func NewScheduleScanResource() resource.Resource {
	return &ScheduleScanResource{}
}

// Metadata returns the resource type name.
func (r *ScheduleScanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_type_schedule_scan"
}

// Schema defines the schema for the resource.
func (r *ScheduleScanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a scheduled scan for a Britive resource manager resource type",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the schedule scan",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Description: "The ID of the associated resource type",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"task_id": schema.StringAttribute{
				Description: "The unique identifier of the scheduled scan task",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the scheduled scan",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the scheduled scan",
				Optional:    true,
				Computed:    true,
			},
			"frequency_type": schema.StringAttribute{
				Description: "How often the scan runs. One of Daily, Weekly, Monthly (case-insensitive).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Daily", "Weekly", "Monthly"),
				},
			},
			"day_of_week": schema.StringAttribute{
				Description: "The day of the week the scan runs. Required when frequency_type = \"Weekly\"; must be unset otherwise. One of Sunday/Sun, Monday/Mon, Tuesday/Tue, Wednesday/Wed, Thursday/Thu, Friday/Fri, Saturday/Sat (case-insensitive).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"Sunday", "Sun", "Monday", "Mon", "Tuesday", "Tue", "Wednesday", "Wed",
						"Thursday", "Thu", "Friday", "Fri", "Saturday", "Sat",
					),
					stringvalidator.ConflictsWith(path.MatchRoot("day_of_month")),
				},
			},
			"day_of_month": schema.Int64Attribute{
				Description: "The day of the month (1-31) the scan runs. Required when frequency_type = \"Monthly\"; must be unset otherwise.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 31),
				},
			},
			"start_time": schema.StringAttribute{
				Description: "The time of day the scan runs, in 24-hour \"HH:MM\" format",
				Required:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the scheduled scan",
				Computed:    true,
			},
			"created": schema.Int64Attribute{
				Description: "The creation timestamp of the scheduled scan (epoch milliseconds)",
				Computed:    true,
			},
			"modified": schema.Int64Attribute{
				Description: "The last-modified timestamp of the scheduled scan (epoch milliseconds). Null until the first update.",
				Computed:    true,
			},
			"modified_by": schema.StringAttribute{
				Description: "The user who last modified the scheduled scan",
				Computed:    true,
			},
			"next_run": schema.Int64Attribute{
				Description: "The next scheduled run timestamp (epoch milliseconds)",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"resource_labels": schema.SetNestedBlock{
				Description: "Resource labels the scan is filtered to. Omit entirely to scan every resource of this type.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"label_key": schema.StringAttribute{
							Required:    true,
							Description: "Name of the resource label",
						},
						"values": schema.SetAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "The label's selected values to filter the scan to",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ScheduleScanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// ValidateConfig validates the resource configuration. day_of_week/day_of_month are gated
// by frequency_type: Daily requires both unset, Weekly requires day_of_week, Monthly
// requires day_of_month. day_of_week/day_of_month's own ConflictsWith validator already
// guarantees they're never both set, regardless of frequency_type.
func (r *ScheduleScanResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ScheduleScanResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.FrequencyType.IsUnknown() || data.DayOfWeek.IsUnknown() || data.DayOfMonth.IsUnknown() {
		return
	}

	hasDayOfWeek := !data.DayOfWeek.IsNull() && data.DayOfWeek.ValueString() != ""
	hasDayOfMonth := !data.DayOfMonth.IsNull()

	switch strings.ToLower(data.FrequencyType.ValueString()) {
	case "daily":
		if hasDayOfWeek || hasDayOfMonth {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Daily"' requires day_of_week and day_of_month to be unset`,
			)
		}
	case "weekly":
		if !hasDayOfWeek {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Weekly"' requires day_of_week to be set`,
			)
		}
		if hasDayOfMonth {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Weekly"' cannot be combined with day_of_month`,
			)
		}
	case "monthly":
		if !hasDayOfMonth {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Monthly"' requires day_of_month to be set`,
			)
		}
		if hasDayOfWeek {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Monthly"' cannot be combined with day_of_week`,
			)
		}
	}
}

// ModifyPlan normalizes description: when absent from config (null), force plan to null so
// the Computed+Optional field does not carry forward a prior-state value.
func (r *ScheduleScanResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ScheduleScanResourceModel
	var config ScheduleScanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Description.IsNull() && !plan.Description.IsNull() {
		plan.Description = types.StringNull()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ScheduleScanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScheduleScanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(plan.ResourceTypeID.ValueString())

	task := r.buildTaskPayload(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Creating schedule scan task for resource type: %s", resourceTypeID)

	created, err := r.client.CreateScheduleScanTask(resourceTypeID, task)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Schedule Scan", err.Error())
		return
	}

	plan.TaskID = types.StringValue(created.TaskID)
	plan.ID = types.StringValue(scheduleScanCompositeID(resourceTypeID, created.TaskID))

	r.mapModelToResource(ctx, created, &plan, false, &resp.Diagnostics)

	log.Printf("[INFO] Created schedule scan task: %s", created.TaskID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ScheduleScanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScheduleScanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())
	taskID := state.TaskID.ValueString()

	taskServiceID, err := r.resolveTaskServiceID(resourceTypeID)
	if errors.Is(err, britive.ErrScheduleScanTaskServiceNotBootstrapped) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Schedule Scan Task Service", err.Error())
		return
	}

	log.Printf("[INFO] Reading schedule scan task: %s", taskID)

	task, err := r.client.GetScheduleScanTask(taskServiceID, taskID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Schedule Scan", err.Error())
		return
	}

	r.mapModelToResource(ctx, task, &state, false, &resp.Diagnostics)
	state.ResourceLabels = r.refreshResourceLabels(ctx, task.Properties, &resp.Diagnostics)
	state.StartTime = formatStartTime(task.StartTime)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ScheduleScanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScheduleScanResourceModel
	var state ScheduleScanResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())
	taskID := state.TaskID.ValueString()

	taskServiceID, err := r.resolveTaskServiceID(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Schedule Scan Task Service", err.Error())
		return
	}

	task := r.buildTaskPayload(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Updating schedule scan task: %s", taskID)

	updated, err := r.client.UpdateScheduleScanTask(taskServiceID, taskID, task)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Schedule Scan", err.Error())
		return
	}

	plan.ID = state.ID
	plan.TaskID = state.TaskID

	r.mapModelToResource(ctx, updated, &plan, false, &resp.Diagnostics)

	log.Printf("[INFO] Updated schedule scan task: %s", taskID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ScheduleScanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScheduleScanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())
	taskID := state.TaskID.ValueString()

	taskServiceID, err := r.resolveTaskServiceID(resourceTypeID)
	if errors.Is(err, britive.ErrScheduleScanTaskServiceNotBootstrapped) {
		// The task service (and therefore this task) is already gone - nothing to delete.
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Schedule Scan Task Service", err.Error())
		return
	}

	log.Printf("[INFO] Deleting schedule scan task: %s", taskID)

	if err := r.client.DeleteScheduleScanTask(taskServiceID, taskID); err != nil {
		resp.Diagnostics.AddError("Error Deleting Schedule Scan", err.Error())
		return
	}

	log.Printf("[INFO] Deleted schedule scan task: %s", taskID)
}

// ImportState imports the resource state.
func (r *ScheduleScanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceTypeID, taskID, err := parseScheduleScanCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	taskServiceID, err := r.resolveTaskServiceID(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Schedule Scan Task Service", err.Error())
		return
	}

	log.Printf("[INFO] Importing schedule scan task: %s", taskID)

	task, err := r.client.GetScheduleScanTask(taskServiceID, taskID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Schedule Scan Not Found", fmt.Sprintf("Task %s not found under resource type %s", taskID, resourceTypeID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Schedule Scan", err.Error())
		return
	}

	var state ScheduleScanResourceModel
	state.ID = types.StringValue(scheduleScanCompositeID(resourceTypeID, taskID))
	state.ResourceTypeID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", resourceTypeID))
	state.TaskID = types.StringValue(taskID)

	r.mapModelToResource(ctx, task, &state, true, &resp.Diagnostics)
	state.ResourceLabels = r.refreshResourceLabels(ctx, task.Properties, &resp.Diagnostics)
	state.StartTime = formatStartTime(task.StartTime)

	log.Printf("[INFO] Imported schedule scan task: %s", taskID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// resolveTaskServiceID resolves resourceTypeID's scan task-service ID. Never stored in
// state - re-resolved on every call so the resource stays correct even if the task
// service's own ID were ever to change server-side.
func (r *ScheduleScanResource) resolveTaskServiceID(resourceTypeID string) (string, error) {
	taskService, err := r.client.GetScheduleScanTaskService(resourceTypeID)
	if err != nil {
		return "", err
	}
	return taskService.TaskServiceID, nil
}

// buildTaskPayload maps the plan into the API's create/update request shape, deriving
// frequencyInterval from day_of_week/day_of_month based on frequency_type.
func (r *ScheduleScanResource) buildTaskPayload(ctx context.Context, plan *ScheduleScanResourceModel, diags *diag.Diagnostics) britive.ScheduleScanTask {
	task := britive.ScheduleScanTask{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		FrequencyType: canonicalFrequencyType(plan.FrequencyType.ValueString()),
		StartTime:     plan.StartTime.ValueString(),
		// Always initialize the map so that removing all labels sends {} (not null) to the
		// API - a nil map marshals as JSON null, which the API treats as "no change" rather
		// than "clear" (confirmed by capture for this same map[string][]string shape).
		Properties: make(map[string][]string),
	}

	seen := make(map[string]bool, len(plan.ResourceLabels))
	for _, label := range plan.ResourceLabels {
		var values []string
		diags.Append(label.Values.ElementsAs(ctx, &values, false)...)
		labelKey := label.LabelKey.ValueString()
		// resource_labels is keyed by label_key on the wire, so two blocks with the same
		// label_key would silently collapse into one, sending only the last block's values
		// while Terraform's plan still expects both. Fail clearly instead of dropping data.
		if seen[labelKey] {
			diags.AddError(
				"Duplicate resource_labels Block",
				fmt.Sprintf("duplicate resource_labels block for label_key %q: each label_key must appear in at most one resource_labels block, merge their values into a single block", labelKey),
			)
			continue
		}
		seen[labelKey] = true
		task.Properties[labelKey] = values
	}

	switch strings.ToLower(plan.FrequencyType.ValueString()) {
	case "weekly":
		interval := weekdayToInterval[strings.ToLower(plan.DayOfWeek.ValueString())]
		task.FrequencyInterval = &interval
	case "monthly":
		interval := int(plan.DayOfMonth.ValueInt64())
		task.FrequencyInterval = &interval
	default: // daily
		task.FrequencyInterval = nil
	}

	return task
}

// canonicalFrequencyType normalizes a case-insensitive frequency_type input to the exact
// casing confirmed by capture ("Daily"/"Weekly"/"Monthly"), mirroring
// RotationTemplateResource's canonicalVariableType.
func canonicalFrequencyType(t string) string {
	switch strings.ToLower(t) {
	case "daily":
		return "Daily"
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	default:
		return t
	}
}

// mapModelToResource maps the API's schedule scan task detail onto Terraform state, except
// for resource_labels - see refreshResourceLabels for why that's handled separately.
// frequency_type and day_of_week are validated case-insensitively but sent to the API in
// one fixed casing/mapping, so the value echoed back is not necessarily what the user
// typed - when not importing, and the API's value case-insensitively/numerically matches
// what's already in state, the user's original casing/abbreviation is preserved instead of
// being overwritten, mirroring RotationTemplateResource.mapModelToResource.
func (r *ScheduleScanResource) mapModelToResource(ctx context.Context, task *britive.ScheduleScanTaskDetail, state *ScheduleScanResourceModel, imported bool, diags *diag.Diagnostics) {
	state.Name = types.StringValue(task.Name)
	state.Description = preserveOptionalString(task.Description, state.Description)

	if !imported && strings.EqualFold(state.FrequencyType.ValueString(), task.FrequencyType) {
		// Prior state already matches (case-insensitively) - keep the user's casing.
	} else {
		state.FrequencyType = types.StringValue(task.FrequencyType)
	}

	switch strings.ToLower(task.FrequencyType) {
	case "weekly":
		state.DayOfMonth = types.Int64Null()
		if task.FrequencyInterval == nil {
			state.DayOfWeek = types.StringNull()
			break
		}
		canonical := intervalToWeekday[*task.FrequencyInterval]
		if !imported && !state.DayOfWeek.IsNull() &&
			weekdayToInterval[strings.ToLower(state.DayOfWeek.ValueString())] == *task.FrequencyInterval {
			// Prior state already resolves to the same day - keep the user's original
			// casing/abbreviation (e.g. "Mon" vs "Monday").
		} else {
			state.DayOfWeek = types.StringValue(canonical)
		}
	case "monthly":
		state.DayOfWeek = types.StringNull()
		if task.FrequencyInterval != nil {
			state.DayOfMonth = types.Int64Value(int64(*task.FrequencyInterval))
		} else {
			state.DayOfMonth = types.Int64Null()
		}
	default: // daily
		state.DayOfWeek = types.StringNull()
		state.DayOfMonth = types.Int64Null()
	}

	state.CreatedBy = optionalStringValue(task.CreatedBy)
	state.Created = types.Int64Value(task.Created)
	if task.Modified == 0 {
		state.Modified = types.Int64Null()
	} else {
		state.Modified = types.Int64Value(task.Modified)
	}
	state.ModifiedBy = optionalStringValue(task.ModifiedBy)
	state.NextRun = types.Int64Value(task.NextRun)
}

// refreshResourceLabels rebuilds the resource_labels block list from the API's live
// properties map. Used only by Read/ImportState, never by Create/Update: label_key/values
// have no Computed sub-attributes, so the plan value the framework already resolved from
// config IS the exact value Create/Update must return. Rebuilding it there from the API
// response instead risks "Provider produced inconsistent result after apply" the moment the
// API's echoed properties aren't byte-for-byte identical to what was sent (e.g. an
// unrecognized label_key silently dropped) - Read/Import don't have that constraint, since
// refreshing state to match live truth (and letting any difference show up as an ordinary
// plan diff) is exactly their job.
func (r *ScheduleScanResource) refreshResourceLabels(ctx context.Context, properties map[string][]string, diags *diag.Diagnostics) []ResourceLabelModel {
	resourceLabelsList := make([]ResourceLabelModel, 0, len(properties))
	for labelKey, values := range properties {
		valuesSet, diagsSet := types.SetValueFrom(ctx, types.StringType, values)
		diags.Append(diagsSet...)
		if diagsSet.HasError() {
			continue
		}
		resourceLabelsList = append(resourceLabelsList, ResourceLabelModel{
			LabelKey: types.StringValue(labelKey),
			Values:   valuesSet,
		})
	}
	return resourceLabelsList
}

// formatStartTime renders the API's [hour, minute] pair as a "HH:MM" string. Used only by
// Read/ImportState, for the same reason as refreshResourceLabels: start_time has no Computed
// flag, so Create/Update must return exactly what was planned (already true by construction,
// since they never touch it) rather than a reformatted value that could disagree with a
// non-zero-padded input the user typed (e.g. "6:30").
func formatStartTime(startTime []int) types.String {
	if len(startTime) != 2 {
		return types.StringNull()
	}
	return types.StringValue(fmt.Sprintf("%02d:%02d", startTime[0], startTime[1]))
}

// scheduleScanCompositeID builds the resource's `id` value from resourceTypeID/taskID.
func scheduleScanCompositeID(resourceTypeID, taskID string) string {
	return fmt.Sprintf("resource-manager/resource-types/%s/schedule-scans/%s", resourceTypeID, taskID)
}

// parseScheduleScanCompositeID extracts resourceTypeID and taskID from either the composite
// ID ("resource-manager/resource-types/{resource_type_id}/schedule-scans/{task_id}") or a
// bare "{resource_type_id}/{task_id}" pair, for use on import.
func parseScheduleScanCompositeID(id string) (resourceTypeID string, taskID string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 5:
		return parts[2], parts[4], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", errs.NewInvalidResourceIDError("schedule scan", id)
	}
}
