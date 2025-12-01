package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/argon2"
)

var (
	_ resource.Resource              = &ResourceApplication{}
	_ resource.ResourceWithConfigure = &ResourceApplication{}
	// _ resource.ResourceWithImportState = &ResourceApplication{}
)

type ResourceApplication struct {
	client *britive_client.Client
	helper *ResourceApplicationHelper
	// importHelper *imports.ImportHelper
}

type ResourceApplicationHelper struct{}

func NewResourceApplication() resource.Resource {
	return &ResourceApplication{}
}

func NewResourceApplicationHelper() *ResourceApplicationHelper {
	return &ResourceApplicationHelper{}
}

func (ra *ResourceApplication) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_application"
}

func (ra *ResourceApplication) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Application resource")

	if req.ProviderData == nil {
		return
	}

	ra.client = req.ProviderData.(*britive_client.Client)
	if ra.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for ResourceApplication")
	ra.helper = NewResourceApplicationHelper()
}

func (ra *ResourceApplication) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Application resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
			},
			"application_type": schema.StringAttribute{
				Required:    true,
				Description: "Britive application type. Supported types: 'Snowflake', 'Snowflake Standalone', 'GCP', 'GCP Standalone', 'Google Workspace', 'AWS', 'AWS Standalone', 'Azure', 'Okta'.",
				Validators: []validator.String{
					validate.CaseInsensitiveOneOfValidator(
						"Snowflake",
						"Snowflake Standalone",
						"GCP",
						"GCP Standalone",
						"Google Workspace",
						"AWS",
						"AWS Standalone",
						"Azure",
						"Okta",
					),
				},
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Britive application version",
			},
			"catalog_app_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Britive application base catalog ID",
			},
			"entity_root_environment_group_id": schema.StringAttribute{
				Computed:    true,
				Description: "Britive application root environment ID for Snowflake Standalone applications",
			},
		},
		Blocks: map[string]schema.Block{
			"properties": schema.SetNestedBlock{
				Description: "Britive application override properties",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Property name",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "Property value",
						},
					},
				},
			},

			"sensitive_properties": schema.SetNestedBlock{
				Description: "Britive application override sensitive properties",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Property name",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "Property value",
						},
					},
				},
			},

			"user_account_mappings": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				Description: "Application user account",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Application user account name",
						},
						"description": schema.StringAttribute{
							Required:    true,
							Description: "Application user account description",
						},
					},
				},
			},
		},
	}
}

func (ra *ResourceApplication) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_application")

	var plan britive_client.ApplicationPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during application creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	appCatalogDetails, err := ra.helper.validatePropertiesAgainstSystemApps(ctx, plan, ra.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to validate application",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Failed to validate application", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	applicationName, err := ra.helper.getApplicationName(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Missing `displayName` property",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Missing `displayName` property", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	application := britive_client.ApplicationRequest{}
	ra.helper.mapApplicationResourceToModel(&application, applicationName, appCatalogDetails)

	tflog.Info(ctx, "Creating new application")

	appResponse, err := ra.client.CreateApplication(ctx, application)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create application",
			fmt.Sprintf("CreateApplication API call failed: %v", err),
		)
		tflog.Error(ctx, "CreateApplication API call failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	tflog.Info(ctx, "Application creation success")

	plan.ID = types.StringValue(ra.helper.generateUniqueID(appResponse.AppContainerId))

	// Patch properties
	properties := britive_client.Properties{}
	err = ra.helper.mapPropertiesResourceToModel(plan, &properties, appResponse)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unabe to map properties",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Unabe to map properties", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	tflog.Info(ctx, "Updating application properties")

	_, err = ra.client.PatchApplicationPropertyTypes(ctx, appResponse.AppContainerId, properties)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unabe to update properties",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Unabe to update properties", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	tflog.Info(ctx, "Updated application properties")

	// configer user account mappings
	userMappings := britive_client.UserMappings{}
	err = ra.helper.mapUserMappingsResourceToModel(plan, &userMappings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unabe to map user_mappings",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Unabe to map user_mappings", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	tflog.Info(ctx, "Updating user mapping", map[string]interface{}{
		"userMapping": userMappings,
	})
	err = ra.client.ConfigureUserMappings(ctx, appResponse.AppContainerId, userMappings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unabe to configure user_mappings",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Unabe to configure user_mappings", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	tflog.Info(ctx, "User Mappings configured successfully")

	//The root environment group creation can be skipped when PAB-20648 is fixed
	allowedEnvGroupApps := map[int64]string{
		2: "AWS Standalone",
		8: "Okta",
		9: "Snowflake Standalone",
	}
	if _, ok := allowedEnvGroupApps[application.CatalogAppId]; ok {
		tflog.Info(ctx, "Creating root environment group")
		err = ra.client.CreateRootEnvironmentGroup(ctx, appResponse.AppContainerId, application.CatalogAppId)
		if err != nil {

		}
		tflog.Info(ctx, "Created root environment group")
		rootEnvID, err := ra.client.GetRootEnvID(ctx, appResponse.AppContainerId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unabe to get root environment id",
				fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
			)
			tflog.Error(ctx, "Unabe to get root environment id", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		plan.EntityRootEnvironmentGroupID = types.StringValue(rootEnvID)
	}

	planPtr, err := ra.helper.getAndMapModelToPlan(ctx, plan, *ra.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get application",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map application model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	log.Printf("===== out of getandmap:%v", planPtr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after create", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Create completed and state set", map[string]interface{}{
		"application": planPtr,
	})
}

func (ra *ResourceApplication) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_application")

	if ra.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ApplicationPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get application state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	applicationID := state.ID.ValueString()
	if applicationID == "" {
		resp.Diagnostics.AddError(
			"Failed to get application",
			"application Id not found",
		)
		tflog.Error(ctx, "Read failed: missing application ID in state")
		return
	}

	newPlan, err := ra.helper.getAndMapModelToPlan(ctx, state, *ra.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get application",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map application model to plan failed in Read", map[string]interface{}{
			"error":          err.Error(),
			"application_id": applicationID,
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

	tflog.Info(ctx, "Read completed for britive_application", map[string]interface{}{
		"application_id": applicationID,
	})
}

func (ra *ResourceApplication) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_application")

	var plan, state britive_client.ApplicationPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	foundApp, err := ra.helper.validatePropertiesAgainstSystemApps(ctx, plan, ra.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to validate application",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Failed to validate application", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	applicationID, err := ra.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"ApplicationID if invalid",
			fmt.Sprintf("Error: %v.", err),
		)
		tflog.Error(ctx, "Failed to parse applicationID", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	var hasChanges bool
	if !plan.Properties.Equal(state.Properties) || plan.SensitiveProperties.Equal(state.SensitiveProperties) {
		hasChanges = true

		tflog.Info(ctx, "Reading application", map[string]interface{}{
			"applicationID": applicationID,
		})
		application, err := ra.client.GetApplication(ctx, applicationID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to fetch applicationID",
				fmt.Sprintf("Error: %v.", err),
			)
			tflog.Error(ctx, "Failed to fetch applicationID", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		tflog.Info(ctx, "Received application", map[string]interface{}{
			"application": application,
		})

		properties := britive_client.Properties{}

		oldProps, err := ra.helper.mapSetToPropertyPlan(state.Properties)
		if err != nil {
			tflog.Error(ctx, "Unable to map properties set to plan")
		}
		newProps, err := ra.helper.mapSetToPropertyPlan(plan.Properties)
		if err != nil {
			tflog.Error(ctx, "Unable to map properties set to plan")
		}

		oldSprops, err := ra.helper.mapSetToSensitivePropertyPlan(state.SensitiveProperties)
		if err != nil {
			tflog.Error(ctx, "Unable to map sensitive properties set to plan")
		}
		newSprops, err := ra.helper.mapSetToSensitivePropertyPlan(plan.SensitiveProperties)
		if err != nil {
			tflog.Error(ctx, "Unable to map sensitive properties set to plan")
		}

		getRemovedProperties(foundApp, &properties, oldProps, newProps, oldSprops, newSprops)
		err = ra.helper.mapPropertiesResourceToModel(plan, &properties, application)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unabe to map properties",
				fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
			)
			tflog.Error(ctx, "Unabe to map properties", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		tflog.Info(ctx, "Updating application properties")
		_, err = ra.client.PatchApplicationPropertyTypes(ctx, applicationID, properties)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unabe to update properties",
				fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
			)
			tflog.Error(ctx, "Unabe to update properties", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		tflog.Info(ctx, "Updated application properties")
	}
	if !plan.UserAccountMappings.Equal(state.UserAccountMappings) {
		hasChanges = true
		userMappings := britive_client.UserMappings{}
		err = ra.helper.mapUserMappingsResourceToModel(plan, &userMappings)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to map user-account-mappings",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Unable to map user-account-mappings", map[string]interface{}{
				"Error": err,
			})
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Updating user mappings: %#v", userMappings))
		err = ra.client.ConfigureUserMappings(ctx, applicationID, userMappings)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to update user_account_mappings",
				fmt.Sprintf("Error:%v", err),
			)
			tflog.Error(ctx, fmt.Sprintf("Failed to update user_account_mappings, Error:%v", err))
			return
		}
		tflog.Info(ctx, fmt.Sprintf("Updated user mappings: %#v", userMappings))
	}

	plan.ID = types.StringValue(ra.helper.generateUniqueID(applicationID))

	if hasChanges {
		planPtr, err := ra.helper.getAndMapModelToPlan(ctx, plan, *ra.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get application",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map application model to plan", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &planPtr)...)
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "Failed to set state after update", map[string]interface{}{
				"diagnostics": resp.Diagnostics,
			})
			return
		}
		tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
			"application": planPtr,
		})
	}

}

func (ra *ResourceApplication) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_application")

	var state britive_client.ApplicationPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	applicationID, err := ra.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch applicationID",
			fmt.Sprintf("applicationID: %s", applicationID),
		)
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch applicationID: %s", applicationID))
		return
	}

	tflog.Info(ctx, "Deleting application", map[string]interface{}{
		"applicationID": applicationID,
	})

	err = ra.client.DeleteApplication(ctx, applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting application",
			"Reason: "+err.Error(),
		)
		tflog.Error(ctx, "Delete Application API call failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	tflog.Info(ctx, "Delete Application API call succeeded", map[string]interface{}{
		"applicationID": applicationID,
	})
	resp.State.RemoveResource(ctx)
}

func getRemovedProperties(application *britive_client.SystemApp, properties *britive_client.Properties, oldProps, newProps []britive_client.PropertyPlan, oldSprops, newSprops []britive_client.SensitivePropertyPlan) {
	newPropertyNames := make(map[string]interface{})
	for _, item := range newProps {
		newPropertyNames[item.Name.ValueString()] = nil
	}
	for _, item := range newSprops {
		newPropertyNames[item.Name.ValueString()] = nil
	}

	propertyTypeMap := make(map[string]interface{})
	for _, foundProp := range application.PropertyTypes {
		prop := map[string]interface{}{
			"value": foundProp.Value,
			"type":  foundProp.Type,
		}
		propertyTypeMap[foundProp.Name] = prop
	}

	var oldPropertiesList []interface{}
	for _, prop := range oldProps {
		oldPropertiesList = append(oldPropertiesList, map[string]interface{}{
			"name":  prop.Name.ValueString(),
			"value": prop.Value.ValueString(),
		})
	}
	for _, prop := range oldSprops {
		oldPropertiesList = append(oldPropertiesList, map[string]interface{}{
			"name":  prop.Name.ValueString(),
			"value": prop.Value.ValueString(),
		})
	}
	for _, propRaw := range oldPropertiesList {
		prop, _ := propRaw.(map[string]interface{})

		propName := prop["name"].(string)

		if _, found := newPropertyNames[propName]; !found {
			var property britive_client.PropertyTypes
			property.Name = propName

			valueAndType := propertyTypeMap[propName]
			propValue := valueAndType.(map[string]interface{})["value"]
			property.Value = propValue
			properties.PropertyTypes = append(properties.PropertyTypes, property)
		}
	}
}

func (rrth *ResourceApplicationHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ApplicationPlan, c britive_client.Client) (*britive_client.ApplicationPlan, error) {

	applicationID, err := rrth.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, "Reading apllication", map[string]interface{}{
		"applicationId": applicationID,
	})

	application, err := c.GetApplication(ctx, applicationID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("application %s", applicationID)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, "Received application", map[string]interface{}{
		"applicationId": applicationID,
	})

	if plan.ApplicationType.IsNull() || plan.ApplicationType.ValueString() == britive_client.EmptyString {
		plan.ApplicationType = types.StringValue(application.CatalogAppName)
	}
	plan.CatalogAppID = types.Int64Value(application.CatalogAppId)

	var planUserAccountMappings []britive_client.UserAccountMappingPlan
	for _, userAccMapRaw := range application.UserAccountMappings {
		userAccMap := userAccMapRaw.(map[string]interface{})
		var userAccMapPlan britive_client.UserAccountMappingPlan
		userAccMapPlan.Name = types.StringValue(userAccMap["name"].(string))
		userAccMapPlan.Description = types.StringValue(userAccMap["description"].(string))
		planUserAccountMappings = append(planUserAccountMappings, userAccMapPlan)
	}
	plan.UserAccountMappings, err = rrth.mapUserAccountMappingPlanToSet(planUserAccountMappings)
	if err != nil {
		return nil, err
	}

	plan.Version = types.StringValue(application.Properties.Version)

	allowedEnvGroupApps := map[int64]string{
		2: "AWS Standalone",
		8: "Okta",
		9: "Snowflake Standalone",
	}
	if _, ok := allowedEnvGroupApps[application.CatalogAppId]; ok {
		rootGroupId := ""
		rootEnvironmentGroups := application.RootEnvironmentGroup
		if rootEnvironmentGroups != nil {
			environmentGroups := rootEnvironmentGroups.EnvironmentGroups
			for _, envGroup := range environmentGroups {
				if envGroup.Name == "root" {
					rootGroupId = envGroup.ID
				}
			}
		}
		if rootGroupId != "" {
			plan.EntityRootEnvironmentGroupID = types.StringValue(rootGroupId)
		} else {
			plan.EntityRootEnvironmentGroupID = types.StringNull()
		}
	} else {
		plan.EntityRootEnvironmentGroupID = types.StringNull()
	}

	systemApps, err := c.GetSystemApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch system apps: %v", err)
	}

	var foundApp *britive_client.SystemApp
	for _, app := range systemApps {
		if app.CatalogAppId == application.CatalogAppId {
			foundApp = &app
			break
		}
	}
	if foundApp == nil {
		return nil, fmt.Errorf("failed to found the system app with catalog ID: %v", application.CatalogAppId)
	}
	systemPropertyTypeMap := make(map[string]map[string]interface{})
	for _, foundProp := range foundApp.PropertyTypes {
		prop := map[string]interface{}{
			"value": foundProp.Value,
			"type":  foundProp.Type,
		}
		systemPropertyTypeMap[foundProp.Name] = prop
	}

	applicationProperties := application.Properties.PropertyTypes

	var stateProperties []map[string]interface{}
	var stateSensitiveProperties []map[string]interface{}

	userProperties := make(map[string]interface{})
	planProperties, err := rrth.mapSetToPropertyPlan(plan.Properties)
	if err != nil {
		return nil, err
	}
	planSensitiveProperties, err := rrth.mapSetToSensitivePropertyPlan(plan.SensitiveProperties)
	if err != nil {
		return nil, err
	}
	for _, prop := range planProperties {
		propName := prop.Name.ValueString()
		propValue := prop.Value.ValueString()
		userProperties[propName] = propValue
	}

	for _, property := range applicationProperties {
		propertyName := property.Name
		propertyValType := property.Type
		propertyValue := property.Value

		if _, ok := userProperties[propertyName]; !ok || propertyName == "iconUrl" {
			if propertyValue == nil || propertyValue == "" || propertyName == "iconUrl" {
				continue
			}
		}
		if propertyValue == nil {
			propertyValue = ""
		} else {
			propertyValue = fmt.Sprintf("%v", propertyValue)
		}
		if propertyValType == "com.britive.pab.api.Secret" || propertyValType == "com.britive.pab.api.SecretFile" {
			if propertyValue == "*" {

				for _, sp := range planSensitiveProperties {
					if sp.Name.ValueString() == propertyName {
						propertyValue = sp.Value.ValueString()
						break
					}
				}
			}
			stateSensitiveProperties = append(stateSensitiveProperties, map[string]interface{}{
				"name":  propertyName,
				"value": propertyValue,
			})
		} else {
			if systemPropertyTypeMap[propertyName]["value"] != property.Value {
				stateProperties = append(stateProperties, map[string]interface{}{
					"name":  propertyName,
					"value": propertyValue,
				})
			} else {
				if _, ok := userProperties[propertyName]; ok {
					stateProperties = append(stateProperties, map[string]interface{}{
						"name":  propertyName,
						"value": propertyValue,
					})
				}
			}
		}
	}
	planProperties = nil
	for _, prop := range stateProperties {
		var propertyPlan britive_client.PropertyPlan
		propertyPlan.Name = types.StringValue(prop["name"].(string))
		propertyPlan.Value = types.StringValue(prop["value"].(string))
		planProperties = append(planProperties, propertyPlan)
	}
	plan.Properties, err = rrth.mapPropertyPlanToSet(planProperties)
	if err != nil {
		return nil, err
	}
	planSensitiveProperties = nil
	for _, prop := range stateSensitiveProperties {
		var sensPropertyPlan britive_client.SensitivePropertyPlan
		sensPropertyPlan.Name = types.StringValue(prop["name"].(string))
		sensPropertyPlan.Value = types.StringValue(prop["value"].(string))
		planSensitiveProperties = append(planSensitiveProperties, sensPropertyPlan)
	}
	plan.SensitiveProperties, err = rrth.mapSensitivePropertyPlanToSet(planSensitiveProperties)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (rrth *ResourceApplicationHelper) mapUserMappingsResourceToModel(plan britive_client.ApplicationPlan, userMappings *britive_client.UserMappings) error {
	planUserAccountMappings, err := rrth.mapSetToUserAccountMappingPlan(plan.UserAccountMappings)
	if err != nil {
		return err
	}
	for _, user := range planUserAccountMappings {
		userMapping := britive_client.UserMapping{}
		userMapping.Name = user.Name.ValueString()
		userMapping.Description = user.Description.ValueString()
		userMappings.UserAccountMappings = append(userMappings.UserAccountMappings, userMapping)
	}
	return nil
}

func (rrth *ResourceApplicationHelper) mapPropertiesResourceToModel(plan britive_client.ApplicationPlan, properties *britive_client.Properties, appResponse *britive_client.ApplicationResponse) error {

	applicationProperties := appResponse.Properties.PropertyTypes
	propertiesMap := make(map[string]string)
	for _, property := range applicationProperties {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Type)
	}

	planProperties, err := rrth.mapSetToPropertyPlan(plan.Properties)
	if err != nil {
		return err
	}
	for _, property := range planProperties {
		propertyType := britive_client.PropertyTypes{}
		propertyType.Name = property.Name.ValueString()

		if propertyType.Name == "iconUrl" {
			continue
		}

		if propertiesMap[propertyType.Name] == "java.lang.Boolean" {
			propertyValue, err := strconv.ParseBool(property.Value.ValueString())
			if err != nil {
				return err
			}
			propertyType.Value = propertyValue
		} else {
			propertyValue := property.Value.ValueString()
			propertyType.Value = propertyValue
		}
		properties.PropertyTypes = append(properties.PropertyTypes, propertyType)
	}

	sensitivePropertiesMap := make(map[string]string)

	planSensitiveProperties, err := rrth.mapSetToSensitivePropertyPlan(plan.SensitiveProperties)
	if err != nil {
		return err
	}
	for _, property := range planSensitiveProperties {
		propertyName := property.Name.ValueString()
		propertyValue := property.Value.ValueString()
		if prePropertyValue, ok := sensitivePropertiesMap[propertyName]; ok {
			if rrth.isHashValue(prePropertyValue, propertyValue) {
				continue
			} else if rrth.isHashValue(propertyValue, prePropertyValue) {
				sensitivePropertiesMap[propertyName] = propertyValue
				continue
			} else {
				return errors.New("an error has occurred related to the sensitive properties")
			}
		} else {
			sensitivePropertiesMap[propertyName] = propertyValue
		}
	}

	for sensitivePropertyName, sensitivePropertyValue := range sensitivePropertiesMap {
		propertyType := britive_client.PropertyTypes{}
		propertyType.Name = sensitivePropertyName
		propertyType.Value = sensitivePropertyValue
		properties.PropertyTypes = append(properties.PropertyTypes, propertyType)
	}

	return nil
}

func (rrth *ResourceApplicationHelper) getHash(val string) string {
	hash := argon2.IDKey([]byte(val), []byte{}, 1, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(hash)
}

func (rrth *ResourceApplicationHelper) isHashValue(val string, hash string) bool {
	return hash == rrth.getHash(val)
}

func (rrth *ResourceApplicationHelper) generateUniqueID(applicationID string) string {
	return applicationID
}

func (resourceApplicationHelper *ResourceApplicationHelper) parseUniqueID(ID string) (applicationID string, err error) {
	return ID, nil
}

func (rrth *ResourceApplicationHelper) mapApplicationResourceToModel(application *britive_client.ApplicationRequest, applicationName string, appCatalogDetails *britive_client.SystemApp) {
	application.CatalogAppId = appCatalogDetails.CatalogAppId
	application.CatalogAppDisplayName = applicationName
}

func (rrth *ResourceApplicationHelper) getApplicationName(plan britive_client.ApplicationPlan) (string, error) {
	planProperties, err := rrth.mapSetToPropertyPlan(plan.Properties)
	if err != nil {
		return "", err
	}
	for _, property := range planProperties {
		propertyName := property.Name.ValueString()
		propertyValue := property.Value.ValueString()
		if propertyName == "displayName" {
			return propertyValue, nil
		}
	}
	return "", errors.New("missing mandatory property displayName")
}

func (rrth *ResourceApplicationHelper) validatePropertiesAgainstSystemApps(ctx context.Context, plan britive_client.ApplicationPlan, c *britive_client.Client) (*britive_client.SystemApp, error) {
	appType := strings.ToLower(plan.ApplicationType.ValueString())

	systemApps, err := c.GetSystemApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch system apps: %v", err)
	}

	latestVersion, allAppVersions := rrth.getLatestVersion(systemApps, appType)

	if plan.Version.IsNull() || plan.Version.ValueString() == britive_client.EmptyString {
		tflog.Info(ctx, "Selected latest version", map[string]interface{}{
			"Latest version": latestVersion,
		})
		plan.Version = types.StringValue(latestVersion)
	}

	var foundApp *britive_client.SystemApp
	for _, app := range systemApps {
		if strings.ToLower(app.Name) == appType && app.Version == plan.Version.ValueString() {
			foundApp = &app
			break
		}
	}
	if foundApp == nil {
		return nil, fmt.Errorf("application_type '%s' with version '%s' not supportted by britive. \nTry %v versions", appType, plan.Version.ValueString(), allAppVersions)
	}
	tflog.Info(ctx, fmt.Sprintf("Selecting catalog with id %d for application of type %s.", foundApp.CatalogAppId, plan.ApplicationType.ValueString()))

	allowedProps := map[string]britive_client.SystemAppPropertyType{}
	allowedSensitive := map[string]britive_client.SystemAppPropertyType{}
	for _, pt := range foundApp.PropertyTypes {
		if pt.Type == "com.britive.pab.api.Secret" || pt.Type == "com.britive.pab.api.SecretFile" {
			allowedSensitive[pt.Name] = pt
		} else {
			allowedProps[pt.Name] = pt
		}
	}
	// Validate properti
	userProps := map[string]bool{}
	planProperties, err := rrth.mapSetToPropertyPlan(plan.Properties)
	if err != nil {
		return nil, err
	}
	for _, prop := range planProperties {
		name := prop.Name.ValueString()
		val := prop.Value.ValueString()
		userProps[name] = true
		pt, ok := allowedProps[name]
		if !ok {
			return nil, fmt.Errorf("property '%s' is not supported for application type '%s'", name, foundApp.Name)
		}
		// Type validation for non-sensitive properties
		if err := rrth.validatePropertyValueType(val, pt.Type, name); err != nil {
			return nil, err
		}
	}
	// Validate sensitive_properties
	userSensitive := map[string]bool{}
	planSensitiveProperties, err := rrth.mapSetToSensitivePropertyPlan(plan.SensitiveProperties)
	if err != nil {
		return nil, err
	}
	for _, prop := range planSensitiveProperties {
		name := prop.Name.ValueString()
		userSensitive[name] = true
		if _, ok := allowedSensitive[name]; !ok {
			return nil, fmt.Errorf("sensitive property '%s' is not supported for application type '%s'", name, foundApp.Name)
		}
	}
	return foundApp, nil
}

func (rrth *ResourceApplicationHelper) validatePropertyValueType(val string, typ string, name string) error {
	switch typ {
	case "java.lang.Boolean":
		if _, err := strconv.ParseBool(val); err != nil {
			return fmt.Errorf("property '%s' value '%s' is not a valid boolean", name, val)
		}
	case "java.lang.Integer", "java.lang.Long", "java.time.Duration":
		if _, err := strconv.ParseInt(val, 10, 64); err != nil {
			return fmt.Errorf("property '%s' value '%s' is not a valid integer", name, val)
		}
	case "java.lang.Float", "java.lang.Double":
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return fmt.Errorf("property '%s' value '%s' is not a valid float", name, val)
		}
	// For secrets, files, and strings, accept any string
	case "java.lang.String":
		// no-op
	default:
		// Unknown type, skip validation
	}
	return nil
}

func (rrth *ResourceApplicationHelper) getLatestVersion(systemApps []britive_client.SystemApp, appType string) (string, []string) {
	latestVersionParts := []string{"0", "0", "0", "0", "0"}
	var latestVersion string
	var allAppVersions []string
	for _, app := range systemApps {
		if strings.ToLower(app.Name) == appType {
			appVersionStr := strings.TrimPrefix(app.Version, "Custom-")
			appVersionParts := strings.Split(appVersionStr, ".")

			var size int
			if len(appVersionParts) <= len(latestVersionParts) {
				size = len(appVersionParts)
			} else {
				size = len(latestVersionParts)
			}
			for i := 0; i < size; i++ {
				if appVersionParts[i] > latestVersionParts[i] {
					latestVersionParts = appVersionParts
					latestVersion = app.Version
					break
				}
			}
			allAppVersions = append(allAppVersions, app.Version)
		}
	}
	return latestVersion, allAppVersions
}

func (rrth *ResourceApplicationHelper) mapPropertyPlanToSet(plans []britive_client.PropertyPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
			map[string]attr.Value{
				"name":  p.Name,
				"value": p.Value,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for property: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create properties set: %v", diags)
	}

	return set, nil
}

func (rrth *ResourceApplicationHelper) mapSetToPropertyPlan(set types.Set) ([]britive_client.PropertyPlan, error) {
	var result []britive_client.PropertyPlan
	objs := set.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected Object, got %T", e)
		}
		var p britive_client.PropertyPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert object to PropertyPlan: %v", diags)
		}
		result = append(result, p)
	}
	return result, nil
}

func (rrt *ResourceApplicationHelper) mapSensitivePropertyPlanToSet(plans []britive_client.SensitivePropertyPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
			map[string]attr.Value{
				"name":  p.Name,
				"value": p.Value,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for sensitive property: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create sensitive properties set: %v", diags)
	}

	return set, nil
}

func (rrth *ResourceApplicationHelper) mapSetToSensitivePropertyPlan(set types.Set) ([]britive_client.SensitivePropertyPlan, error) {
	var result []britive_client.SensitivePropertyPlan
	objs := set.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected Object, got %T", e)
		}
		var p britive_client.SensitivePropertyPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert object to SensitivePropertyPlan: %v", diags)
		}
		result = append(result, p)
	}
	return result, nil
}

func (rrth *ResourceApplicationHelper) mapUserAccountMappingPlanToSet(plans []britive_client.UserAccountMappingPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"name":        types.StringType,
				"description": types.StringType,
			},
			map[string]attr.Value{
				"name":        p.Name,
				"description": p.Description,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for user account mapping: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":        types.StringType,
				"description": types.StringType,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create user account mapping set: %v", diags)
	}

	return set, nil
}

func (rrth *ResourceApplicationHelper) mapSetToUserAccountMappingPlan(set types.Set) ([]britive_client.UserAccountMappingPlan, error) {
	var result []britive_client.UserAccountMappingPlan
	objs := set.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected Object, got %T", e)
		}
		var p britive_client.UserAccountMappingPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert object to UserAccountMappingPlan: %v", diags)
		}
		result = append(result, p)
	}
	return result, nil
}
