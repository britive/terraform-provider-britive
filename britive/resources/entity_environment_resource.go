package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EntityEnvironmentResource struct {
	client *britive.Client
}

type EntityEnvironmentResourceModel struct {
	ID                  types.String          `tfsdk:"id"`
	EntityID            types.String          `tfsdk:"entity_id"`
	ApplicationID       types.String          `tfsdk:"application_id"`
	ParentGroupID       types.String          `tfsdk:"parent_group_id"`
	Properties          []EntityPropertyModel `tfsdk:"properties"`
	SensitiveProperties []EntityPropertyModel `tfsdk:"sensitive_properties"`
}

type EntityPropertyModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func NewEntityEnvironmentResource() resource.Resource {
	return &EntityEnvironmentResource{}
}

func (r *EntityEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity_environment"
}

func (r *EntityEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive application entity environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The identity of the application entity of type environment",
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The identity of the Britive application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent group id under which the environment will be created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
							Description: "Britive application entity environment property name.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "Britive application entity environment property value.",
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
							Description: "Britive application entity environment property name.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "Britive application entity environment property value (stored as hash).",
						},
					},
				},
			},
		},
	}
}

func (r *EntityEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *EntityEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EntityEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationEntity, err := r.mapResourceToModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}

	applicationID := plan.ApplicationID.ValueString()

	log.Printf("[INFO] Creating new application entity environment: %#v", applicationEntity)

	ae, err := r.client.CreateEntityEnvironment(applicationEntity, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Entity Environment", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new application entity environment: %#v", ae)

	plan.ID = types.StringValue(fmt.Sprintf("apps/%s/root-environment-group/environments/%s", applicationID, ae.EntityID))
	plan.EntityID = types.StringValue(ae.EntityID)

	// Get application environment for entity properties
	appEnvDetails, err := r.client.GetApplicationEnvironment(applicationID, ae.EntityID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Application Environment", err.Error())
		return
	}

	// Patch properties
	properties, err := r.mapPropertiesResourceToModel(&plan, appEnvDetails)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Properties", err.Error())
		return
	}

	log.Printf("[INFO] Updating application environment properties")
	_, err = r.client.PatchApplicationEnvPropertyTypes(applicationID, ae.EntityID, properties)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Properties", err.Error())
		return
	}
	log.Printf("[INFO] Updated application environment properties")

	// Read back state
	err = r.readAndMapState(applicationID, ae.EntityID, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Entity Environment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EntityEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EntityEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, entityID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Reading entity environment %s for application %s", entityID, applicationID)

	err = r.readAndMapState(applicationID, entityID, &state)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Entity Environment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EntityEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EntityEnvironmentResourceModel
	var state EntityEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, entityID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	plan.ID = state.ID
	plan.EntityID = state.EntityID

	// Get application Environment for entity with type Environment
	appEnvDetails, err := r.client.GetApplicationEnvironment(applicationID, entityID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Application Environment", err.Error())
		return
	}

	// Patch properties
	properties, err := r.mapPropertiesResourceToModel(&plan, appEnvDetails)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Properties", err.Error())
		return
	}

	log.Printf("[INFO] Updating application entity environment properties")
	_, err = r.client.PatchApplicationEnvPropertyTypes(applicationID, entityID, properties)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Properties", err.Error())
		return
	}
	log.Printf("[INFO] Updated application entity environment properties")

	err = r.readAndMapState(applicationID, entityID, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Entity Environment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EntityEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EntityEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, entityID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting entity %s of type environment for application %s", entityID, applicationID)
	err = r.client.DeleteEntityEnvironment(applicationID, entityID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Entity Environment", err.Error())
		return
	}
	log.Printf("[INFO] Deleted entity %s of type environment for application %s", entityID, applicationID)
}

func (r *EntityEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var applicationID, entityID string

	// Support two formats: "apps/{app_id}/root-environment-group/environments/{entity_id}" or "{app_id}/environments/{entity_id}"
	if strings.HasPrefix(importID, "apps/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 5 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'apps/{app_id}/root-environment-group/environments/{entity_id}' or '{app_id}/environments/{entity_id}', got: %s", importID))
			return
		}
		applicationID = parts[1]
		entityID = parts[4]
	} else {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 || parts[1] != "environments" {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'apps/{app_id}/root-environment-group/environments/{entity_id}' or '{app_id}/environments/{entity_id}', got: %s", importID))
			return
		}
		applicationID = parts[0]
		entityID = parts[2]
	}

	if strings.TrimSpace(applicationID) == "" || strings.TrimSpace(entityID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Application ID and Entity ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing entity %s of type environment for application %s", entityID, applicationID)

	// Verify entity exists in app environments
	appEnvs, err := r.client.GetAppEnvs(applicationID, "environments")
	if err != nil {
		resp.Diagnostics.AddError("Error Getting App Environments", err.Error())
		return
	}

	envIDList, err := r.client.GetEnvDetails(appEnvs, "id")
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Env Details", err.Error())
		return
	}

	found := false
	for _, id := range envIDList {
		if id == entityID {
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Entity Environment Not Found", fmt.Sprintf("entity %s of type environment for application %s not found", entityID, applicationID))
		return
	}

	var state EntityEnvironmentResourceModel
	state.ID = types.StringValue(fmt.Sprintf("apps/%s/root-environment-group/environments/%s", applicationID, entityID))
	state.ApplicationID = types.StringValue(applicationID)
	state.EntityID = types.StringValue(entityID)

	// Get application environment details and map properties
	appEnvDetails, err := r.client.GetApplicationEnvironment(applicationID, entityID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Application Environment", err.Error())
		return
	}

	for _, property := range appEnvDetails.Properties.PropertyTypes {
		propName := property.Name
		propValue := fmt.Sprintf("%v", property.Value)

		if property.Type == "com.britive.pab.api.Secret" || property.Type == "com.britive.pab.api.SecretFile" {
			state.SensitiveProperties = append(state.SensitiveProperties, EntityPropertyModel{
				Name:  types.StringValue(propName),
				Value: types.StringValue(propValue),
			})
		} else {
			state.Properties = append(state.Properties, EntityPropertyModel{
				Name:  types.StringValue(propName),
				Value: types.StringValue(propValue),
			})
		}
	}

	// Get parent_group_id from root environment group
	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err == nil && appRootEnvironmentGroup != nil {
		for _, assoc := range appRootEnvironmentGroup.Environments {
			if assoc.ID == entityID {
				state.ParentGroupID = types.StringValue(assoc.ParentGroupID)
				break
			}
		}
	}

	log.Printf("[INFO] Imported entity %s of type environment for application %s", entityID, applicationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *EntityEnvironmentResource) mapResourceToModel(plan *EntityEnvironmentResourceModel) (britive.ApplicationEntityEnvironment, error) {
	entity := britive.ApplicationEntityEnvironment{
		ParentGroupID: plan.ParentGroupID.ValueString(),
		EntityID:      plan.EntityID.ValueString(),
	}

	for _, prop := range plan.Properties {
		switch prop.Name.ValueString() {
		case "displayName":
			entity.Name = prop.Value.ValueString()
		case "description":
			entity.Description = prop.Value.ValueString()
		}
	}

	if entity.Name == "" {
		return entity, errors.New("Missing mandatory property displayName")
	}

	return entity, nil
}

func (r *EntityEnvironmentResource) mapPropertiesResourceToModel(plan *EntityEnvironmentResourceModel, appResponse *britive.ApplicationResponse) (britive.Properties, error) {
	properties := britive.Properties{}

	// Build a type map from API response
	propertiesTypeMap := make(map[string]string)
	for _, property := range appResponse.Properties.PropertyTypes {
		propertiesTypeMap[property.Name] = fmt.Sprintf("%v", property.Type)
	}

	// Map regular properties
	for _, prop := range plan.Properties {
		pt := britive.PropertyTypes{}
		pt.Name = prop.Name.ValueString()

		if propertiesTypeMap[pt.Name] == "java.lang.Boolean" {
			boolVal, err := strconv.ParseBool(prop.Value.ValueString())
			if err != nil {
				return properties, err
			}
			pt.Value = boolVal
		} else {
			pt.Value = prop.Value.ValueString()
		}
		properties.PropertyTypes = append(properties.PropertyTypes, pt)
	}

	// Map sensitive properties (deduplicate and handle hashing)
	sensitivePropertiesMap := make(map[string]string)
	for _, prop := range plan.SensitiveProperties {
		propName := prop.Name.ValueString()
		propValue := prop.Value.ValueString()

		if existing, ok := sensitivePropertiesMap[propName]; ok {
			if planmodifiers.IsHashValue(existing, propValue) {
				continue
			} else if planmodifiers.IsHashValue(propValue, existing) {
				sensitivePropertiesMap[propName] = propValue
				continue
			} else {
				return properties, fmt.Errorf("conflicting values for sensitive property %s", propName)
			}
		} else {
			sensitivePropertiesMap[propName] = propValue
		}
	}

	for name, value := range sensitivePropertiesMap {
		properties.PropertyTypes = append(properties.PropertyTypes, britive.PropertyTypes{
			Name:  name,
			Value: value,
		})
	}

	return properties, nil
}

func (r *EntityEnvironmentResource) readAndMapState(applicationID, entityID string, state *EntityEnvironmentResourceModel) error {
	state.ApplicationID = types.StringValue(applicationID)
	state.EntityID = types.StringValue(entityID)

	// Get basic entity info from root environment group
	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		return err
	}
	if appRootEnvironmentGroup == nil {
		return britive.ErrNotFound
	}

	found := false
	for _, assoc := range appRootEnvironmentGroup.Environments {
		if assoc.ID == entityID {
			state.ParentGroupID = types.StringValue(assoc.ParentGroupID)
			found = true
			break
		}
	}

	if !found {
		return britive.ErrNotFound
	}

	// Get properties
	applicationEnvironmentDetails, err := r.client.GetApplicationEnvironment(applicationID, entityID)
	if err != nil {
		return err
	}

	apiPropsMap := make(map[string]string)
	for _, property := range applicationEnvironmentDetails.Properties.PropertyTypes {
		apiPropsMap[property.Name] = fmt.Sprintf("%v", property.Value)
	}

	// Update regular properties from current state
	var updatedProperties []EntityPropertyModel
	for _, prop := range state.Properties {
		propName := prop.Name.ValueString()
		updatedProperties = append(updatedProperties, EntityPropertyModel{
			Name:  types.StringValue(propName),
			Value: types.StringValue(apiPropsMap[propName]),
		})
	}
	state.Properties = updatedProperties

	// Update sensitive properties - preserve hashed values when API returns "*"
	var updatedSensitiveProperties []EntityPropertyModel
	for _, prop := range state.SensitiveProperties {
		propName := prop.Name.ValueString()
		apiValue := apiPropsMap[propName]
		stateValue := prop.Value.ValueString()

		var newValue string
		if apiValue == "*" {
			// API masked the value, keep the hashed state value
			newValue = stateValue
		} else {
			newValue = planmodifiers.GetHash(apiValue)
		}

		updatedSensitiveProperties = append(updatedSensitiveProperties, EntityPropertyModel{
			Name:  types.StringValue(propName),
			Value: types.StringValue(newValue),
		})
	}
	state.SensitiveProperties = updatedSensitiveProperties

	return nil
}

func (r *EntityEnvironmentResource) parseUniqueID(id string) (applicationID, entityID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 5 {
		return "", "", fmt.Errorf("invalid resource ID format: %s (expected 'apps/{applicationID}/root-environment-group/environments/{entityID}')", id)
	}
	return parts[1], parts[4], nil
}
