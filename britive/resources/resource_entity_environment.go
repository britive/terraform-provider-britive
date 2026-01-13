package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/argon2"
)

var (
	_ resource.Resource                = &ResourceEntityEnvironment{}
	_ resource.ResourceWithConfigure   = &ResourceEntityEnvironment{}
	_ resource.ResourceWithImportState = &ResourceEntityEnvironment{}
)

type ResourceEntityEnvironment struct {
	client       *britive_client.Client
	helper       *ResourceEntityEnvironmentHelper
	importHelper *imports.ImportHelper
}

type ResourceEntityEnvironmentHelper struct{}

func NewResourceEntityEnvironment() resource.Resource {
	return &ResourceEntityEnvironment{}
}

func NewResourceEntityEnvironmentHelper() *ResourceEntityEnvironmentHelper {
	return &ResourceEntityEnvironmentHelper{}
}

func (ree *ResourceEntityEnvironment) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_entity_environment"
}

func (ree *ResourceEntityEnvironment) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Entity Environment resource")

	if req.ProviderData == nil {
		return
	}

	ree.client = req.ProviderData.(*britive_client.Client)
	if ree.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Entity Environment")
	ree.helper = NewResourceEntityEnvironmentHelper()
}

func (ree *ResourceEntityEnvironment) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for entity environment resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The identity of the application entity of type environment",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"entityId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The identity of the Britive application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"applicationId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"parent_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent group id under which the environment will be created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"parentGroupId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"properties": schema.SetNestedBlock{
				Description: "Britive application entity environment overwrite properties.",
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
				Description: "Britive application entity environment overwrite sensitive properties.",
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
		},
	}
}

func (ree *ResourceEntityEnvironment) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_entity_environment")

	var plan britive_client.EntityEnvironmentPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during entity_environment creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	applicationEntity := britive_client.ApplicationEntityEnvironment{}

	err := ree.helper.mapResourceToModel(plan, &applicationEntity)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create entity environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new application entity environment: %#v", applicationEntity))

	applicationID := plan.ApplicationID.ValueString()

	ae, err := ree.client.CreateEntityEnvironment(ctx, applicationEntity, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create entity environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create entity environment, %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new application entity environment: %#v", ae))
	plan.ID = types.StringValue(ree.helper.generateUniqueID(applicationID, ae.EntityID))

	// Get application environment for entity with type environment
	appEnvDetails, err := ree.client.GetApplicationEnvironment(ctx, applicationID, ae.EntityID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch entity environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch entity environment, %#v", err))
		return
	}

	// Patch properties
	properties := britive_client.Properties{}
	err = ree.helper.mapPropertiesResourceToModel(plan, &properties, appEnvDetails)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create entity environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map properties to model, %#v", err))
		return
	}

	tflog.Info(ctx, "Updating application environment properties")
	_, err = ree.client.PatchApplicationEnvPropertyTypes(ctx, applicationID, ae.EntityID, properties)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map properties to entity enviroment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map properties to entity enviroment, %#v", err.Error()))
		return
	}

	planPtr, err := ree.helper.getAndMapModelToPlan(ctx, plan, *ree.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get entity_environment",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map entity_environment model to plan", map[string]interface{}{
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
		"entity_environment": planPtr,
	})
}

func (ree *ResourceEntityEnvironment) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_entity_environment")

	if ree.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.EntityEnvironmentPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get entity environment state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := ree.helper.getAndMapModelToPlan(ctx, state, *ree.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get entity environment",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map entity environment model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read entity environment:  %#v", planPtr))
}

func (ree *ResourceEntityEnvironment) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_entity_environment")

	var plan, state britive_client.EntityEnvironmentPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	var hasChanges bool
	if !plan.Properties.Equal(state.Properties) || !plan.SensitiveProperties.Equal(state.SensitiveProperties) {
		hasChanges = true
		applicationID, entityID, err := ree.helper.parseUniqueID(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to update entity_environment", "Failed to fetch entity_environment id")
			tflog.Error(ctx, fmt.Sprintf("Failed to parse id, %#v", err))
			return
		}

		// Get application Environment for entity with type Environment
		appEnvDetails, err := ree.client.GetApplicationEnvironment(ctx, applicationID, entityID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update entity_environment", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to get entity_environment, %#v", err))
			return
		}

		// Patch properties
		properties := britive_client.Properties{}
		err = ree.helper.mapPropertiesResourceToModel(plan, &properties, appEnvDetails)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update entity_environment", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map properties to resource model, %#v", err))
			return
		}

		tflog.Info(ctx, "Updating application entity environment properties")
		_, err = ree.client.PatchApplicationEnvPropertyTypes(ctx, applicationID, entityID, properties)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update entity_environment properties", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update entity_environment properties, %#v", err))
			return
		}
		tflog.Info(ctx, "Updated application entity environment properties")
	}
	if hasChanges {
		planPtr, err := ree.helper.getAndMapModelToPlan(ctx, plan, *ree.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map entity environment model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated entity environment: %#v", planPtr))
	}
}

func (ree *ResourceEntityEnvironment) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_entity_environment")

	var state britive_client.EntityEnvironmentPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	applicationID, entityID, err := ree.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete entity environment", "appicationID or entityID not found")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse applicationID, entityID: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting entity environment %s for application %s", entityID, applicationID))
	err = ree.client.DeleteEntityEnvironment(ctx, applicationID, entityID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete entity environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete entity environment: %#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Deleted entity environment %s for application %s", entityID, applicationID))
	resp.State.RemoveResource(ctx)
}

func (ree *ResourceEntityEnvironment) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := ree.importHelper.ParseImportID([]string{"apps/(?P<application_id>[^/]+)/root-environment-group/environments/(?P<entity_id>[^/]+)", "(?P<application_id>[^/]+)/environments/(?P<entity_id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import entity environment", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	applicationID := importData.Fields["application_id"]
	entityID := importData.Fields["entity_id"]
	if strings.TrimSpace(applicationID) == "" {
		resp.Diagnostics.AddError("Failed to import entitty group", "Invalid applicationID")
		tflog.Error(ctx, "Failed to import entity group, Invalid applicationID")
		return
	}
	if strings.TrimSpace(entityID) == "" {
		resp.Diagnostics.AddError("Failed to import entitty group", "Invalid entityID")
		tflog.Error(ctx, "Failed to import entity group, Invalid entityID")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing entity %s of type environment for application %s", entityID, applicationID))

	appEnvs, err := ree.client.GetAppEnvs(ctx, applicationID, "environments")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import entity_environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch application environments, %#v", err))
		return
	}
	envIdList, err := ree.client.GetEnvDetails(appEnvs, "id")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import entity_environment", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch list of environments, %#v", err))
		return
	}

	for _, id := range envIdList {
		if id == entityID {
			plan := &britive_client.EntityEnvironmentPlan{
				ID: types.StringValue(ree.helper.generateUniqueID(applicationID, entityID)),
			}

			planPtr, err := ree.helper.getAndMapModelToPlan(ctx, *plan, *ree.client)
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to import entity environment",
					fmt.Sprintf("Error: %v", err),
				)
				tflog.Error(ctx, "Failed import entity environment model to plan", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}

			importedPlan, err := ree.helper.importAndMapPropertiesToResource(ctx, *planPtr, *ree.client)
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to import entity environment properties",
					fmt.Sprintf("Error: %v", err),
				)
				tflog.Error(ctx, "Failed import entity environment model to plan", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, importedPlan)...)
			if resp.Diagnostics.HasError() {
				tflog.Error(ctx, "Failed to set state after import", map[string]interface{}{
					"diagnostics": resp.Diagnostics,
				})
				return
			}

			tflog.Info(ctx, fmt.Sprintf("Imported entity environment : %#v", importedPlan))
			return
		}
	}
}

func (reeh *ResourceEntityEnvironmentHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.EntityEnvironmentPlan, c britive_client.Client) (*britive_client.EntityEnvironmentPlan, error) {

	applicationID, entityID, err := reeh.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, "Reading entitiy environment", map[string]interface{}{
		"applicationId": applicationID,
		"entityID":      entityID,
	})

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(ctx, applicationID)
	if err != nil || appRootEnvironmentGroup == nil {
		return nil, err
	}

	associationFound := false
	for _, association := range appRootEnvironmentGroup.Environments {
		if association.ID == entityID {
			associationFound = true
			tflog.Info(ctx, fmt.Sprintf("Received entity environment: %#v", association))
			plan.EntityID = types.StringValue(entityID)
			plan.ParentGroupID = types.StringValue(association.ParentGroupID)
		}
	}
	if !associationFound {
		return nil, errs.NewNotFoundErrorf("entity environment %s for application %s", entityID, applicationID)
	}

	tflog.Info(ctx, "Reading properties of entity environment")

	applicationEnvironmentDetails, err := c.GetApplicationEnvironment(ctx, applicationID, entityID)
	if err != nil {
		return nil, err
	}

	applicationProperties := applicationEnvironmentDetails.Properties.PropertyTypes
	propertiesMap := make(map[string]string)
	for _, property := range applicationProperties {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Value)
	}

	var stateProperties []britive_client.PropertyPlan
	var stateSensitiveProperties []britive_client.SensitivePropertyPlan
	properties, err := reeh.mapSetToPropertyPlan(plan.Properties)
	if err != nil {
		return nil, err
	}
	sensitiveProperties, err := reeh.mapSetToSensitivePropertyPlan(plan.SensitiveProperties)
	if err != nil {
		return nil, err
	}

	for _, property := range properties {
		propertyName := property.Name.ValueString()
		stateProperties = append(stateProperties, britive_client.PropertyPlan{
			Name:  types.StringValue(propertyName),
			Value: types.StringValue(propertiesMap[propertyName]),
		})
	}
	for _, property := range sensitiveProperties {
		propertyName := property.Name.ValueString()
		if propertiesMap[propertyName] == "*" {
			for _, existing := range sensitiveProperties {
				if existing.Name.ValueString() == propertyName {
					propertiesMap[propertyName] = existing.Value.ValueString()
					break
				}
			}
		}
		stateSensitiveProperties = append(stateSensitiveProperties, britive_client.SensitivePropertyPlan{
			Name:  types.StringValue(propertyName),
			Value: types.StringValue(propertiesMap[propertyName]),
		})
	}
	stateSetProperties, err := reeh.mapPropertyPlanToSet(stateProperties)
	if err != nil {
		return nil, err
	}
	stateSetSensitiveProperties, err := reeh.mapSensitivePropertyPlanToSet(stateSensitiveProperties)
	if err != nil {
		return nil, err
	}
	plan.Properties = stateSetProperties
	plan.SensitiveProperties = stateSetSensitiveProperties

	tflog.Info(ctx, fmt.Sprintf("Read entity_environemt, %#v", plan))

	return &plan, nil
}

func (reeh *ResourceEntityEnvironmentHelper) mapResourceToModel(plan britive_client.EntityEnvironmentPlan, applicationEntity *britive_client.ApplicationEntityEnvironment) error {
	entityName, err := reeh.getPropertyByKey(plan, "displayName")
	if err != nil {
		return err
	}
	description, err := reeh.getPropertyByKey(plan, "description")
	if err != nil {
		return err
	}
	applicationEntity.Name = entityName
	applicationEntity.Description = description
	applicationEntity.ParentGroupID = plan.ParentGroupID.ValueString()
	applicationEntity.EntityID = plan.EntityID.ValueString()
	return nil
}

func (reeh *ResourceEntityEnvironmentHelper) getPropertyByKey(plan britive_client.EntityEnvironmentPlan, key string) (string, error) {
	propertyTypes, err := reeh.mapSetToPropertyPlan(plan.Properties)
	if err != nil {
		return "", err
	}
	for _, property := range propertyTypes {
		propertyName := property.Name.ValueString()
		propertyValue := property.Value.ValueString()
		if propertyName == key {
			return propertyValue, nil
		}
	}
	return "", errors.New("Missing mandatory property " + key)
}

func (reeh *ResourceEntityEnvironmentHelper) mapPropertiesResourceToModel(plan britive_client.EntityEnvironmentPlan, properties *britive_client.Properties, appResponse *britive_client.ApplicationResponse) error {

	applicationProperties := appResponse.Properties.PropertyTypes
	propertiesMap := make(map[string]string)
	for _, property := range applicationProperties {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Type)
	}

	planProperties, err := reeh.mapSetToPropertyPlan(plan.Properties)
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

	planSensitiveProperties, err := reeh.mapSetToSensitivePropertyPlan(plan.SensitiveProperties)
	if err != nil {
		return err
	}
	for _, property := range planSensitiveProperties {
		propertyName := property.Name.ValueString()
		propertyValue := property.Value.ValueString()
		if prePropertyValue, ok := sensitivePropertiesMap[propertyName]; ok {
			if reeh.isHashValue(prePropertyValue, propertyValue) {
				continue
			} else if reeh.isHashValue(propertyValue, prePropertyValue) {
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

func (reeh *ResourceEntityEnvironmentHelper) importAndMapPropertiesToResource(ctx context.Context, plan britive_client.EntityEnvironmentPlan, c britive_client.Client) (*britive_client.EntityEnvironmentPlan, error) {
	applicationID, entityID, err := reeh.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Importing properties for entity %s for application %s", entityID, applicationID))

	// Get application Environment for entity with type Environment
	appEnvDetails, err := c.GetApplicationEnvironment(ctx, applicationID, entityID)
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received entity env %#v", appEnvDetails))
	//ToDo: check env type should be envrionment

	var stateProperties []britive_client.PropertyPlan
	var stateSensitiveProperties []britive_client.SensitivePropertyPlan
	applicationProperties := appEnvDetails.Properties.PropertyTypes
	for _, property := range applicationProperties {
		propertyName := property.Name

		if property.Type == "com.britive.pab.api.Secret" || property.Type == "com.britive.pab.api.SecretFile" {
			stateSensitiveProperties = append(stateSensitiveProperties, britive_client.SensitivePropertyPlan{
				Name:  types.StringValue(propertyName),
				Value: types.StringValue(fmt.Sprintf("%v", property.Value)),
			})
		} else {
			stateProperties = append(stateProperties, britive_client.PropertyPlan{
				Name:  types.StringValue(propertyName),
				Value: types.StringValue(fmt.Sprintf("%v", property.Value)),
			})
		}
	}
	stateSetProperties, err := reeh.mapPropertyPlanToSet(stateProperties)
	if err != nil {
		return nil, err
	}
	stateSetSensitiveProperties, err := reeh.mapSensitivePropertyPlanToSet(stateSensitiveProperties)
	if err != nil {
		return nil, err
	}
	plan.Properties = stateSetProperties
	plan.SensitiveProperties = stateSetSensitiveProperties
	return &plan, nil
}

func (reeh *ResourceEntityEnvironmentHelper) getHash(val string) string {
	hash := argon2.IDKey([]byte(val), []byte{}, 1, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(hash)
}

func (reeh *ResourceEntityEnvironmentHelper) isHashValue(val string, hash string) bool {
	return hash == reeh.getHash(val)
}

func (resourceEntityEnvironmentHelper *ResourceEntityEnvironmentHelper) generateUniqueID(applicationID, entityID string) string {
	return fmt.Sprintf("apps/%s/root-environment-group/environments/%s", applicationID, entityID)
}

func (resourceEntityEnvironmentHelper *ResourceEntityEnvironmentHelper) parseUniqueID(ID string) (applicationID, entityID string, err error) {
	applicationEntityParts := strings.Split(ID, "/")
	if len(applicationEntityParts) < 5 {
		err = errs.NewInvalidResourceIDError("application entity environment", ID)
		return
	}

	applicationID = applicationEntityParts[1]
	entityID = applicationEntityParts[4]
	return
}

func (reeh *ResourceEntityEnvironmentHelper) mapPropertyPlanToSet(plans []britive_client.PropertyPlan) (types.Set, error) {
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

func (reeh *ResourceEntityEnvironmentHelper) mapSetToPropertyPlan(set types.Set) ([]britive_client.PropertyPlan, error) {
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

func (reeh *ResourceEntityEnvironmentHelper) mapSensitivePropertyPlanToSet(plans []britive_client.SensitivePropertyPlan) (types.Set, error) {
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

func (reeh *ResourceEntityEnvironmentHelper) mapSetToSensitivePropertyPlan(set types.Set) ([]britive_client.SensitivePropertyPlan, error) {
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
