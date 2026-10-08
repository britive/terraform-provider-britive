package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/applicationscan"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/schedulescan"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ApplicationScanScheduleResource{}
	_ resource.ResourceWithConfigure      = &ApplicationScanScheduleResource{}
	_ resource.ResourceWithImportState    = &ApplicationScanScheduleResource{}
	_ resource.ResourceWithValidateConfig = &ApplicationScanScheduleResource{}
)

// ApplicationScanScheduleResource manages a single scheduled scan ("task") under an
// application's scan task-service. The task service itself is an application-scoped
// singleton, registered automatically when the application itself is created (see
// britive_application's task_service_id and scan_enabled) - this resource never creates it
// directly, and never caches its ID in state, instead re-resolving it from application_id on
// every Create/Read/Update/Delete/Import via resolveTaskServiceID. This mirrors
// resourcemanager.ScheduleScanResource's relationship to ResourceTypeResource.
//
// The whole task service's enabled/disabled toggle (POST .../enabled-statuses,
// .../disabled-statuses) is application-wide, not scoped to any one schedule, so it is
// deliberately NOT part of this resource - see scan_enabled on britive_application instead.
type ApplicationScanScheduleResource struct {
	client *britive.Client
}

// ApplicationScanScheduleResourceModel describes the resource data model. Field naming
// mirrors resourcemanager.ScheduleScanResourceModel (day_of_week/day_of_month) since both
// resources configure tasks on the same underlying scan task-service API; hour_interval has
// no resource-manager equivalent, since Hourly is an application-only frequency_type.
type ApplicationScanScheduleResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	TaskID        types.String `tfsdk:"task_id"`
	Name          types.String `tfsdk:"name"`
	FrequencyType types.String `tfsdk:"frequency_type"`
	DayOfWeek     types.String `tfsdk:"day_of_week"`
	DayOfMonth    types.Int64  `tfsdk:"day_of_month"`
	HourInterval  types.Int64  `tfsdk:"hour_interval"`
	StartTime     types.String `tfsdk:"start_time"`
	OrgScan       types.Bool   `tfsdk:"org_scan"`
	// A types.Set, not a plain Go slice: associations is a block populated via a nested
	// "dynamic" block whose for_each is itself derived from this resource's own for_each (e.g.
	// each.value.associations). Terraform core can't statically resolve that repetition count
	// during validate/plan, so it represents the whole block collection as unknown - a plain
	// []ApplicationScanAssociationModel can't hold that ("Value Conversion Error ... Suggested
	// Type: basetypes.SetValue"), so keep it as the framework's own attr.Value and convert
	// to/from []ApplicationScanAssociationModel manually (see
	// applicationScanAssociationsFromSet/applicationScanAssociationsToSet) wherever concrete
	// elements are needed. Mirrors resourcemanager.ScheduleScanResourceModel.ResourceLabels.
	Associations types.Set   `tfsdk:"associations"`
	NextRun      types.Int64 `tfsdk:"next_run"`
}

// ApplicationScanAssociationModel is a single association entry restricting a scan to one
// Environment or EnvironmentGroup. Named "association" (not "scope") to match the terminology
// britive_profile/britive_profile_policy already use for assigning environments/environment
// groups - the backend API itself still calls this "scope" (see
// britive.ApplicationScanScheduleProperties.Scope), so that name is kept at the client-go
// layer; only the Terraform-facing name changes here.
type ApplicationScanAssociationModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// applicationScanAssociationObjectType is the tftypes shape of one associations block element,
// mirroring ApplicationScanAssociationModel's tfsdk tags.
var applicationScanAssociationObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	},
}

// applicationScanAssociationsFromSet decodes an associations set into concrete
// []ApplicationScanAssociationModel. Returns nil without error for a null/unknown set (unknown
// occurs at plan time - see Associations's doc comment - and is always fully resolved again by
// apply time).
func applicationScanAssociationsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []ApplicationScanAssociationModel {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var associations []ApplicationScanAssociationModel
	diags.Append(set.ElementsAs(ctx, &associations, false)...)
	return associations
}

// applicationScanAssociationsToSet is applicationScanAssociationsFromSet's inverse.
func applicationScanAssociationsToSet(ctx context.Context, associations []ApplicationScanAssociationModel, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, applicationScanAssociationObjectType, associations)
	diags.Append(d...)
	return set
}

// NewApplicationScanScheduleResource is a helper function to simplify the provider implementation.
func NewApplicationScanScheduleResource() resource.Resource {
	return &ApplicationScanScheduleResource{}
}

// Metadata returns the resource type name.
func (r *ApplicationScanScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_scan_schedule"
}

// Schema defines the schema for the resource.
func (r *ApplicationScanScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a scheduled scan for a Britive application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the scan schedule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the associated application.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"task_id": schema.StringAttribute{
				Description: "The unique identifier of the scheduled scan task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the scheduled scan.",
				Required:    true,
			},
			"frequency_type": schema.StringAttribute{
				Description: "How often the scan runs. One of Hourly, Daily, Weekly, Monthly (case-insensitive).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Hourly", "Daily", "Weekly", "Monthly"),
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
					stringvalidator.ConflictsWith(path.MatchRoot("day_of_month"), path.MatchRoot("hour_interval")),
				},
			},
			"day_of_month": schema.Int64Attribute{
				Description: "The day of the month (1-31) the scan runs. Required when frequency_type = \"Monthly\"; must be unset otherwise. The valid range is enforced by the API, not this provider.",
				Optional:    true,
			},
			"hour_interval": schema.Int64Attribute{
				Description: "Number of hours between scans (e.g. 5 = every 5 hours). Required when frequency_type = \"Hourly\"; must be unset otherwise. Has no equivalent on britive_resource_manager_resource_type_schedule_scan, since Hourly is an application-only frequency_type.",
				Optional:    true,
			},
			"start_time": schema.StringAttribute{
				Description: "The time of day the scan runs, in 24-hour \"HH:MM\" format. Required for Daily/Weekly/Monthly; must be unset for Hourly, which runs on its own interval instead.",
				Optional:    true,
			},
			"org_scan": schema.BoolAttribute{
				Description: "Whether to scan the entire organization, ignoring associations. Left unmanaged when omitted from config - the provider only sends this field to the API when it's explicitly present in config; the exported value otherwise just reflects whatever the task's actual orgScan status already is.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"next_run": schema.Int64Attribute{
				Description: "The next scheduled run timestamp (epoch milliseconds).",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				Description: "Environments/environment groups the scan is restricted to. Omit entirely (and set org_scan = true) to scan the whole organization.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "One of Environment, EnvironmentGroup (case-insensitive).",
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive("Environment", "EnvironmentGroup"),
							},
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The environment or environment group, by name or ID (mirrors britive_profile's associations block).",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ApplicationScanScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig validates the resource configuration. day_of_week/day_of_month/hour_interval
// and start_time are gated by frequency_type: Hourly requires hour_interval set and start_time
// unset (day_of_week/day_of_month must also be unset - guaranteed by day_of_week's own
// ConflictsWith); Daily requires all three unset and start_time set; Weekly requires
// day_of_week and start_time set; Monthly requires day_of_month and start_time set. Mirrors
// resourcemanager.ScheduleScanResource.ValidateConfig, extended for the Hourly/hour_interval
// case that has no resource-manager equivalent.
func (r *ApplicationScanScheduleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ApplicationScanScheduleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.FrequencyType.IsUnknown() || data.StartTime.IsUnknown() ||
		data.DayOfWeek.IsUnknown() || data.DayOfMonth.IsUnknown() || data.HourInterval.IsUnknown() {
		return
	}

	hasStartTime := !data.StartTime.IsNull() && data.StartTime.ValueString() != ""
	hasDayOfWeek := !data.DayOfWeek.IsNull() && data.DayOfWeek.ValueString() != ""
	hasDayOfMonth := !data.DayOfMonth.IsNull()
	hasHourInterval := !data.HourInterval.IsNull()

	switch strings.ToLower(data.FrequencyType.ValueString()) {
	case "hourly":
		if hasStartTime {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Hourly"' requires start_time to be unset`,
			)
		}
		if !hasHourInterval {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Hourly"' requires hour_interval to be set`,
			)
		}
	case "daily":
		if !hasStartTime {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Daily"' requires start_time to be set`,
			)
		}
		if hasDayOfWeek || hasDayOfMonth || hasHourInterval {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Daily"' requires day_of_week, day_of_month, and hour_interval to be unset`,
			)
		}
	case "weekly":
		if !hasStartTime {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Weekly"' requires start_time to be set`,
			)
		}
		if !hasDayOfWeek {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Weekly"' requires day_of_week to be set`,
			)
		}
		if hasHourInterval {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Weekly"' cannot be combined with hour_interval`,
			)
		}
	case "monthly":
		if !hasStartTime {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Monthly"' requires start_time to be set`,
			)
		}
		if !hasDayOfMonth {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Monthly"' requires day_of_month to be set`,
			)
		}
		if hasHourInterval {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'frequency_type = "Monthly"' cannot be combined with hour_interval`,
			)
		}
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ApplicationScanScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationScanScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read separately from plan: org_scan is only ever sent to the API when explicitly present
	// in config, not just carried forward from state - see buildTaskPayload's doc comment.
	var config ApplicationScanScheduleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueString()

	applicationType, capabilities, err := r.resolveApplicationScanScheduleCapabilities(applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Application", err.Error())
		return
	}
	if err := validateApplicationScanScheduleConfig(applicationType, capabilities, &plan, config.OrgScan); err != nil {
		resp.Diagnostics.AddError("Unsupported Application Scan Schedule Configuration", err.Error())
		return
	}

	// Application creation registers the scan task service automatically, but asynchronously -
	// retry, since a lookup immediately after creation (e.g. when this schedule is created in
	// the same apply as its application) can transiently 404/error before it exists yet.
	taskService, err := applicationscan.GetTaskServiceWithRetry(r.client, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Task Service", err.Error())
		return
	}
	taskServiceID := taskService.TaskServiceID

	task := r.buildTaskPayload(ctx, &plan, config.OrgScan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Creating application scan schedule task for application: %s", applicationID)

	created, err := r.client.CreateApplicationScanTask(taskServiceID, task)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Application Scan Schedule", err.Error())
		return
	}

	plan.TaskID = types.StringValue(created.TaskID)
	plan.ID = types.StringValue(applicationScanScheduleCompositeID(applicationID, created.TaskID))

	r.mapModelToResource(created, &plan, false)

	log.Printf("[INFO] Created application scan schedule task: %s", created.TaskID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ApplicationScanScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationScanScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	taskID := state.TaskID.ValueString()

	taskServiceID, err := r.resolveTaskServiceID(applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Task Service", err.Error())
		return
	}

	log.Printf("[INFO] Reading application scan schedule task: %s", taskID)

	task, err := r.client.GetApplicationScanTask(taskServiceID, taskID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Application Scan Schedule", err.Error())
		return
	}

	priorAssociations := applicationScanAssociationsFromSet(ctx, state.Associations, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.mapModelToResource(task, &state, false)
	associations, err := r.refreshApplicationScanAssociations(applicationID, task.Properties.Scope, priorAssociations)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Schedule Associations", err.Error())
		return
	}
	state.Associations = applicationScanAssociationsToSet(ctx, associations, &resp.Diagnostics)
	state.StartTime = formatApplicationScanStartTime(task.StartTime, task.FrequencyType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ApplicationScanScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationScanScheduleResourceModel
	var state ApplicationScanScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read separately from plan/state: org_scan is only ever sent to the API when explicitly
	// present in config, not just carried forward from state (via its UseStateForUnknown plan
	// modifier) - see buildTaskPayload's doc comment.
	var config ApplicationScanScheduleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	taskID := state.TaskID.ValueString()

	applicationType, capabilities, err := r.resolveApplicationScanScheduleCapabilities(applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Application", err.Error())
		return
	}
	if err := validateApplicationScanScheduleConfig(applicationType, capabilities, &plan, config.OrgScan); err != nil {
		resp.Diagnostics.AddError("Unsupported Application Scan Schedule Configuration", err.Error())
		return
	}

	taskServiceID, err := r.resolveTaskServiceID(applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Task Service", err.Error())
		return
	}

	task := r.buildTaskPayload(ctx, &plan, config.OrgScan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Updating application scan schedule task: %s", taskID)

	updated, err := r.client.UpdateApplicationScanTask(taskServiceID, taskID, task)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Application Scan Schedule", err.Error())
		return
	}

	plan.ID = state.ID
	plan.TaskID = state.TaskID

	r.mapModelToResource(updated, &plan, false)

	log.Printf("[INFO] Updated application scan schedule task: %s", taskID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ApplicationScanScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationScanScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	taskID := state.TaskID.ValueString()

	taskServiceID, err := r.resolveTaskServiceID(applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Task Service", err.Error())
		return
	}

	log.Printf("[INFO] Deleting application scan schedule task: %s", taskID)

	if err := r.client.DeleteApplicationScanTask(taskServiceID, taskID); err != nil {
		resp.Diagnostics.AddError("Error Deleting Application Scan Schedule", err.Error())
		return
	}

	log.Printf("[INFO] Deleted application scan schedule task: %s", taskID)
}

// ImportState imports the resource state.
func (r *ApplicationScanScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	applicationID, taskID, err := parseApplicationScanScheduleCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	taskServiceID, err := r.resolveTaskServiceID(applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Task Service", err.Error())
		return
	}

	log.Printf("[INFO] Importing application scan schedule task: %s", taskID)

	task, err := r.client.GetApplicationScanTask(taskServiceID, taskID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Application Scan Schedule Not Found", fmt.Sprintf("Task %s not found under application %s", taskID, applicationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Application Scan Schedule", err.Error())
		return
	}

	var state ApplicationScanScheduleResourceModel
	state.ID = types.StringValue(applicationScanScheduleCompositeID(applicationID, taskID))
	state.ApplicationID = types.StringValue(applicationID)
	state.TaskID = types.StringValue(taskID)

	r.mapModelToResource(task, &state, true)
	associations, err := r.refreshApplicationScanAssociations(applicationID, task.Properties.Scope, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Application Scan Schedule Associations", err.Error())
		return
	}
	state.Associations = applicationScanAssociationsToSet(ctx, associations, &resp.Diagnostics)
	state.StartTime = formatApplicationScanStartTime(task.StartTime, task.FrequencyType)

	log.Printf("[INFO] Imported application scan schedule task: %s", taskID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// applicationScanScheduleCapabilities describes whether org_scan and associations are
// supported for a given application, derived from live catalog flags (see
// applicationScanScheduleCapabilitiesFromFlags) rather than a static application_type table -
// the backend enforces these rules itself, so this is used only to fail fast with a clear
// error instead of surfacing whatever generic error the backend would otherwise return, and
// to keep working automatically for application types added after this code is written.
type applicationScanScheduleCapabilities struct {
	OrgScan      bool
	Associations bool
}

// applicationScanScheduleCapabilitiesFromFlags derives org_scan/associations support from an
// application's catalog flags (GetApplication's response, under "catalogApplication" - see
// britive.Properties), per this business rule:
//
//	supportsEnvironmentScanning=true,  requiresHierarchicalModel=false -> org_scan and associations both supported
//	supportsEnvironmentScanning=false, requiresHierarchicalModel=false -> only associations supported
//	supportsEnvironmentScanning=true,  requiresHierarchicalModel=true  -> neither supported
//	supportsEnvironmentScanning=false, requiresHierarchicalModel=true  -> org_scan and associations both supported
//
// Equivalently: OrgScan is supportsEnvironmentScanning XOR requiresHierarchicalModel;
// Associations is supported except when both flags are true.
func applicationScanScheduleCapabilitiesFromFlags(supportsEnvironmentScanning, requiresHierarchicalModel bool) applicationScanScheduleCapabilities {
	return applicationScanScheduleCapabilities{
		OrgScan:      supportsEnvironmentScanning != requiresHierarchicalModel,
		Associations: !(supportsEnvironmentScanning && requiresHierarchicalModel),
	}
}

// resolveApplicationScanScheduleCapabilities resolves applicationID's application type (for
// use in validateApplicationScanScheduleConfig's error messages) and its org_scan/associations
// support via GetApplication's catalog flags. Re-resolved on every Create/Update rather than
// cached, since these flags describe the application's catalog type, which callers may expect
// to reflect the live backend rather than a value captured once at an earlier apply.
func (r *ApplicationScanScheduleResource) resolveApplicationScanScheduleCapabilities(applicationID string) (applicationType string, capabilities applicationScanScheduleCapabilities, err error) {
	application, err := r.client.GetApplication(applicationID)
	if err != nil {
		return "", applicationScanScheduleCapabilities{}, err
	}
	capabilities = applicationScanScheduleCapabilitiesFromFlags(application.Properties.SupportsEnvironmentScanning, application.Properties.RequiresHierarchicalModel)
	return application.CatalogAppName, capabilities, nil
}

// validateApplicationScanScheduleConfig checks plan's associations usage, and configOrgScan's
// org_scan usage, against capabilities, naming applicationType in the error so a practitioner
// managing several application types doesn't have to guess which one tripped it. configOrgScan
// is the raw config value, not plan.OrgScan: org_scan is Computed with a UseStateForUnknown
// plan modifier, so once the server has ever echoed back a value for it, plan.OrgScan would
// carry that forward on every later Update even when config never set it - checking config
// here is what makes "never configured" actually mean "never sent", matching
// buildTaskPayload's own reasoning for the same distinction. associations has no such concern:
// it's a plain block with no Computed flag, so plan.Associations always reflects config
// directly.
func validateApplicationScanScheduleConfig(applicationType string, capabilities applicationScanScheduleCapabilities, plan *ApplicationScanScheduleResourceModel, configOrgScan types.Bool) error {
	hasAssociations := !plan.Associations.IsNull() && !plan.Associations.IsUnknown() && len(plan.Associations.Elements()) > 0
	if !capabilities.Associations && hasAssociations {
		return fmt.Errorf("associations are not supported for application type %q - leave it unset", applicationType)
	}
	if !capabilities.OrgScan && !configOrgScan.IsNull() && !configOrgScan.IsUnknown() {
		return fmt.Errorf("org_scan is not supported for application type %q - leave it unset", applicationType)
	}
	return nil
}

// resolveTaskServiceID resolves applicationID's scan task-service ID - application creation
// registers this automatically, so this is expected to always succeed. Never stored in this
// resource's own state - re-resolved on every call so the resource stays correct even if the
// task service's own ID were ever to change server-side (mirrors
// resourcemanager.ScheduleScanResource.resolveTaskServiceID). britive_application caches its
// own copy of the same ID in task_service_id since it owns the task service's lifecycle;
// this resource does not, since it may be applied independently of any britive_application
// resource in the same configuration. Create uses applicationscan.GetTaskServiceWithRetry
// instead of this method - see that function's doc comment for why.
func (r *ApplicationScanScheduleResource) resolveTaskServiceID(applicationID string) (string, error) {
	taskService, err := r.client.GetApplicationScanTaskService(applicationID)
	if err != nil {
		return "", err
	}
	return taskService.TaskServiceID, nil
}

// buildTaskPayload maps the plan into the API's create/update request shape, deriving
// frequencyInterval from day_of_week/day_of_month/hour_interval based on frequency_type -
// mirrors resourcemanager.ScheduleScanResource.buildTaskPayload, reusing the same
// schedulescan.WeekdayToInterval mapping for the Weekly case. association entries are resolved
// from name-or-ID to the API's expected raw ID via resolveApplicationScanAssociations; a
// resolution failure is reported through diags and yields a zero-value task, matching the
// pattern resourcemanager.ScheduleScanResource.buildTaskPayload uses for a duplicate
// resource_labels block. configOrgScan is the raw config value, not plan.OrgScan: org_scan is
// Computed with a UseStateForUnknown plan modifier, so once the server has ever echoed back a
// value for it, plan.OrgScan would carry that forward on every later Update even when config
// never set it - checking config here is what makes "never configured" actually mean "never
// sent".
func (r *ApplicationScanScheduleResource) buildTaskPayload(ctx context.Context, plan *ApplicationScanScheduleResourceModel, configOrgScan types.Bool, diags *diag.Diagnostics) britive.ApplicationScheduleScanTask {
	applicationID := plan.ApplicationID.ValueString()

	associations := applicationScanAssociationsFromSet(ctx, plan.Associations, diags)
	if diags.HasError() {
		return britive.ApplicationScheduleScanTask{}
	}

	resolvedAssociations, err := r.resolveApplicationScanAssociations(applicationID, associations)
	if err != nil {
		diags.AddError("Error Resolving Application Scan Schedule Associations", err.Error())
		return britive.ApplicationScheduleScanTask{}
	}

	task := britive.ApplicationScheduleScanTask{
		Name:          plan.Name.ValueString(),
		FrequencyType: schedulescan.CanonicalCasing(plan.FrequencyType.ValueString(), "Hourly", "Daily", "Weekly", "Monthly"),
		Properties: britive.ApplicationScanScheduleProperties{
			AppID: applicationID,
			// The backend API calls this "scope" - see ApplicationScanScheduleProperties.Scope's
			// doc comment. resolvedAssociations is the Terraform-facing "associations" block,
			// resolved to the wire shape the API expects.
			Scope: resolvedAssociations,
		},
	}

	// org_scan is only ever sent when explicitly present in config - see this function's own
	// doc comment for why configOrgScan (not plan.OrgScan) is what's checked here.
	if !configOrgScan.IsNull() && !configOrgScan.IsUnknown() {
		orgScan := plan.OrgScan.ValueBool()
		task.Properties.OrgScan = &orgScan
	}

	switch strings.ToLower(plan.FrequencyType.ValueString()) {
	case "hourly":
		task.StartTime = nil
		interval := int(plan.HourInterval.ValueInt64())
		task.FrequencyInterval = &interval
	case "weekly":
		st := plan.StartTime.ValueString()
		task.StartTime = &st
		interval := schedulescan.WeekdayToInterval[strings.ToLower(plan.DayOfWeek.ValueString())]
		task.FrequencyInterval = &interval
	case "monthly":
		st := plan.StartTime.ValueString()
		task.StartTime = &st
		interval := int(plan.DayOfMonth.ValueInt64())
		task.FrequencyInterval = &interval
	default: // daily
		st := plan.StartTime.ValueString()
		task.StartTime = &st
		task.FrequencyInterval = nil
	}

	return task
}

// mapModelToResource maps the API's scan schedule task detail onto Terraform state, except
// for associations and start_time - see refreshApplicationScanAssociations and
// formatApplicationScanStartTime for why those are handled separately. frequency_type and
// day_of_week are validated case-insensitively but sent to the API in one fixed
// casing/mapping, so the value echoed back is not necessarily what the user typed - when not
// importing, and the API's value case-insensitively/numerically matches what's already in
// state, the user's original casing/abbreviation is preserved instead of being overwritten.
// Mirrors resourcemanager.ScheduleScanResource.mapModelToResource, reusing the same
// schedulescan helpers.
func (r *ApplicationScanScheduleResource) mapModelToResource(task *britive.ApplicationScheduleScanTaskDetail, state *ApplicationScanScheduleResourceModel, imported bool) {
	state.Name = types.StringValue(task.Name)

	if !imported && strings.EqualFold(state.FrequencyType.ValueString(), task.FrequencyType) {
		// Prior state already matches (case-insensitively) - keep the user's casing.
	} else {
		state.FrequencyType = types.StringValue(task.FrequencyType)
	}

	switch strings.ToLower(task.FrequencyType) {
	case "hourly":
		state.DayOfWeek = types.StringNull()
		state.DayOfMonth = types.Int64Null()
		if task.FrequencyInterval != nil {
			state.HourInterval = types.Int64Value(int64(*task.FrequencyInterval))
		} else {
			state.HourInterval = types.Int64Null()
		}
	case "weekly":
		state.DayOfMonth = types.Int64Null()
		state.HourInterval = types.Int64Null()
		if task.FrequencyInterval == nil {
			state.DayOfWeek = types.StringNull()
			break
		}
		canonical := schedulescan.IntervalToWeekday[*task.FrequencyInterval]
		if !imported && !state.DayOfWeek.IsNull() &&
			schedulescan.WeekdayToInterval[strings.ToLower(state.DayOfWeek.ValueString())] == *task.FrequencyInterval {
			// Prior state already resolves to the same day - keep the user's original
			// casing/abbreviation (e.g. "Mon" vs "Monday").
		} else {
			state.DayOfWeek = types.StringValue(canonical)
		}
	case "monthly":
		state.DayOfWeek = types.StringNull()
		state.HourInterval = types.Int64Null()
		if task.FrequencyInterval != nil {
			state.DayOfMonth = types.Int64Value(int64(*task.FrequencyInterval))
		} else {
			state.DayOfMonth = types.Int64Null()
		}
	default: // daily
		state.DayOfWeek = types.StringNull()
		state.DayOfMonth = types.Int64Null()
		state.HourInterval = types.Int64Null()
	}

	if task.Properties.OrgScan != nil {
		state.OrgScan = types.BoolValue(*task.Properties.OrgScan)
	} else {
		// Confirmed by manual API check: the response omits orgScan entirely (not even
		// false) for application types that don't support it (e.g. AWS Standalone) - null
		// represents "no known value" more accurately here than false, which would imply a
		// real, toggleable-but-currently-off status.
		state.OrgScan = types.BoolNull()
	}
	state.NextRun = types.Int64Value(task.NextRun)
}

// resolveApplicationScanAssociations resolves each association entry's value - accepted as
// either a name or a raw environment/environment-group ID, mirroring
// ProfileResource.saveProfileAssociations - into the actual ID the API's properties.scope[].value
// expects (the backend API itself calls this "scope"; see
// britive.ApplicationScanScheduleProperties.Scope). Fetches the application's environment tree
// once via GetApplicationRootEnvironmentGroup and matches each entry against either Name or ID
// within the list for its association type (Environment vs EnvironmentGroup). Collects every
// entry that matched neither into a single error, rather than failing on the first one, so a
// practitioner sees every bad value in one apply.
func (r *ApplicationScanScheduleResource) resolveApplicationScanAssociations(applicationID string, associations []ApplicationScanAssociationModel) ([]britive.ApplicationScanScope, error) {
	resolved := make([]britive.ApplicationScanScope, 0, len(associations))
	if len(associations) == 0 {
		return resolved, nil
	}

	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		return nil, err
	}

	unmatched := make([]string, 0)
	for _, s := range associations {
		associationType := schedulescan.CanonicalCasing(s.Type.ValueString(), "Environment", "EnvironmentGroup")
		associationValue := s.Value.ValueString()

		var rootAssociations []britive.Association
		if associationType == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}

		found := false
		for _, aeg := range rootAssociations {
			if aeg.Name == associationValue || aeg.ID == associationValue {
				resolved = append(resolved, britive.ApplicationScanScope{
					Type:  associationType,
					Value: aeg.ID,
				})
				found = true
				break
			}
		}
		if !found {
			unmatched = append(unmatched, fmt.Sprintf("%s=%s", associationType, associationValue))
		}
	}

	if len(unmatched) > 0 {
		return nil, errs.NewNotFoundErrorf("association %v", unmatched)
	}

	return resolved, nil
}

// refreshApplicationScanAssociations rebuilds the associations block list from the API's live
// properties.scope (see britive.ApplicationScanScheduleProperties.Scope's doc comment for why
// the wire name differs), converting each entry's raw environment/environment-group ID back to
// its name - unless priorAssociations shows the practitioner's config used the raw ID form for
// that entry, in which case the ID is preserved to avoid a perpetual name/ID diff. Mirrors
// ProfileResource.mapProfileAssociationsModelToResource. Used only by Read/ImportState, never
// by Create/Update: see buildTaskPayload/resolveApplicationScanAssociations for why those
// accept (and must return exactly) whichever form - name or ID - the user configured.
// consumed tracks which priorAssociations entries have already been matched to an earlier
// entry in this same call. Without it, two association entries that resolve to the same
// environment/environment-group (e.g. one block by name, one by that same environment's raw
// ID) would both match the same single prior ID-form entry and both come back as identical
// {Type, ID} pairs - not just a diff, but a hard "Duplicate Set Element" planning error, since
// associations is a Set and the framework rejects a returned state containing two identical
// members outright. Marking an entry consumed once it's chosen forces the next entry that
// would otherwise reuse it to fall through to the (still-correct, and distinct) name-form
// default instead.
func (r *ApplicationScanScheduleResource) refreshApplicationScanAssociations(applicationID string, scope []britive.ApplicationScanScope, priorAssociations []ApplicationScanAssociationModel) ([]ApplicationScanAssociationModel, error) {
	associationList := make([]ApplicationScanAssociationModel, 0, len(scope))
	if len(scope) == 0 {
		return associationList, nil
	}

	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		return nil, err
	}

	consumed := make([]bool, len(priorAssociations))

	for _, s := range scope {
		var rootAssociations []britive.Association
		if s.Type == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}

		var a *britive.Association
		for i := range rootAssociations {
			if rootAssociations[i].ID == s.Value {
				a = &rootAssociations[i]
				break
			}
		}
		if a == nil {
			return nil, errs.NewNotFoundErrorf("association %s %s", s.Type, s.Value)
		}

		value := a.Name
		for i, prior := range priorAssociations {
			if consumed[i] {
				continue
			}
			if prior.Type.ValueString() == s.Type && prior.Value.ValueString() == a.ID {
				value = a.ID
				consumed[i] = true
				break
			}
		}

		associationList = append(associationList, ApplicationScanAssociationModel{
			Type:  types.StringValue(s.Type),
			Value: types.StringValue(value),
		})
	}

	return associationList, nil
}

// formatApplicationScanStartTime renders the API's [hour, minute] pair as a "HH:MM" string,
// reusing schedulescan.FormatStartTime. Used only by Read/ImportState, for the same reason as
// refreshApplicationScanAssociations: start_time has no Computed flag, so Create/Update must return
// exactly what was planned rather than a reformatted value. For Hourly schedules the server
// assigns an arbitrary, not-user-meaningful startTime (confirmed by capture: the create
// request sends startTime = null and gets a non-null value back) - normalized to null here so
// it never appears as a perpetual diff. This Hourly override has no resource-manager
// equivalent, since resource-manager has no Hourly frequency_type.
func formatApplicationScanStartTime(startTime []int, frequencyType string) types.String {
	if strings.EqualFold(frequencyType, "hourly") {
		return types.StringNull()
	}
	return schedulescan.FormatStartTime(startTime)
}

// applicationScanScheduleCompositeID builds the resource's `id` value from
// applicationID/taskID.
func applicationScanScheduleCompositeID(applicationID, taskID string) string {
	return fmt.Sprintf("apps/%s/scan-schedules/%s", applicationID, taskID)
}

// parseApplicationScanScheduleCompositeID extracts applicationID and taskID from either the
// composite ID ("apps/{application_id}/scan-schedules/{task_id}") or a bare
// "{application_id}/{task_id}" pair, for use on import.
func parseApplicationScanScheduleCompositeID(id string) (applicationID string, taskID string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 4:
		if parts[0] != "apps" || parts[2] != "scan-schedules" {
			return "", "", errs.NewInvalidResourceIDError("application scan schedule", id)
		}
		return parts[1], parts[3], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", errs.NewInvalidResourceIDError("application scan schedule", id)
	}
}
