package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceAdvancedSettings{}
	_ resource.ResourceWithConfigure   = &ResourceAdvancedSettings{}
	_ resource.ResourceWithImportState = &ResourceAdvancedSettings{}
)

type ResourceAdvancedSettings struct {
	client *britive_client.Client
	helper *ResourceAdvancedSettingsHelper
}

type ResourceAdvancedSettingsHelper struct{}

func NewResourceAdvancedSettings() resource.Resource {
	return &ResourceAdvancedSettings{}
}

func NewResourceAdvancedSettingsHelper() *ResourceAdvancedSettingsHelper {
	return &ResourceAdvancedSettingsHelper{}
}

func (ras *ResourceAdvancedSettings) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_advanced_settings"
}

func (ras *ResourceAdvancedSettings) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Advanced Setting resource")

	if req.ProviderData == nil {
		return
	}

	ras.client = req.ProviderData.(*britive_client.Client)
	if ras.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for ResourceAdvancedSettings")
	ras.helper = NewResourceAdvancedSettingsHelper()
}

func (ras *ResourceAdvancedSettings) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for advanced settings resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Required:    true,
				Description: "Britive resource ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type": schema.StringAttribute{
				Required:    true,
				Description: "Britive resource type",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"resource_type",
						validate.CaseInsensitiveOneOf(
							"APPLICATION",
							"PROFILE",
							"PROFILE_POLICY",
							"RESOURCE_MANAGER_PROFILE",
							"RESOURCE_MANAGER_PROFILE_POLICY",
						),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"justification_settings": schema.SetNestedBlock{
				Description: "Resource justification settings",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"justification_id": schema.StringAttribute{
							Computed:    true,
							Description: "Justification setting ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"is_justification_required": schema.BoolAttribute{
							Required:    true,
							Description: "Required resource justification or not",
						},
						"justification_regex": schema.StringAttribute{
							Optional:    true,
							Description: "Resource justification setting regular expression",
						},
					},
				},
			},
			"itsm": schema.SetNestedBlock{
				Description: "Resource ITSM settings",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"itsm_id": schema.StringAttribute{
							Computed:    true,
							Description: "ITSM Setting ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"connection_id": schema.StringAttribute{
							Required:    true,
							Description: "ITSM Connection ID",
						},
						"connection_type": schema.StringAttribute{
							Required:    true,
							Description: "ITSM Connection Type",
						},
						"is_itsm_enabled": schema.BoolAttribute{
							Optional:    true,
							Description: "Whether ITSM integration is enabled",
						},
					},
					Blocks: map[string]schema.Block{
						"itsm_filter_criteria": schema.SetNestedBlock{
							Description: "ITSM settings filter criteria",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"supported_ticket_type": schema.StringAttribute{
										Required:    true,
										Description: "supported ticket type for ITSM filter criteria",
									},
									"filter": schema.StringAttribute{
										Required:    true,
										Description: "Filter for ITSM filter criteria",
										Validators: []validator.String{
											validate.StringFunc(
												"filter",
												validate.IsValidJSON(),
											),
										},
									},
								},
							},
						},
					},
				},
			},
			"im": schema.SetNestedBlock{
				Description: "Resource IM setting",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"im_id": schema.StringAttribute{
							Computed:    true,
							Description: "IM settings ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"connection_id": schema.StringAttribute{
							Required:    true,
							Description: "IM connection ID",
						},
						"connection_type": schema.StringAttribute{
							Required:    true,
							Description: "IM connection type",
						},
						"is_auto_approval_enabled": schema.BoolAttribute{
							Required:    true,
							Description: "IM auto approval toggle",
						},
						"escalation_policies": schema.SetAttribute{
							Required:    true,
							Description: "IM Escalation Policies",
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (ras *ResourceAdvancedSettings) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_advanced_settings")
	var plan britive_client.AdvancedSettingsPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during advanced settings creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	resourceId := plan.ResourceID.ValueString()
	resourceType := plan.ResourceType.ValueString()
	resourceType = strings.ToLower(resourceType)

	if resourceId == "" || plan.ResourceID.IsNull() || plan.ResourceID.IsUnknown() {
		resp.Diagnostics.AddError("Invalid resource_id", fmt.Sprintf("resource_id: %s", resourceId))
		tflog.Info(ctx, fmt.Sprintf("Invalid resource_id: %s", resourceId))
		return
	}
	if resourceType == "" {
		resp.Diagnostics.AddError("Invalid resource_type", fmt.Sprintf("resource_type: %s", resourceType))
		tflog.Info(ctx, fmt.Sprintf("Invalid resource_type: %s", resourceType))
		return
	}

	advancedSettings := britive_client.AdvancedSettings{}
	err := ras.helper.mapAdvancedSettingResourceToModel(plan, &advancedSettings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map adanced settings to resource", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map advanced settings to resource, %T", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Added new advanced settings %#v", advancedSettings))

	advancedSettingsCheck, err := ras.client.GetAdvancedSettings(ctx, resourceId, resourceType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch advanced settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch advanced settings, %T", err))
		return
	}

	isUpdate := false
	if len(advancedSettingsCheck.Settings) != 0 {
		isUpdate = true
	}

	err = ras.client.CreateUpdateAdvancedSettings(ctx, resourceId, resourceType, advancedSettings, isUpdate)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create advanced settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create advanced settings, %T", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new advanced settings: %#v", advancedSettings))

	settingId := ras.helper.generateUniqueID(resourceId, resourceType)
	plan.ID = types.StringValue(settingId)

	tflog.Info(ctx, fmt.Sprintf("Generated advanced settings id: %s", settingId))

	planPtr, err := ras.helper.getAndMapModelToPlan(ctx, plan, *ras.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get advanced settings",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map advanced settings model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after create", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Create completed and state set", map[string]interface{}{
		"advanced_settings": planPtr,
	})
}

func (ras *ResourceAdvancedSettings) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_advanced_settings")

	if ras.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.AdvancedSettingsPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get advanced settings state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := ras.helper.getAndMapModelToPlan(ctx, state, *ras.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get advanced settings",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map advanced settings model to plan failed in Read", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	diags = resp.State.Set(ctx, newPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state in Read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Read completed for britive_advanced_settings")
}

func (ras *ResourceAdvancedSettings) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_advanced_settings")

	var plan, state britive_client.AdvancedSettingsPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	advancedSettings := britive_client.AdvancedSettings{}

	tflog.Info(ctx, "Mapping resource to model for update")

	err := ras.helper.mapAdvancedSettingResourceToModel(plan, &advancedSettings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update advanced settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map advanced_settings resource to model: %#v", err))
		return
	}

	resourceId := plan.ResourceID.ValueString()
	resourceType := plan.ResourceType.ValueString()
	resourceType = strings.ToLower(resourceType)

	if resourceId == "" {
		resp.Diagnostics.AddError("Failed to update advanced_settings", "resource_id not found")
		tflog.Error(ctx, "resource_id not found : ''")
		return
	}
	if resourceType == "" {
		resp.Diagnostics.AddError("Failed to update advanced_settings", "resource_type not found")
		tflog.Error(ctx, "resource_type not found : ''")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Updating advanced settings: %#v", advancedSettings))

	err = ras.client.CreateUpdateAdvancedSettings(ctx, resourceId, resourceType, advancedSettings, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update advanced_settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to update advanced settings: %#v", err))
		return
	}

	planPtr, err := ras.helper.getAndMapModelToPlan(ctx, plan, *ras.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get advanced_settings",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map advanced_settings model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after update", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set")
}

func (ras *ResourceAdvancedSettings) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_advanced_settings")

	var state britive_client.AdvancedSettingsPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	advancedSettings := britive_client.AdvancedSettings{}

	resourceId := state.ResourceID.ValueString()
	resourceType := state.ResourceType.ValueString()
	resourceType = strings.ToLower(resourceType)

	if resourceId == "" {
		resp.Diagnostics.AddError("Failed to delete advanced_settings", "resource_id not found")
		tflog.Error(ctx, "resource_id not found : ''")
		return
	}
	if resourceType == "" {
		resp.Diagnostics.AddError("Failed to delete advanced_settings", "resource_type not found")
		tflog.Error(ctx, "resource_type not found : ''")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting advanced settings of %s", resourceId))

	err := ras.client.CreateUpdateAdvancedSettings(ctx, resourceId, resourceType, advancedSettings, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete advanced_settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete advanced_settings: %#v", err))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "advanced_settings deleted successfully")
}

func (ras *ResourceAdvancedSettings) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	importID := importData.ID

	log.Printf("==== %s", importID)

	importArr := strings.Split(importID, "/")
	resourceType := importArr[len(importArr)-1]
	resourceType = strings.ToLower(resourceType)

	log.Printf("==== resTypes %s", resourceType)

	if !strings.EqualFold("application", resourceType) && !strings.EqualFold("profile", resourceType) && !strings.EqualFold("profile_policy", resourceType) && !strings.EqualFold("resource_manager_profile", resourceType) && !strings.EqualFold("resource_manager_profile_policy", resourceType) {
		resp.Diagnostics.AddError("Invalid import ID", errs.NewNotSupportedError(resourceType).Error())
		tflog.Error(ctx, fmt.Sprintf("Invalid import ID: %s", resourceType))
		return
	}

	importArr = importArr[:len(importArr)-1]
	resourceID := strings.Join(importArr, "/")

	tflog.Info(ctx, fmt.Sprintf("Importing advanced settings, %s", resourceID))

	if len(importArr) > 1 {
		resourceID = importArr[len(importArr)-1]
	} else {
		resourceID = importArr[0]
	}

	plan := britive_client.AdvancedSettingsPlan{
		ID:           types.StringValue(ras.helper.generateUniqueID(resourceID, resourceType)),
		ResourceID:   types.StringValue(resourceID),
		ResourceType: types.StringValue(resourceType),
	}

	planPtr, err := ras.helper.getAndMapModelToPlan(ctx, plan, *ras.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch advanced settings", fmt.Sprintf("Error: %v", err))
		tflog.Error(ctx, "Failed to build state from API during import", map[string]interface{}{"error": err.Error()})
		return
	}

	diags := resp.State.Set(ctx, planPtr)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state in Import", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Imported advanced settings")
}

func (rash *ResourceAdvancedSettingsHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.AdvancedSettingsPlan, c britive_client.Client) (*britive_client.AdvancedSettingsPlan, error) {
	resourceID, resourceType := rash.parseUniqueID(plan.ID.ValueString())
	tflog.Info(ctx, fmt.Sprintf("Reading advanced settings %s/%s", resourceID, resourceType))

	resourceID = plan.ResourceID.ValueString()

	advancedSettings, err := c.GetAdvancedSettings(ctx, resourceID, resourceType)
	if err != nil {
		return nil, err
	}

	log.Printf("==== fetched settings %#v", advancedSettings)

	var rawJustificationSetting britive_client.Setting
	var rawItsmSetting britive_client.Setting
	var rawImSetting britive_client.Setting

	for _, rawSetting := range advancedSettings.Settings {
		if strings.EqualFold(rawSetting.SettingsType, "JUSTIFICATION") {
			rawJustificationSetting = rawSetting
		} else if strings.EqualFold(rawSetting.SettingsType, "ITSM") {
			rawItsmSetting = rawSetting
		} else if strings.EqualFold(rawSetting.SettingsType, "IM") {
			rawImSetting = rawSetting
		}
	}

	// Mapping Justification Settings
	if rawJustificationSetting.ID != "" && !(rawJustificationSetting.IsInherited != nil && *rawJustificationSetting.IsInherited) {
		var justificationPlan []britive_client.JustificationSettingsPlan
		justificationPlan = append(justificationPlan, britive_client.JustificationSettingsPlan{
			JustificationID:         types.StringValue(rawJustificationSetting.ID),
			JustificationRegex:      types.StringValue(rawJustificationSetting.JustificationRegex),
			IsJustificationRequired: types.BoolValue(*rawJustificationSetting.IsJustificationRequired),
		})
		plan.JustificationSettings, err = rash.mapJustificationPlanToSet(justificationPlan)
		if err != nil {
			return nil, err
		}
	} else {
		var justificationSettingsObjType = types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"justification_id":          types.StringType,
				"justification_regex":       types.StringType,
				"is_justification_required": types.BoolType,
			},
		}
		emptySet := types.SetNull(justificationSettingsObjType)
		plan.JustificationSettings = emptySet
	}

	// Mapping ITSM setting
	if rawItsmSetting.ID != "" && !(rawItsmSetting.IsInherited != nil && *rawItsmSetting.IsInherited) {
		var itsmFilterCriteriaPlan []britive_client.ItsmFilterCriteriaPlan
		for _, criteria := range rawItsmSetting.ItsmFilterCriterias {
			filterStr := ""
			if criteria.Filter != nil {
				bytes, err := json.Marshal(criteria.Filter)
				if err != nil {
					return nil, err
				}
				filterStr = string(bytes)
			}
			itsmFilterCriteriaPlan = append(itsmFilterCriteriaPlan, britive_client.ItsmFilterCriteriaPlan{
				SupportedTicketType: types.StringValue(criteria.SupportedTicketType),
				Filter:              types.StringValue(filterStr),
			})
		}

		var userConnType, connType string
		itsmStatePlan, _, err := rash.mapSetToItsmPlan(plan.Itsm)
		if err != nil {
			return nil, err
		}
		if len(itsmStatePlan) == 1 {
			itsm := itsmStatePlan[0]
			userConnType = itsm.ConnectionType.ValueString()
		}
		if strings.EqualFold(rawItsmSetting.ConnectionType, userConnType) {
			connType = userConnType
		} else {
			connType = rawItsmSetting.ConnectionType
		}

		var itsmPlan []britive_client.ItsmPlan
		itsmPlan = append(itsmPlan, britive_client.ItsmPlan{
			ConnectionID:   types.StringValue(rawItsmSetting.ConnectionID),
			ConnectionType: types.StringValue(connType),
			IsItsmEnabled:  types.BoolPointerValue(rawItsmSetting.IsITSMEnabled),
			ItsmID:         types.StringValue(rawItsmSetting.ID),
		})

		plan.Itsm, err = rash.mapItsmPlanToSet(itsmPlan, itsmFilterCriteriaPlan)
		if err != nil {
			return nil, err
		}

	} else {
		var itsmFilterCriteriaObjType = types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"supported_ticket_type": types.StringType,
				"filter":                types.StringType,
			},
		}

		var itsmObjType = types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"itsm_id":              types.StringType,
				"connection_id":        types.StringType,
				"connection_type":      types.StringType,
				"is_itsm_enabled":      types.BoolType,
				"itsm_filter_criteria": types.SetType{ElemType: itsmFilterCriteriaObjType},
			},
		}
		emptySet := types.SetNull(itsmObjType)
		plan.Itsm = emptySet
	}

	// Mapping IM Settings
	if rawImSetting.ID != "" && !(rawImSetting.IsInherited != nil && *rawImSetting.IsInherited) {
		var userConnType, connType string
		imStatePlan, _, err := rash.mapSetToImPlan(plan.Im)
		if err != nil {
			return nil, err
		}
		if len(imStatePlan) == 1 {
			im := imStatePlan[0]
			userConnType = im.ConnectionType.ValueString()
		}
		if strings.EqualFold(rawImSetting.ConnectionType, userConnType) {
			connType = userConnType
		} else {
			connType = rawImSetting.ConnectionType
		}
		var imPlan []britive_client.ImPlan
		imPlan = append(imPlan, britive_client.ImPlan{
			ConnectionID:          types.StringValue(rawImSetting.ConnectionID),
			ConnectionType:        types.StringValue(connType),
			IsAutoApprovalEnabled: types.BoolPointerValue(rawImSetting.IsAutoApprovalEnabled),
			ImID:                  types.StringValue(rawImSetting.ID),
		})
		plan.Im, err = rash.mapImPlanToSet(imPlan, rawImSetting.EscalationPolicies)
		if err != nil {
			return nil, err
		}
	} else {
		var imObjType = types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"im_id":                    types.StringType,
				"connection_id":            types.StringType,
				"connection_type":          types.StringType,
				"is_auto_approval_enabled": types.BoolType,
				"escalation_policies":      types.SetType{ElemType: types.StringType},
			},
		}
		emptySet := types.SetNull(imObjType)
		plan.Im = emptySet
	}

	tflog.Info(ctx, fmt.Sprintf("Updated state of advanced settings %#v", plan))

	return &plan, nil
}

func (rash *ResourceAdvancedSettingsHelper) generateUniqueID(resourceID, resourceType string) string {
	resourceArr := strings.Split(resourceID, "/")
	if len(resourceArr) > 1 {
		return resourceType + "/" + resourceArr[len(resourceArr)-1] + "/advanced-settings"
	}
	generatedID := resourceType + "/" + resourceID + "/advanced-settings"

	return generatedID
}

func (rash *ResourceAdvancedSettingsHelper) parseUniqueID(ID string) (string, string) {
	arr := strings.Split(ID, "/")
	resourceId := arr[1]
	resourceType := arr[0]
	return resourceId, resourceType
}

func (rash *ResourceAdvancedSettingsHelper) mapAdvancedSettingResourceToModel(plan britive_client.AdvancedSettingsPlan, advancedSettings *britive_client.AdvancedSettings) error {
	isInherited := false
	resourceId := plan.ResourceID.ValueString()
	resourceType := plan.ResourceType.ValueString()
	resourceTypeArr := strings.Split(resourceType, "_")
	resourceType = resourceTypeArr[len(resourceTypeArr)-1]
	resourceType = strings.ToUpper(resourceType)

	resourceIDArr := strings.Split(resourceId, "/")
	if len(resourceIDArr) > 1 {
		resourceId = resourceIDArr[len(resourceIDArr)-1]
	}

	// Handle justification settings
	justificationList, err := rash.mapSetToJustificationPlan(plan.JustificationSettings)
	if err != nil {
		return err
	}

	log.Printf("==== justificationList:%#v", justificationList)

	if len(justificationList) > 1 {
		return fmt.Errorf("must contain exactly one justification setting")
	}
	if len(justificationList) == 1 {
		userJustificationSetting := justificationList[0]
		justificationSetting := britive_client.Setting{
			SettingsType:            "JUSTIFICATION",
			EntityID:                resourceId,
			EntityType:              resourceType,
			IsInherited:             &isInherited,
			ID:                      userJustificationSetting.JustificationID.ValueString(),
			IsJustificationRequired: userJustificationSetting.IsJustificationRequired.ValueBoolPointer(),
			JustificationRegex:      userJustificationSetting.JustificationRegex.ValueString(),
		}

		advancedSettings.Settings = append(advancedSettings.Settings, justificationSetting)
	}

	// Handle ITSM settings
	itsmList, itsmFilterList, err := rash.mapSetToItsmPlan(plan.Itsm)
	if err != nil {
		return err
	}
	if len(itsmList) > 1 {
		return fmt.Errorf("must contain exactly one ITSM setting")
	}
	if len(itsmList) == 1 {
		userItsmSetting := itsmList[0]
		itsmSetting := britive_client.Setting{
			SettingsType:   "ITSM",
			EntityID:       resourceId,
			EntityType:     resourceType,
			IsInherited:    &isInherited,
			ID:             userItsmSetting.ItsmID.ValueString(),
			ConnectionID:   userItsmSetting.ConnectionID.ValueString(),
			ConnectionType: userItsmSetting.ConnectionType.ValueString(),
			IsITSMEnabled:  userItsmSetting.IsItsmEnabled.ValueBoolPointer(),
		}

		for _, item := range itsmFilterList {
			var js map[string]interface{}
			if err := json.Unmarshal([]byte(item.Filter.ValueString()), &js); err != nil {
				return err
			}
			itsmFilter := britive_client.ItsmFilterCriteria{
				SupportedTicketType: item.SupportedTicketType.ValueString(),
				Filter:              make(map[string]interface{}),
			}

			for k, v := range js {
				itsmFilter.Filter[k] = v
			}
			itsmSetting.ItsmFilterCriterias = append(itsmSetting.ItsmFilterCriterias, itsmFilter)
		}
		advancedSettings.Settings = append(advancedSettings.Settings, itsmSetting)
	}

	// Handle IM settings
	imList, escPolicies, err := rash.mapSetToImPlan(plan.Im)
	if err != nil {
		return err
	}
	if len(imList) > 1 {
		return fmt.Errorf("must contain exactly one IM setting")
	}
	if len(imList) == 1 {
		userImSetting := imList[0]
		imSetting := britive_client.Setting{
			SettingsType:          "IM",
			EntityID:              resourceId,
			EntityType:            resourceType,
			IsInherited:           &isInherited,
			ID:                    userImSetting.ImID.ValueString(),
			ConnectionID:          userImSetting.ConnectionID.ValueString(),
			ConnectionType:        userImSetting.ConnectionType.ValueString(),
			IsAutoApprovalEnabled: userImSetting.IsAutoApprovalEnabled.ValueBoolPointer(),
			EscalationPolicies:    escPolicies,
		}
		advancedSettings.Settings = append(advancedSettings.Settings, imSetting)
	}

	return nil
}

func (rash *ResourceAdvancedSettingsHelper) mapSetToJustificationPlan(set types.Set) ([]britive_client.JustificationSettingsPlan, error) {
	var result []britive_client.JustificationSettingsPlan
	if set.IsNull() || set.IsUnknown() {
		return result, nil
	}
	objs := set.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected Object, got %T", e)
		}
		var p britive_client.JustificationSettingsPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert object to JustificationSettingsPlan: %v", diags)
		}
		result = append(result, p)
	}
	return result, nil
}

func (rash *ResourceAdvancedSettingsHelper) mapJustificationPlanToSet(plans []britive_client.JustificationSettingsPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"justification_id":          types.StringType,
				"is_justification_required": types.BoolType,
				"justification_regex":       types.StringType,
			},
			map[string]attr.Value{
				"justification_id":          p.JustificationID,
				"is_justification_required": p.IsJustificationRequired,
				"justification_regex":       p.JustificationRegex,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for justification settings: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"justification_id":          types.StringType,
				"is_justification_required": types.BoolType,
				"justification_regex":       types.StringType,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create justification settings set: %v", diags)
	}

	return set, nil
}

func (rash *ResourceAdvancedSettingsHelper) mapSetToImPlan(set types.Set) ([]britive_client.ImPlan, []string, error) {
	var result []britive_client.ImPlan
	var policies []string

	if set.IsNull() || set.IsUnknown() {
		return result, policies, nil
	}

	objs := set.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, nil, fmt.Errorf("expected Object for im element, got %T", e)
		}

		var p britive_client.ImPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, nil, fmt.Errorf("failed to convert object to ImPlan: %v", diags)
		}

		var thisPolicies []string
		if !p.EscalationPolicies.IsNull() {
			diags := p.EscalationPolicies.ElementsAs(context.Background(), &thisPolicies, false)
			if diags.HasError() {
				return nil, nil, fmt.Errorf("failed to read escalation_policies: %v", diags)
			}
		}

		policies = thisPolicies

		result = append(result, p)
	}
	return result, policies, nil
}

func (rash *ResourceAdvancedSettingsHelper) mapImPlanToSet(plans []britive_client.ImPlan, escPolicies []string) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		policyObjs := make([]attr.Value, 0, len(escPolicies))
		for _, pol := range escPolicies {
			policyObjs = append(policyObjs, types.StringValue(pol))
		}
		policySet, diags := types.SetValue(types.StringType, policyObjs)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create escalation_policies set: %v", diags)
		}

		// create object for im nested block
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"im_id":                    types.StringType,
				"connection_id":            types.StringType,
				"connection_type":          types.StringType,
				"is_auto_approval_enabled": types.BoolType,
				"escalation_policies":      types.SetType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"im_id":                    p.ImID,
				"connection_id":            p.ConnectionID,
				"connection_type":          p.ConnectionType,
				"is_auto_approval_enabled": p.IsAutoApprovalEnabled,
				"escalation_policies":      policySet,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for im: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"im_id":                    types.StringType,
				"connection_id":            types.StringType,
				"connection_type":          types.StringType,
				"is_auto_approval_enabled": types.BoolType,
				"escalation_policies":      types.SetType{ElemType: types.StringType},
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create im set: %v", diags)
	}

	return set, nil
}

func (rash *ResourceAdvancedSettingsHelper) mapSetToItsmPlan(set types.Set) ([]britive_client.ItsmPlan, []britive_client.ItsmFilterCriteriaPlan, error) {
	var result []britive_client.ItsmPlan
	var criteria []britive_client.ItsmFilterCriteriaPlan

	if set.IsNull() || set.IsUnknown() {
		return result, criteria, nil
	}

	objs := set.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, nil, fmt.Errorf("expected Object for itsm element, got %T", e)
		}

		var p britive_client.ItsmPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, nil, fmt.Errorf("failed to convert object to ItsmPlan: %v", diags)
		}

		if !p.ItsmFilterCriteria.IsNull() {
			elems := p.ItsmFilterCriteria.Elements()

			for _, ee := range elems {
				o, ok := ee.(types.Object)
				if !ok {
					return nil, nil, fmt.Errorf(
						"expected Object inside itsm_filter_criteria, got %T", ee,
					)
				}

				var c britive_client.ItsmFilterCriteriaPlan
				var ctx context.Context
				diags := o.As(ctx, &c, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, nil, fmt.Errorf(
						"failed to convert itsm_filter_criteria element: %v", diags,
					)
				}

				criteria = append(criteria, c)
			}
		}

		if len(criteria) == 0 {
			elems := p.ItsmFilterCriteria.Elements()
			for _, ee := range elems {
				o, ok := ee.(types.Object)
				if !ok {
					return nil, nil, fmt.Errorf("expected Object inside itsm_filter_criteria, got %T", ee)
				}
				var c britive_client.ItsmFilterCriteriaPlan
				diags := o.As(context.Background(), &c, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, nil, fmt.Errorf("failed to convert itsm_filter_criteria element: %v", diags)
				}
				criteria = append(criteria, c)
			}
		}

		api := britive_client.ItsmPlan{
			ItsmID:         p.ItsmID,
			ConnectionID:   p.ConnectionID,
			ConnectionType: p.ConnectionType,
			IsItsmEnabled:  p.IsItsmEnabled,
		}

		result = append(result, api)

	}
	return result, criteria, nil
}

func (rash *ResourceAdvancedSettingsHelper) mapItsmPlanToSet(plans []britive_client.ItsmPlan, itsmFilterCriterias []britive_client.ItsmFilterCriteriaPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		critObjs := make([]attr.Value, 0, len(itsmFilterCriterias))
		for _, c := range itsmFilterCriterias {
			co, diags := types.ObjectValue(
				map[string]attr.Type{
					"supported_ticket_type": types.StringType,
					"filter":                types.StringType,
				},
				map[string]attr.Value{
					"supported_ticket_type": c.SupportedTicketType,
					"filter":                c.Filter,
				},
			)
			if diags.HasError() {
				return types.Set{}, fmt.Errorf("failed to create itsm_filter_criteria object: %v", diags)
			}
			critObjs = append(critObjs, co)
		}
		critSet, diags := types.SetValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"supported_ticket_type": types.StringType,
					"filter":                types.StringType,
				},
			},
			critObjs,
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create itsm_filter_criteria set: %v", diags)
		}

		// create outer itsm object
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"itsm_id":         types.StringType,
				"connection_id":   types.StringType,
				"connection_type": types.StringType,
				"is_itsm_enabled": types.BoolType,
				"itsm_filter_criteria": types.SetType{ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"supported_ticket_type": types.StringType,
						"filter":                types.StringType,
					},
				}},
			},
			map[string]attr.Value{
				"itsm_id":              p.ItsmID,
				"connection_id":        p.ConnectionID,
				"connection_type":      p.ConnectionType,
				"is_itsm_enabled":      p.IsItsmEnabled,
				"itsm_filter_criteria": critSet,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create itsm object: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"itsm_id":         types.StringType,
				"connection_id":   types.StringType,
				"connection_type": types.StringType,
				"is_itsm_enabled": types.BoolType,
				"itsm_filter_criteria": types.SetType{ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"supported_ticket_type": types.StringType,
						"filter":                types.StringType,
					},
				}},
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create itsm set: %v", diags)
	}
	return set, nil
}
