package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceTypeResource is the resource implementation.
type ResourceTypeResource struct {
	client *britive.Client
}

// ResourceTypeResourceModel describes the resource data model.
type ResourceTypeResourceModel struct {
	ID              types.String                 `tfsdk:"id"`
	Name            types.String                 `tfsdk:"name"`
	Description     types.String                 `tfsdk:"description"`
	Icon            types.String                 `tfsdk:"icon"`
	TaskServiceID   types.String                 `tfsdk:"task_service_id"`
	ScanEnabled     types.Bool                   `tfsdk:"scan_enabled"`
	RotationEnabled types.Bool                   `tfsdk:"rotation_enabled"`
	Parameters      []ResourceTypeParameterModel `tfsdk:"parameters"`
}

type ResourceTypeParameterModel struct {
	ParamName   types.String `tfsdk:"param_name"`
	ParamType   types.String `tfsdk:"param_type"`
	IsMandatory types.Bool   `tfsdk:"is_mandatory"`
}

// NewResourceTypeResource is a helper function to simplify the provider implementation.
func NewResourceTypeResource() resource.Resource {
	return &ResourceTypeResource{}
}

// Metadata returns the resource type name.
func (r *ResourceTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_type"
}

// Schema defines the schema for the resource.
func (r *ResourceTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager resource type",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource type ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of Britive resource type",
				Required:    true,
				Validators: []validator.String{
					validators.Alphanumeric(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the Britive resource type",
				Optional:    true,
			},
			"icon": schema.StringAttribute{
				Description: "Icon of Britive resource type (SVG format)",
				Optional:    true,
				Validators: []validator.String{
					validators.SVG(),
				},
			},
			"task_service_id": schema.StringAttribute{
				Description: "The ID of this resource type's scan task service. Registered automatically when the resource type is created and removed automatically when it's deleted - not independently managed by this provider. Used internally to operate on britive_resource_manager_resource_type_schedule_scan and scan_enabled.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scan_enabled": schema.BoolAttribute{
				Description: "Whether scheduled scanning is enabled for this resource type. Left unmanaged when omitted from config - the provider never enables or disables scanning unless this is explicitly set, so existing resource types (with or without scheduled scans already configured some other way) are unaffected. The scan task service is registered automatically when the resource type is created, so this can be set in the very same apply that creates the resource type (and any of its schedule scans) - no multi-step apply required. Removing this argument from config (after previously setting it) disables scanning.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"rotation_enabled": schema.BoolAttribute{
				Description: "Whether rotation is enabled for this resource type's rotation templates. Left unmanaged when omitted from config - the provider only sends this field when it's explicitly present, so existing resource types (with or without rotation already turned on some other way) are unaffected. Independent of any britive_resource_manager_resource_type_rotation_template existing, so it can be set at resource type creation. Removing this argument from config (after previously setting it) disables rotation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"parameters": schema.SetNestedBlock{
				Description: "Parameters/Fields of the resource type",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"param_name": schema.StringAttribute{
							Description: "The parameter name",
							Required:    true,
							Validators: []validator.String{
								validators.Alphanumeric(),
							},
						},
						"param_type": schema.StringAttribute{
							Description: "The parameter type (string, password, ip-cidr, regex-pattern, list)",
							Required:    true,
						},
						"is_mandatory": schema.BoolAttribute{
							Description: "Whether the parameter is mandatory",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ResourceTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan handles the one scan_enabled/rotation_enabled transition that
// UseStateForUnknown's own carry-forward gets wrong: removing the argument from config
// after it was previously tracked as enabled. Without this override, an absent config value
// simply carries the prior state's value forward unchanged (UseStateForUnknown), which would
// silently leave scanning/rotation on forever once enabled - instead, removing the argument
// is planned as an explicit transition to false. In every other case (never configured, or
// configured and unchanged) the attribute is left alone here; see setScanEnabled and
// mapResourceToModel's RotationEnabled handling for why this resource otherwise only ever
// touches this state when explicitly asked to.
//
// Known limitation: this can't distinguish "the user removed a scan_enabled/rotation_enabled
// = true they used to manage in Terraform" from "it was never in this resource's config, but
// happens to already be enabled some other way (e.g. via the UI)" - both look identical
// (state has enabled=true, config has no value). The next apply that touches this resource
// for any reason will turn it off in the second case too.
func (r *ResourceTypeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		// Destroy, or Create - nothing to have "removed" yet.
		return
	}

	var plan, config, state ResourceTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modified := false

	if config.ScanEnabled.IsNull() && !state.ScanEnabled.IsNull() && state.ScanEnabled.ValueBool() {
		plan.ScanEnabled = types.BoolValue(false)
		modified = true
	}

	if config.RotationEnabled.IsNull() && !state.RotationEnabled.IsNull() && state.RotationEnabled.ValueBool() {
		plan.RotationEnabled = types.BoolValue(false)
		modified = true
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ResourceTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map resource to API model
	resourceType := r.mapResourceToModel(&plan)

	log.Printf("[INFO] Adding new resource type: %#v", resourceType)

	// Create resource type
	rto, err := r.client.CreateResourceType(resourceType)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Type", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new resource type: %#v", rto)

	// Add icon if provided
	if !plan.Icon.IsNull() && plan.Icon.ValueString() != "" {
		log.Printf("[INFO] Adding icon to resource type: %#v", rto)
		userSVG := plan.Icon.ValueString()
		err = r.client.AddRemoveIcon(rto.ResourceTypeID, userSVG)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Icon", err.Error())
			// Cleanup: delete the resource type if icon add fails
			if delErr := r.client.DeleteResourceType(rto.ResourceTypeID); delErr != nil {
				resp.Diagnostics.AddError("Error Cleaning Up Resource Type", delErr.Error())
			}
			return
		}
		log.Printf("[INFO] Added icon to resource type: %#v", rto)
	}

	// Set ID
	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", rto.ResourceTypeID))

	// Read back to get computed values
	resourceTypeRead, err := r.client.GetResourceType(rto.ResourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(resourceTypeRead, &plan, false)

	// The backend registers this resource type's scan task service automatically at
	// creation time (rto.TaskServiceID) and deletes it automatically when the resource type
	// is deleted - nothing to bootstrap or clean up here. Fetching it via
	// GetScheduleScanTaskService also gets its actual current enabled state in the same
	// call, rather than assuming a default. Because the task service now exists immediately,
	// scan_enabled can be set in this very same apply - unlike the old lazily-created task
	// service, there is no dependency on a schedule scan existing first.
	taskService, err := r.client.GetScheduleScanTaskService(rto.ResourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type Scan Task Service", err.Error())
		return
	}
	plan.TaskServiceID = types.StringValue(taskService.TaskServiceID)

	// scan_enabled has no schema Default, so plan.ScanEnabled is Unknown here precisely when
	// the user didn't set it in config - in that case, don't touch scan state at all, just
	// reflect the task service's actual just-registered value. A failure setting it doesn't
	// roll back the resource type itself, which was already created successfully; state is
	// still written below so the resource type stays tracked.
	if !plan.ScanEnabled.IsUnknown() && plan.ScanEnabled.ValueBool() != taskService.Enabled {
		enabled, err := r.setScanEnabled(taskService.TaskServiceID, plan.ScanEnabled.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Error Setting Resource Type Scan Status", err.Error())
			plan.ScanEnabled = types.BoolValue(taskService.Enabled)
		} else {
			plan.ScanEnabled = types.BoolValue(enabled)
		}
	} else {
		plan.ScanEnabled = types.BoolValue(taskService.Enabled)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ResourceTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Reading resource type %s", resourceTypeID)

	resourceType, err := r.client.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type", err.Error())
		return
	}

	log.Printf("[INFO] Received resource type %#v", resourceType)

	// Map model to resource
	r.mapModelToResource(resourceType, &state, false)

	// The task service is registered automatically alongside its resource type and deleted
	// along with it (backend cascade), so this is expected to always succeed here.
	taskService, err := r.client.GetScheduleScanTaskService(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type Scan Task Service", err.Error())
		return
	}
	state.TaskServiceID = types.StringValue(taskService.TaskServiceID)
	state.ScanEnabled = types.BoolValue(taskService.Enabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ResourceTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceTypeResourceModel
	var state ResourceTypeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	// Check for changes in main fields. rotation_enabled is included here (not handled
	// separately like scan_enabled) since it's a plain field on this same resource type
	// object, sent via the same PUT - see mapResourceToModel's doc comment for how an
	// unconfigured value is kept out of that PUT entirely rather than clobbering it.
	if !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.RotationEnabled.Equal(state.RotationEnabled) ||
		!parametersEqual(plan.Parameters, state.Parameters) {

		resourceType := r.mapResourceToModel(&plan)

		log.Printf("[INFO] Updating resource type: %#v", resourceType)

		ur, err := r.client.UpdateResourceType(resourceType, resourceTypeID)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Resource Type", err.Error())
			return
		}

		log.Printf("[INFO] Updated resource type: %#v", ur)
	}

	// Handle icon separately
	if !plan.Icon.Equal(state.Icon) {
		log.Printf("[INFO] Updating icon to resource type: %#v", resourceTypeID)
		userSVG := plan.Icon.ValueString()
		err = r.client.AddRemoveIcon(resourceTypeID, userSVG)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Icon", err.Error())
			return
		}
		log.Printf("[INFO] Added icon to resource type: %#v", resourceTypeID)
	}

	// Handle scan_enabled separately. plan.ScanEnabled only differs from state.ScanEnabled
	// here when config explicitly set a value that doesn't match reality, or when
	// ModifyPlan planned an explicit false because scan_enabled was removed from config -
	// in the steady "never configured" case UseStateForUnknown carries state forward
	// unchanged, so no API call happens. A failure here doesn't abort the update - name/
	// description/parameters/icon changes above already succeeded, so surface the error but
	// still read back and persist accurate state below.
	if !plan.ScanEnabled.Equal(state.ScanEnabled) {
		taskServiceID := state.TaskServiceID.ValueString()
		if taskServiceID == "" {
			// Defensive: state predates task_service_id being tracked, or refresh was
			// skipped - resolve it now rather than failing outright.
			taskService, getErr := r.client.GetScheduleScanTaskService(resourceTypeID)
			if getErr != nil {
				resp.Diagnostics.AddError("Error Reading Resource Type Scan Task Service", getErr.Error())
			} else {
				taskServiceID = taskService.TaskServiceID
				plan.TaskServiceID = types.StringValue(taskServiceID)
			}
		}
		if taskServiceID != "" {
			enabled, err := r.setScanEnabled(taskServiceID, plan.ScanEnabled.ValueBool())
			if err != nil {
				resp.Diagnostics.AddError("Error Updating Resource Type Scan Status", err.Error())
				plan.ScanEnabled = state.ScanEnabled
			} else {
				plan.ScanEnabled = types.BoolValue(enabled)
			}
		}
	}

	// Read back to get updated values
	resourceType, err := r.client.GetResourceType(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(resourceType, &plan, false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ResourceTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting resource type: %s", resourceTypeID)

	err = r.client.DeleteResourceType(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Type", err.Error())
		return
	}

	log.Printf("[INFO] Resource type %s deleted", resourceTypeID)
}

// ImportState imports the resource state.
func (r *ResourceTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support format: resource-manager/resource-types/{id}
	importID := req.ID

	if !strings.HasPrefix(importID, "resource-manager/resource-types/") {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'resource-manager/resource-types/{id}', got: %s", importID),
		)
		return
	}

	parts := strings.Split(importID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'resource-manager/resource-types/{id}', got: %s", importID),
		)
		return
	}

	resourceTypeID := parts[2]
	if strings.TrimSpace(resourceTypeID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Resource type ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing resource type: %s", resourceTypeID)

	resourceType, err := r.client.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Type Not Found", fmt.Sprintf("Resource type %s not found", resourceTypeID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Type", err.Error())
		return
	}

	// Set the state
	var state ResourceTypeResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", resourceType.ResourceTypeID))

	// Map model to resource (imported = true to preserve API param types)
	r.mapModelToResource(resourceType, &state, true)

	taskService, err := r.client.GetScheduleScanTaskService(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type Scan Task Service", err.Error())
		return
	}
	state.TaskServiceID = types.StringValue(taskService.TaskServiceID)
	state.ScanEnabled = types.BoolValue(taskService.Enabled)

	log.Printf("[INFO] Imported resource type: %s", resourceTypeID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// setScanEnabled enables or disables scheduled scanning for a resource type's scan task
// service, given its already-resolved taskServiceID, and returns the resulting actual
// enabled state from the API's own response.
func (r *ResourceTypeResource) setScanEnabled(taskServiceID string, enabled bool) (bool, error) {
	var taskService *britive.ScheduleScanTaskService
	var err error
	if enabled {
		taskService, err = r.client.EnableScheduleScanTaskService(taskServiceID)
	} else {
		taskService, err = r.client.DisableScheduleScanTaskService(taskServiceID)
	}
	if err != nil {
		return false, err
	}
	return taskService.Enabled, nil
}

// mapResourceToModel builds the create/update request payload from plan. IsRotationEnabled
// is left nil (omitted from the request) when plan.RotationEnabled is unknown - i.e. never
// configured - so the field simply isn't part of the request at all rather than sending an
// explicit false that would disable rotation as a side effect of an unrelated change. When
// it IS known (either explicitly configured, or carried forward from actual state by
// UseStateForUnknown on Update), it's always sent, since that's either the user's intent or
// a no-op restating the current truth.
func (r *ResourceTypeResource) mapResourceToModel(plan *ResourceTypeResourceModel) britive.ResourceType {
	resourceType := britive.ResourceType{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Parameters:  make([]britive.Parameter, 0),
	}

	if !plan.RotationEnabled.IsUnknown() && !plan.RotationEnabled.IsNull() {
		v := plan.RotationEnabled.ValueBool()
		resourceType.IsRotationEnabled = &v
	}

	for _, param := range plan.Parameters {
		resourceType.Parameters = append(resourceType.Parameters, britive.Parameter{
			ParamName:   param.ParamName.ValueString(),
			ParamType:   strings.ToLower(param.ParamType.ValueString()),
			IsMandatory: param.IsMandatory.ValueBool(),
		})
	}

	return resourceType
}

func (r *ResourceTypeResource) mapModelToResource(resourceType *britive.ResourceType, state *ResourceTypeResourceModel, imported bool) {
	state.Name = types.StringValue(resourceType.Name)
	// Use prior state as fallback when API returns "": if the prior state was null (user never
	// set description), keep null; if it was "" or non-empty, preserve that intent. This avoids
	// the "was '' but now null" inconsistency when the plan had "" for an Optional-only field.
	state.Description = preserveOptionalString(resourceType.Description, state.Description)

	if resourceType.IsRotationEnabled != nil {
		state.RotationEnabled = types.BoolValue(*resourceType.IsRotationEnabled)
	} else {
		state.RotationEnabled = types.BoolValue(false)
	}

	// Build map of user's param types to preserve case (unless imported)
	paramMap := make(map[string]string)
	if !imported {
		for _, param := range state.Parameters {
			paramMap[param.ParamName.ValueString()] = param.ParamType.ValueString()
		}
	}

	// Map parameters
	parameters := make([]ResourceTypeParameterModel, 0)
	for _, param := range resourceType.Parameters {
		paramType := param.ParamType
		if !imported {
			// Use the user's original case for param_type if it exists in state
			if userType, ok := paramMap[param.ParamName]; ok {
				paramType = userType
			}
		}

		parameters = append(parameters, ResourceTypeParameterModel{
			ParamName:   types.StringValue(param.ParamName),
			ParamType:   types.StringValue(paramType),
			IsMandatory: types.BoolValue(param.IsMandatory),
		})
	}

	state.Parameters = parameters
}

func (r *ResourceTypeResource) parseUniqueID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 3 {
		return "", errs.NewInvalidResourceIDError("resource type", id)
	}
	return parts[2], nil
}

// parametersEqual compares two slices of ResourceTypeParameterModel for equality
func parametersEqual(a, b []ResourceTypeParameterModel) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for comparison (order doesn't matter for sets)
	aMap := make(map[string]ResourceTypeParameterModel)
	for _, param := range a {
		key := param.ParamName.ValueString()
		aMap[key] = param
	}

	bMap := make(map[string]ResourceTypeParameterModel)
	for _, param := range b {
		key := param.ParamName.ValueString()
		bMap[key] = param
	}

	if len(aMap) != len(bMap) {
		return false
	}

	for key, aParam := range aMap {
		bParam, ok := bMap[key]
		if !ok {
			return false
		}
		// Compare all fields (case-insensitive for param_type)
		if !strings.EqualFold(aParam.ParamType.ValueString(), bParam.ParamType.ValueString()) ||
			aParam.IsMandatory.ValueBool() != bParam.IsMandatory.ValueBool() {
			return false
		}
	}

	return true
}
