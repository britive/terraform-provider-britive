package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AdvancedSettingsResource{}
	_ resource.ResourceWithConfigure   = &AdvancedSettingsResource{}
	_ resource.ResourceWithImportState = &AdvancedSettingsResource{}
)

func NewAdvancedSettingsResource() resource.Resource {
	return &AdvancedSettingsResource{}
}

type AdvancedSettingsResource struct {
	client *britive.Client
}

type AdvancedSettingsResourceModel struct {
	ID                    types.String                            `tfsdk:"id"`
	ResourceID            types.String                            `tfsdk:"resource_id"`
	ResourceType          types.String                            `tfsdk:"resource_type"`
	JustificationSettings *JustificationSettingsModel             `tfsdk:"justification_settings"`
	ITSM                  *ITSMModel                              `tfsdk:"itsm"`
	IM                    *IMModel                                `tfsdk:"im"`
}

type JustificationSettingsModel struct {
	JustificationID         types.String `tfsdk:"justification_id"`
	IsJustificationRequired types.Bool   `tfsdk:"is_justification_required"`
	JustificationRegex      types.String `tfsdk:"justification_regex"`
}

type ITSMModel struct {
	ITSMID             types.String `tfsdk:"itsm_id"`
	ConnectionID       types.String `tfsdk:"connection_id"`
	ConnectionType     types.String `tfsdk:"connection_type"`
	IsITSMEnabled      types.Bool   `tfsdk:"is_itsm_enabled"`
	ITSMFilterCriteria types.Set    `tfsdk:"itsm_filter_criteria"`
}

type ITSMFilterCriteriaModel struct {
	SupportedTicketType types.String `tfsdk:"supported_ticket_type"`
	Filter              types.String `tfsdk:"filter"`
}

type IMModel struct {
	IMID                   types.String `tfsdk:"im_id"`
	ConnectionID           types.String `tfsdk:"connection_id"`
	ConnectionType         types.String `tfsdk:"connection_type"`
	IsAutoApprovalEnabled  types.Bool   `tfsdk:"is_auto_approval_enabled"`
	EscalationPolicies     types.Set    `tfsdk:"escalation_policies"`
}

func (r *AdvancedSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_advanced_settings"
}

func (r *AdvancedSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive advanced settings for resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the advanced settings.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Required:    true,
				Description: "Britive resource id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type": schema.StringAttribute{
				Required:    true,
				Description: "Britive resource type.",
				Validators: []validator.String{
					stringvalidator.OneOf("APPLICATION", "PROFILE", "PROFILE_POLICY", "RESOURCE_MANAGER_PROFILE", "RESOURCE_MANAGER_PROFILE_POLICY"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"justification_settings": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Resource's Justification Settings.",
				Attributes: map[string]schema.Attribute{
					"justification_id": schema.StringAttribute{
						Computed:    true,
						Description: "Justification Setting ID.",
					},
					"is_justification_required": schema.BoolAttribute{
						Required:    true,
						Description: "Resource justification.",
					},
					"justification_regex": schema.StringAttribute{
						Optional:    true,
						Description: "Resource justification Regular Expression.",
					},
				},
			},
			"itsm": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Resource ITSM Setting.",
				Attributes: map[string]schema.Attribute{
					"itsm_id": schema.StringAttribute{
						Computed:    true,
						Description: "ITSM Setting ID.",
					},
					"connection_id": schema.StringAttribute{
						Required:    true,
						Description: "ITSM Connection id.",
					},
					"connection_type": schema.StringAttribute{
						Required:    true,
						Description: "ITSM Connection type.",
					},
					"is_itsm_enabled": schema.BoolAttribute{
						Required:    true,
						Description: "ITSM enabled flag.",
					},
					"itsm_filter_criteria": schema.SetNestedAttribute{
						Required:    true,
						Description: "ITSM filter criteria.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"supported_ticket_type": schema.StringAttribute{
									Required:    true,
									Description: "Supported ticket type.",
								},
								"filter": schema.StringAttribute{
									Required:    true,
									Description: "Filter (JSON string).",
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
							},
						},
					},
				},
			},
			"im": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Resource IM Setting.",
				Attributes: map[string]schema.Attribute{
					"im_id": schema.StringAttribute{
						Computed:    true,
						Description: "IM Setting ID.",
					},
					"connection_id": schema.StringAttribute{
						Required:    true,
						Description: "IM Connection id.",
					},
					"connection_type": schema.StringAttribute{
						Required:    true,
						Description: "IM Connection type.",
					},
					"is_auto_approval_enabled": schema.BoolAttribute{
						Required:    true,
						Description: "IM auto approval toggle.",
					},
					"escalation_policies": schema.SetAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "IM Escalation Policies.",
					},
				},
			},
		},
	}
}

func (r *AdvancedSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AdvancedSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AdvancedSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := plan.ResourceID.ValueString()
	resourceType := strings.ToLower(plan.ResourceType.ValueString())

	// Build advanced settings from plan
	advancedSettings, err := r.buildAdvancedSettings(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Advanced Settings",
			fmt.Sprintf("Could not build advanced settings: %s", err.Error()),
		)
		return
	}

	// Check if settings already exist
	advancedSettingsCheck, err := r.client.GetAdvancedSettings(resourceID, resourceType)
	if err != nil && !errors.Is(err, britive.ErrNotFound) && !errors.Is(err, britive.ErrNotSupported) {
		resp.Diagnostics.AddError(
			"Error Checking Advanced Settings",
			fmt.Sprintf("Could not check advanced settings: %s", err.Error()),
		)
		return
	}

	isUpdate := len(advancedSettingsCheck.Settings) != 0

	err = r.client.CreateUpdateAdvancedSettings(resourceID, resourceType, *advancedSettings, isUpdate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Advanced Settings",
			fmt.Sprintf("Could not create advanced settings: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateAdvancedSettingsID(resourceID, resourceType))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Advanced Settings",
			fmt.Sprintf("Could not read advanced settings after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AdvancedSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AdvancedSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID, resourceType, err := parseAdvancedSettingsID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Advanced Settings ID",
			fmt.Sprintf("Could not parse advanced settings ID: %s", err.Error()),
		)
		return
	}

	// Use resource_id from state if available (handles full paths)
	if !state.ResourceID.IsNull() {
		resourceID = state.ResourceID.ValueString()
	}

	advancedSettings, err := r.client.GetAdvancedSettings(resourceID, resourceType)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Advanced Settings",
			fmt.Sprintf("Could not read advanced settings: %s", err.Error()),
		)
		return
	}

	// Map API response to state
	if err := r.mapAPIToState(ctx, &state, advancedSettings); err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Advanced Settings",
			fmt.Sprintf("Could not map advanced settings: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AdvancedSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AdvancedSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := plan.ResourceID.ValueString()
	resourceType := strings.ToLower(plan.ResourceType.ValueString())

	// Build advanced settings from plan
	advancedSettings, err := r.buildAdvancedSettings(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Advanced Settings",
			fmt.Sprintf("Could not build advanced settings: %s", err.Error()),
		)
		return
	}

	err = r.client.CreateUpdateAdvancedSettings(resourceID, resourceType, *advancedSettings, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Advanced Settings",
			fmt.Sprintf("Could not update advanced settings: %s", err.Error()),
		)
		return
	}

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Advanced Settings",
			fmt.Sprintf("Could not read advanced settings after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AdvancedSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AdvancedSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := state.ResourceID.ValueString()
	resourceType := strings.ToLower(state.ResourceType.ValueString())

	// Send empty settings to reset
	advancedSettings := britive.AdvancedSettings{}

	err := r.client.CreateUpdateAdvancedSettings(resourceID, resourceType, advancedSettings, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Advanced Settings",
			fmt.Sprintf("Could not delete advanced settings: %s", err.Error()),
		)
		return
	}
}

func (r *AdvancedSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: {resource_id}/{resource_type}
	// Example: app123/application or paps/profile123/profile
	importArr := strings.Split(req.ID, "/")
	if len(importArr) < 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected format: '{resource_id}/{resource_type}'", req.ID),
		)
		return
	}

	resourceType := importArr[len(importArr)-1]
	resourceType = strings.ToLower(resourceType)

	validTypes := []string{"application", "profile", "profile_policy", "resource_manager_profile", "resource_manager_profile_policy"}
	isValid := false
	for _, vt := range validTypes {
		if strings.EqualFold(resourceType, vt) {
			isValid = true
			break
		}
	}

	if !isValid {
		resp.Diagnostics.AddError(
			"Invalid Resource Type",
			fmt.Sprintf("Resource type %q is not supported. Must be one of: application, profile, profile_policy, resource_manager_profile, resource_manager_profile_policy", resourceType),
		)
		return
	}

	importArr = importArr[:len(importArr)-1]
	resourceID := strings.Join(importArr, "/")

	advancedSettings, err := r.client.GetAdvancedSettings(resourceID, resourceType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Advanced Settings",
			fmt.Sprintf("Could not import advanced settings: %s", err.Error()),
		)
		return
	}

	// Extract actual resource ID for ID generation
	actualResourceID := resourceID
	if len(importArr) > 1 {
		actualResourceID = importArr[len(importArr)-1]
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateAdvancedSettingsID(actualResourceID, resourceType))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), resourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_type"), resourceType)...)

	// Create a temporary state model to map the advanced settings
	var state AdvancedSettingsResourceModel
	state.ID = types.StringValue(generateAdvancedSettingsID(actualResourceID, resourceType))
	state.ResourceID = types.StringValue(resourceID)
	state.ResourceType = types.StringValue(resourceType)

	if err := r.mapAPIToState(ctx, &state, advancedSettings); err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Advanced Settings",
			fmt.Sprintf("Could not map advanced settings during import: %s", err.Error()),
		)
		return
	}

	// Set the full state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildAdvancedSettings converts plan model to API model
func (r *AdvancedSettingsResource) buildAdvancedSettings(ctx context.Context, plan *AdvancedSettingsResourceModel) (*britive.AdvancedSettings, error) {
	advancedSettings := &britive.AdvancedSettings{}
	isInherited := false

	resourceID := plan.ResourceID.ValueString()
	resourceType := plan.ResourceType.ValueString()

	// Extract entity ID from resource ID
	resourceTypeArr := strings.Split(resourceType, "_")
	entityType := resourceTypeArr[len(resourceTypeArr)-1]
	entityType = strings.ToUpper(entityType)

	resourceIDArr := strings.Split(resourceID, "/")
	entityID := resourceID
	if len(resourceIDArr) > 1 {
		entityID = resourceIDArr[len(resourceIDArr)-1]
	}

	// Handle justification settings
	if plan.JustificationSettings != nil {
		justificationSetting := britive.Setting{
			SettingsType:            "JUSTIFICATION",
			EntityID:                entityID,
			EntityType:              entityType,
			IsInherited:             &isInherited,
			ID:                      plan.JustificationSettings.JustificationID.ValueString(),
			JustificationRegex:      plan.JustificationSettings.JustificationRegex.ValueString(),
		}
		isRequired := plan.JustificationSettings.IsJustificationRequired.ValueBool()
		justificationSetting.IsJustificationRequired = &isRequired

		advancedSettings.Settings = append(advancedSettings.Settings, justificationSetting)
	}

	// Handle ITSM settings
	if plan.ITSM != nil {
		itsmSetting := britive.Setting{
			SettingsType:   "ITSM",
			EntityID:       entityID,
			EntityType:     entityType,
			IsInherited:    &isInherited,
			ID:             plan.ITSM.ITSMID.ValueString(),
			ConnectionID:   plan.ITSM.ConnectionID.ValueString(),
			ConnectionType: plan.ITSM.ConnectionType.ValueString(),
		}
		isEnabled := plan.ITSM.IsITSMEnabled.ValueBool()
		itsmSetting.IsITSMEnabled = &isEnabled

		// Convert filter criteria
		var filterCriteria []ITSMFilterCriteriaModel
		diags := plan.ITSM.ITSMFilterCriteria.ElementsAs(ctx, &filterCriteria, false)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting ITSM filter criteria")
		}

		for _, fc := range filterCriteria {
			var filterMap map[string]interface{}
			if err := json.Unmarshal([]byte(fc.Filter.ValueString()), &filterMap); err != nil {
				return nil, fmt.Errorf("invalid JSON in filter: %w", err)
			}

			itsmFilter := britive.ItsmFilterCriteria{
				SupportedTicketType: fc.SupportedTicketType.ValueString(),
				Filter:              filterMap,
			}
			itsmSetting.ItsmFilterCriterias = append(itsmSetting.ItsmFilterCriterias, itsmFilter)
		}

		advancedSettings.Settings = append(advancedSettings.Settings, itsmSetting)
	}

	// Handle IM settings
	if plan.IM != nil {
		imSetting := britive.Setting{
			SettingsType:   "IM",
			EntityID:       entityID,
			EntityType:     entityType,
			IsInherited:    &isInherited,
			ID:             plan.IM.IMID.ValueString(),
			ConnectionID:   plan.IM.ConnectionID.ValueString(),
			ConnectionType: plan.IM.ConnectionType.ValueString(),
		}
		isAutoApproval := plan.IM.IsAutoApprovalEnabled.ValueBool()
		imSetting.IsAutoApprovalEnabled = &isAutoApproval

		// Convert escalation policies
		var policies []string
		diags := plan.IM.EscalationPolicies.ElementsAs(ctx, &policies, false)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting escalation policies")
		}
		imSetting.EscalationPolicies = policies

		advancedSettings.Settings = append(advancedSettings.Settings, imSetting)
	}

	return advancedSettings, nil
}

// mapAPIToState maps API response to state model
func (r *AdvancedSettingsResource) mapAPIToState(ctx context.Context, state *AdvancedSettingsResourceModel, advancedSettings *britive.AdvancedSettings) error {
	// Reset nested attributes
	state.JustificationSettings = nil
	state.ITSM = nil
	state.IM = nil

	// Process each setting
	for _, setting := range advancedSettings.Settings {
		// Skip inherited settings
		if setting.IsInherited != nil && *setting.IsInherited {
			continue
		}

		switch strings.ToUpper(setting.SettingsType) {
		case "JUSTIFICATION":
			state.JustificationSettings = &JustificationSettingsModel{
				JustificationID:         types.StringValue(setting.ID),
				JustificationRegex:      types.StringValue(setting.JustificationRegex),
			}
			if setting.IsJustificationRequired != nil {
				state.JustificationSettings.IsJustificationRequired = types.BoolValue(*setting.IsJustificationRequired)
			}

		case "ITSM":
			itsmModel := &ITSMModel{
				ITSMID:         types.StringValue(setting.ID),
				ConnectionID:   types.StringValue(setting.ConnectionID),
				ConnectionType: types.StringValue(setting.ConnectionType),
			}
			if setting.IsITSMEnabled != nil {
				itsmModel.IsITSMEnabled = types.BoolValue(*setting.IsITSMEnabled)
			}

			// Convert filter criteria
			var filterCriteriaModels []ITSMFilterCriteriaModel
			for _, fc := range setting.ItsmFilterCriterias {
				filterJSON, err := json.Marshal(fc.Filter)
				if err != nil {
					return fmt.Errorf("error marshaling filter: %w", err)
				}

				filterCriteriaModels = append(filterCriteriaModels, ITSMFilterCriteriaModel{
					SupportedTicketType: types.StringValue(fc.SupportedTicketType),
					Filter:              types.StringValue(string(filterJSON)),
				})
			}

			// Convert to Set
			filterSet, diags := types.SetValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"supported_ticket_type": types.StringType,
					"filter":                types.StringType,
				},
			}, filterCriteriaModels)
			if diags.HasError() {
				return fmt.Errorf("error converting filter criteria to set")
			}
			itsmModel.ITSMFilterCriteria = filterSet

			// Preserve user's connection_type case if it matches
			if state.ITSM != nil {
				userConnType := state.ITSM.ConnectionType.ValueString()
				if strings.EqualFold(setting.ConnectionType, userConnType) {
					itsmModel.ConnectionType = types.StringValue(userConnType)
				}
			}

			state.ITSM = itsmModel

		case "IM":
			imModel := &IMModel{
				IMID:           types.StringValue(setting.ID),
				ConnectionID:   types.StringValue(setting.ConnectionID),
				ConnectionType: types.StringValue(setting.ConnectionType),
			}
			if setting.IsAutoApprovalEnabled != nil {
				imModel.IsAutoApprovalEnabled = types.BoolValue(*setting.IsAutoApprovalEnabled)
			}

			// Convert escalation policies
			policiesSet, diags := types.SetValueFrom(ctx, types.StringType, setting.EscalationPolicies)
			if diags.HasError() {
				return fmt.Errorf("error converting escalation policies to set")
			}
			imModel.EscalationPolicies = policiesSet

			// Preserve user's connection_type case if it matches
			if state.IM != nil {
				userConnType := state.IM.ConnectionType.ValueString()
				if strings.EqualFold(setting.ConnectionType, userConnType) {
					imModel.ConnectionType = types.StringValue(userConnType)
				}
			}

			state.IM = imModel
		}
	}

	return nil
}

// populateStateFromAPI fetches advanced settings data from API and populates the state model
func (r *AdvancedSettingsResource) populateStateFromAPI(ctx context.Context, state *AdvancedSettingsResourceModel) error {
	resourceID := state.ResourceID.ValueString()
	resourceType := strings.ToLower(state.ResourceType.ValueString())

	advancedSettings, err := r.client.GetAdvancedSettings(resourceID, resourceType)
	if err != nil {
		return err
	}

	return r.mapAPIToState(ctx, state, advancedSettings)
}

// Helper functions
func generateAdvancedSettingsID(resourceID, resourceType string) string {
	resourceArr := strings.Split(resourceID, "/")
	if len(resourceArr) > 1 {
		return resourceType + "/" + resourceArr[len(resourceArr)-1] + "/advanced-settings"
	}
	return resourceType + "/" + resourceID + "/advanced-settings"
}

func parseAdvancedSettingsID(id string) (resourceID, resourceType string, err error) {
	arr := strings.Split(id, "/")
	if len(arr) < 3 {
		err = fmt.Errorf("invalid advanced settings ID format: %s", id)
		return
	}
	resourceID = arr[1]
	resourceType = arr[0]
	return
}
