package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/argon2"
)

var (
	_ resource.Resource                   = &ApplicationResource{}
	_ resource.ResourceWithConfigure      = &ApplicationResource{}
	_ resource.ResourceWithImportState    = &ApplicationResource{}
	_ resource.ResourceWithValidateConfig = &ApplicationResource{}
)

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

type ApplicationResource struct {
	client *britive.Client
}

type ApplicationResourceModel struct {
	ID                           types.String              `tfsdk:"id"`
	ApplicationType              types.String              `tfsdk:"application_type"`
	Version                      types.String              `tfsdk:"version"`
	CatalogAppID                 types.Int64               `tfsdk:"catalog_app_id"`
	EntityRootEnvironmentGroupID types.String              `tfsdk:"entity_root_environment_group_id"`
	Properties                   []PropertyModel           `tfsdk:"properties"`
	SensitiveProperties          []PropertyModel           `tfsdk:"sensitive_properties"`
	UserAccountMappings          []UserAccountMappingModel `tfsdk:"user_account_mappings"`
}

type PropertyModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type UserAccountMappingModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *ApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_type": schema.StringAttribute{
				Required:    true,
				Description: "Britive application type. Supported types: 'Snowflake', 'Snowflake Standalone', 'GCP', 'GCP Standalone', 'GCP WIF', 'Google Workspace', 'AWS', 'AWS Standalone', 'Azure', 'Okta'.",
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Snowflake", "Snowflake Standalone", "GCP", "GCP Standalone", "GCP WIF", "Google Workspace", "AWS", "AWS Standalone", "Azure", "Okta"),
				},
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Britive application version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_app_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Britive application base catalog id.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"entity_root_environment_group_id": schema.StringAttribute{
				Computed:    true,
				Description: "Britive application root environment ID for Snowflake Standalone applications.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"properties": schema.SetNestedBlock{
				Description: "Britive application overwrite properties.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Britive application property name.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "Britive application property value.",
						},
					},
				},
			},
			"sensitive_properties": schema.SetNestedBlock{
				Description: "Britive application overwrite sensitive properties.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Britive application property name.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "Britive application property value.",
							PlanModifiers: []planmodifier.String{
								planmodifiers.SensitiveHash(),
							},
						},
					},
				},
			},
			"user_account_mappings": schema.SetNestedBlock{
				Description: "Application user account (max 1).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Application user account name.",
						},
						"description": schema.StringAttribute{
							Required:    true,
							Description: "Application user account description.",
						},
					},
				},
			},
		},
	}
}

func (r *ApplicationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that properties contains displayName
	if len(data.Properties) > 0 {
		hasDisplayName := false
		for _, prop := range data.Properties {
			if prop.Name.ValueString() == "displayName" {
				hasDisplayName = true
				break
			}
		}

		if !hasDisplayName {
			resp.Diagnostics.AddAttributeError(
				path.Root("properties"),
				"Missing Required Property",
				"The 'displayName' property is required in properties.",
			)
		}
	}

	// Validate user_account_mappings max 1
	if len(data.UserAccountMappings) > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("user_account_mappings"),
			"Too Many Mappings",
			"user_account_mappings can contain at most 1 mapping.",
		)
	}

	// Validate properties against system apps
	if err := r.validatePropertiesAgainstSystemApps(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Properties",
			fmt.Sprintf("Property validation failed: %s", err.Error()),
		)
	}
}

func (r *ApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read config to get plaintext sensitive values (plan has hashed values from SensitiveHash modifier)
	var config ApplicationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate and get app catalog details
	appCatalogDetails, err := r.getAppCatalogDetails(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Application Type",
			fmt.Sprintf("Could not validate application type: %s", err.Error()),
		)
		return
	}

	// Get application name from displayName property
	applicationName, err := r.getApplicationName(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Application Name",
			fmt.Sprintf("Could not get application name: %s", err.Error()),
		)
		return
	}

	// Create application request
	application := britive.ApplicationRequest{
		CatalogAppId:          appCatalogDetails.CatalogAppId,
		CatalogAppDisplayName: applicationName,
	}

	appResponse, err := r.client.CreateApplication(application)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Application",
			fmt.Sprintf("Could not create application: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(appResponse.AppContainerId)
	plan.CatalogAppID = types.Int64Value(int64(appResponse.CatalogAppId))

	// Use plaintext sensitive values from config for the API call; plan retains hashes for state
	planForAPI := plan
	planForAPI.SensitiveProperties = config.SensitiveProperties

	// Patch properties
	properties, err := r.buildPropertiesForAPI(ctx, &planForAPI, appResponse, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Properties",
			fmt.Sprintf("Could not build properties: %s", err.Error()),
		)
		return
	}

	_, err = r.client.PatchApplicationPropertyTypes(appResponse.AppContainerId, *properties)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Patching Application Properties",
			fmt.Sprintf("Could not patch application properties: %s", err.Error()),
		)
		return
	}

	// Configure user account mappings
	userMappings, err := r.buildUserMappings(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building User Mappings",
			fmt.Sprintf("Could not build user mappings: %s", err.Error()),
		)
		return
	}

	err = r.client.ConfigureUserMappings(appResponse.AppContainerId, *userMappings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Configuring User Mappings",
			fmt.Sprintf("Could not configure user mappings: %s", err.Error()),
		)
		return
	}

	// Create root environment group for certain app types
	allowedEnvGroupApps := map[int]string{
		2: "AWS Standalone",
		8: "Okta",
		9: "Snowflake Standalone",
	}
	if _, ok := allowedEnvGroupApps[appResponse.CatalogAppId]; ok {
		err = r.client.CreateRootEnvironmentGroup(appResponse.AppContainerId, appResponse.CatalogAppId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Root Environment Group",
				fmt.Sprintf("Could not create root environment group: %s", err.Error()),
			)
			return
		}

		rootEnvID, err := r.client.GetRootEnvID(appResponse.AppContainerId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Root Environment ID",
				fmt.Sprintf("Could not get root environment ID: %s", err.Error()),
			)
			return
		}
		plan.EntityRootEnvironmentGroupID = types.StringValue(rootEnvID)
	}

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application",
			fmt.Sprintf("Could not read application after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ID.ValueString()

	application, err := r.client.GetApplication(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application",
			fmt.Sprintf("Could not read application: %s", err.Error()),
		)
		return
	}

	// Set basic fields
	if state.ApplicationType.IsNull() {
		state.ApplicationType = types.StringValue(application.CatalogAppName)
	}
	state.CatalogAppID = types.Int64Value(int64(application.CatalogAppId))
	state.Version = types.StringValue(application.Properties.Version)

	// Handle root environment group ID for certain app types
	allowedEnvGroupApps := map[int]string{
		2: "AWS Standalone",
		8: "Okta",
		9: "Snowflake Standalone",
	}
	if _, ok := allowedEnvGroupApps[application.CatalogAppId]; ok {
		rootGroupID := ""
		if application.RootEnvironmentGroup != nil && application.RootEnvironmentGroup.EnvironmentGroups != nil {
			for _, envGroup := range application.RootEnvironmentGroup.EnvironmentGroups {
				if envGroup.Name == "root" {
					rootGroupID = envGroup.ID
					break
				}
			}
		}
		if rootGroupID != "" {
			state.EntityRootEnvironmentGroupID = types.StringValue(rootGroupID)
		}
	}

	// Map properties and sensitive properties
	if err := r.mapPropertiesFromAPI(ctx, &state, application); err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Properties",
			fmt.Sprintf("Could not map properties: %s", err.Error()),
		)
		return
	}

	// Map user account mappings - handle interface{} type
	var mappings []UserAccountMappingModel
	for _, mapping := range application.UserAccountMappings {
		m, ok := mapping.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		description, _ := m["description"].(string)
		mappings = append(mappings, UserAccountMappingModel{
			Name:        types.StringValue(name),
			Description: types.StringValue(description),
		})
	}
	state.UserAccountMappings = mappings

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	var state ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read config to get plaintext sensitive values (plan has hashed values from SensitiveHash modifier)
	var config ApplicationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ID.ValueString()

	// Validate properties against system apps
	_, err := r.getAppCatalogDetails(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Application Type",
			fmt.Sprintf("Could not validate application type: %s", err.Error()),
		)
		return
	}

	// Get current application
	application, err := r.client.GetApplication(applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application",
			fmt.Sprintf("Could not read application: %s", err.Error()),
		)
		return
	}

	// Update properties if changed
	if !propertySliceEqual(plan.Properties, state.Properties) || !propertySliceEqual(plan.SensitiveProperties, state.SensitiveProperties) {
		// Use plaintext sensitive values from config for the API call; plan retains hashes for state
		planForAPI := plan
		planForAPI.SensitiveProperties = config.SensitiveProperties

		// Build properties including removed ones
		properties, err := r.buildPropertiesForUpdate(ctx, &planForAPI, &state, application)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Building Properties",
				fmt.Sprintf("Could not build properties: %s", err.Error()),
			)
			return
		}

		_, err = r.client.PatchApplicationPropertyTypes(applicationID, *properties)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Patching Application Properties",
				fmt.Sprintf("Could not patch application properties: %s", err.Error()),
			)
			return
		}
	}

	// Update user account mappings if changed
	if !userMappingSliceEqual(plan.UserAccountMappings, state.UserAccountMappings) {
		userMappings, err := r.buildUserMappings(&plan)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Building User Mappings",
				fmt.Sprintf("Could not build user mappings: %s", err.Error()),
			)
			return
		}

		err = r.client.ConfigureUserMappings(applicationID, *userMappings)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Configuring User Mappings",
				fmt.Sprintf("Could not configure user mappings: %s", err.Error()),
			)
			return
		}
	}

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application",
			fmt.Sprintf("Could not read application after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ID.ValueString()

	err := r.client.DeleteApplication(applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Application",
			fmt.Sprintf("Could not delete application: %s", err.Error()),
		)
		return
	}
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. apps/{id}
	// 2. {id}
	idRegexes := []string{
		`^apps/(?P<id>[^/]+)$`,
		`^(?P<id>[^/]+)$`,
	}

	var applicationID string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "id" && i < len(matches) {
					applicationID = matches[i]
					break
				}
			}
			if applicationID != "" {
				break
			}
		}
	}

	if applicationID == "" || strings.TrimSpace(applicationID) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'apps/{id}' or '{id}'", req.ID),
		)
		return
	}

	// Get application
	application, err := r.client.GetApplication(applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Application",
			fmt.Sprintf("Could not import application: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_type"), application.CatalogAppName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), application.Properties.Version)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_app_id"), int64(application.CatalogAppId))...)

	// Handle root environment group ID
	allowedEnvGroupApps := map[int]string{
		2: "AWS Standalone",
		8: "Okta",
		9: "Snowflake Standalone",
	}
	if _, ok := allowedEnvGroupApps[application.CatalogAppId]; ok {
		rootGroupID := ""
		if application.RootEnvironmentGroup != nil && application.RootEnvironmentGroup.EnvironmentGroups != nil {
			for _, envGroup := range application.RootEnvironmentGroup.EnvironmentGroups {
				if envGroup.Name == "root" {
					rootGroupID = envGroup.ID
					break
				}
			}
		}
		if rootGroupID != "" {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entity_root_environment_group_id"), rootGroupID)...)
		}
	}

	// Create temporary state to map properties
	var state ApplicationResourceModel
	state.ID = types.StringValue(applicationID)
	state.ApplicationType = types.StringValue(application.CatalogAppName)

	// Map properties
	if err := r.mapPropertiesFromAPI(ctx, &state, application); err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Properties",
			fmt.Sprintf("Could not map properties during import: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("properties"), state.Properties)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sensitive_properties"), state.SensitiveProperties)...)

	// Map user account mappings - handle interface{} type
	var mappings []UserAccountMappingModel
	for _, mapping := range application.UserAccountMappings {
		m, ok := mapping.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		description, _ := m["description"].(string)
		mappings = append(mappings, UserAccountMappingModel{
			Name:        types.StringValue(name),
			Description: types.StringValue(description),
		})
	}
	if len(mappings) > 0 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_account_mappings"), mappings)...)
	}
}

// Helper functions

func (r *ApplicationResource) validatePropertiesAgainstSystemApps(ctx context.Context, data *ApplicationResourceModel) error {
	if data.ApplicationType.IsNull() {
		return fmt.Errorf("application_type is required")
	}

	_, err := r.getAppCatalogDetails(ctx, data)
	return err
}

func (r *ApplicationResource) getAppCatalogDetails(ctx context.Context, data *ApplicationResourceModel) (*britive.SystemApp, error) {
	if r.client == nil {
		// Provider not yet configured; skip validation
		return nil, nil
	}

	appType := strings.ToLower(data.ApplicationType.ValueString())

	systemApps, err := r.client.GetSystemApps()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch system apps: %w", err)
	}

	// Get latest version or use specified version
	latestVersion, allVersions := getLatestAppVersion(systemApps, appType)
	var appVersion string
	if !data.Version.IsNull() && data.Version.ValueString() != "" {
		appVersion = data.Version.ValueString()
	} else {
		appVersion = latestVersion
	}

	// Find matching system app
	var foundApp *britive.SystemApp
	for _, app := range systemApps {
		if strings.ToLower(app.Name) == appType && app.Version == appVersion {
			foundApp = &app
			break
		}
	}

	if foundApp == nil {
		return nil, fmt.Errorf("application_type '%s' with version '%s' not supported by britive. Try versions: %v", appType, appVersion, allVersions)
	}

	// Validate properties
	if len(data.Properties) > 0 {
		allowedProps := make(map[string]britive.SystemAppPropertyType)
		for _, pt := range foundApp.PropertyTypes {
			if pt.Type != "com.britive.pab.api.Secret" && pt.Type != "com.britive.pab.api.SecretFile" {
				allowedProps[pt.Name] = pt
			}
		}

		for _, prop := range data.Properties {
			name := prop.Name.ValueString()
			value := prop.Value.ValueString()

			pt, ok := allowedProps[name]
			if !ok {
				return nil, fmt.Errorf("property '%s' is not supported for application type '%s'", name, foundApp.Name)
			}

			if err := validateAppPropertyValueType(value, pt.Type, name); err != nil {
				return nil, err
			}
		}
	}

	// Validate sensitive properties
	if len(data.SensitiveProperties) > 0 {
		allowedSensitive := make(map[string]bool)
		for _, pt := range foundApp.PropertyTypes {
			if pt.Type == "com.britive.pab.api.Secret" || pt.Type == "com.britive.pab.api.SecretFile" {
				allowedSensitive[pt.Name] = true
			}
		}

		for _, prop := range data.SensitiveProperties {
			name := prop.Name.ValueString()
			if !allowedSensitive[name] {
				return nil, fmt.Errorf("sensitive property '%s' is not supported for application type '%s'", name, foundApp.Name)
			}
		}
	}

	return foundApp, nil
}

func (r *ApplicationResource) getApplicationName(data *ApplicationResourceModel) (string, error) {
	if len(data.Properties) == 0 {
		return "", fmt.Errorf("properties are required")
	}

	for _, prop := range data.Properties {
		if prop.Name.ValueString() == "displayName" {
			return prop.Value.ValueString(), nil
		}
	}

	return "", fmt.Errorf("missing mandatory property displayName")
}

func (r *ApplicationResource) buildPropertiesForAPI(ctx context.Context, plan *ApplicationResourceModel, appResponse *britive.ApplicationResponse, isUpdate bool) (*britive.Properties, error) {
	properties := &britive.Properties{}

	// Build property type map for type conversion
	propertiesMap := make(map[string]string)
	for _, property := range appResponse.Properties.PropertyTypes {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Type)
	}

	// Add regular properties
	for _, prop := range plan.Properties {
		propertyName := prop.Name.ValueString()
		propertyValue := prop.Value.ValueString()

		// Skip iconUrl
		if propertyName == "iconUrl" {
			continue
		}

		propertyType := britive.PropertyTypes{
			Name: propertyName,
		}

		// Handle boolean conversion
		if propertiesMap[propertyName] == "java.lang.Boolean" {
			boolValue, err := strconv.ParseBool(propertyValue)
			if err != nil {
				return nil, err
			}
			propertyType.Value = boolValue
		} else {
			propertyType.Value = propertyValue
		}

		properties.PropertyTypes = append(properties.PropertyTypes, propertyType)
	}

	// Add sensitive properties
	// Deduplicate sensitive properties by name
	sensitivePropertiesMap := make(map[string]string)
	for _, prop := range plan.SensitiveProperties {
		propertyName := prop.Name.ValueString()
		propertyValue := prop.Value.ValueString()

		if existingValue, ok := sensitivePropertiesMap[propertyName]; ok {
			// Check if one is hash of the other
			if isAppHashValue(existingValue, propertyValue) {
				continue
			} else if isAppHashValue(propertyValue, existingValue) {
				sensitivePropertiesMap[propertyName] = propertyValue
				continue
			} else {
				return nil, fmt.Errorf("duplicate sensitive property with different values: %s", propertyName)
			}
		} else {
			sensitivePropertiesMap[propertyName] = propertyValue
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

func (r *ApplicationResource) buildPropertiesForUpdate(ctx context.Context, plan *ApplicationResourceModel, state *ApplicationResourceModel, application *britive.ApplicationResponse) (*britive.Properties, error) {
	properties := &britive.Properties{}

	// Get system app details for defaults
	appCatalogDetails, err := r.getAppCatalogDetails(ctx, plan)
	if err != nil {
		return nil, err
	}

	// Build map of current plan properties
	newPropertyNames := make(map[string]bool)
	for _, prop := range plan.Properties {
		newPropertyNames[prop.Name.ValueString()] = true
	}
	for _, prop := range plan.SensitiveProperties {
		newPropertyNames[prop.Name.ValueString()] = true
	}

	// Build property type map from application
	propertyTypeMap := make(map[string]interface{})
	for _, foundProp := range application.Properties.PropertyTypes {
		propertyTypeMap[foundProp.Name] = map[string]interface{}{
			"value": foundProp.Value,
			"type":  foundProp.Type,
		}
	}

	// Add removed properties back to their default values
	allOldProps := append(state.Properties, state.SensitiveProperties...)
	for _, prop := range allOldProps {
		propName := prop.Name.ValueString()
		if !newPropertyNames[propName] {
			// Property was removed, reset to default
			if valueAndType, ok := propertyTypeMap[propName]; ok {
				propValue := valueAndType.(map[string]interface{})["value"]
				properties.PropertyTypes = append(properties.PropertyTypes, britive.PropertyTypes{
					Name:  propName,
					Value: propValue,
				})
			}
		}
	}

	// Add new/updated properties
	newProps, err := r.buildPropertiesForAPI(ctx, plan, application, true)
	if err != nil {
		return nil, err
	}
	properties.PropertyTypes = append(properties.PropertyTypes, newProps.PropertyTypes...)

	// Suppress unused variable warning
	_ = appCatalogDetails

	return properties, nil
}

func (r *ApplicationResource) buildUserMappings(plan *ApplicationResourceModel) (*britive.UserMappings, error) {
	userMappings := &britive.UserMappings{}

	for _, mapping := range plan.UserAccountMappings {
		userMappings.UserAccountMappings = append(userMappings.UserAccountMappings, britive.UserMapping{
			Name:        mapping.Name.ValueString(),
			Description: mapping.Description.ValueString(),
		})
	}

	return userMappings, nil
}

func (r *ApplicationResource) mapPropertiesFromAPI(ctx context.Context, state *ApplicationResourceModel, application *britive.ApplicationResponse) error {
	// Get system app details
	systemApps, err := r.client.GetSystemApps()
	if err != nil {
		return fmt.Errorf("failed to fetch system apps: %w", err)
	}

	var foundApp *britive.SystemApp
	for _, app := range systemApps {
		if app.CatalogAppId == application.CatalogAppId {
			foundApp = &app
			break
		}
	}
	if foundApp == nil {
		return fmt.Errorf("system app not found for catalog ID: %d", application.CatalogAppId)
	}

	// Build system property map
	systemPropertyTypeMap := make(map[string]map[string]interface{})
	for _, foundProp := range foundApp.PropertyTypes {
		systemPropertyTypeMap[foundProp.Name] = map[string]interface{}{
			"value": foundProp.Value,
			"type":  foundProp.Type,
		}
	}

	// Build user property map from state
	userProperties := make(map[string]interface{})
	for _, prop := range state.Properties {
		userProperties[prop.Name.ValueString()] = prop.Value.ValueString()
	}

	// Get existing sensitive properties from state
	existingSensitiveProps := make(map[string]string)
	for _, prop := range state.SensitiveProperties {
		existingSensitiveProps[prop.Name.ValueString()] = prop.Value.ValueString()
	}

	var stateProperties []PropertyModel
	var stateSensitiveProperties []PropertyModel

	for _, property := range application.Properties.PropertyTypes {
		propertyName := property.Name
		propertyValType := property.Type
		propertyValue := property.Value

		// Skip if not in user properties or is iconUrl
		if _, ok := userProperties[propertyName]; !ok || propertyName == "iconUrl" {
			if propertyValue == nil || propertyValue == "" || propertyName == "iconUrl" {
				continue
			}
		}

		// Convert value to string
		if propertyValue == nil {
			propertyValue = ""
		} else {
			propertyValue = fmt.Sprintf("%v", propertyValue)
		}

		// Handle sensitive properties
		if propertyValType == "com.britive.pab.api.Secret" || propertyValType == "com.britive.pab.api.SecretFile" {
			// If API returns "*", preserve user's hash from state
			if propertyValue == "*" {
				if existingValue, ok := existingSensitiveProps[propertyName]; ok {
					propertyValue = existingValue
				}
			}
			stateSensitiveProperties = append(stateSensitiveProperties, PropertyModel{
				Name:  types.StringValue(propertyName),
				Value: types.StringValue(propertyValue.(string)),
			})
		} else {
			// Regular property - only include if different from system default or was specified by user
			if systemPropertyTypeMap[propertyName]["value"] != property.Value {
				stateProperties = append(stateProperties, PropertyModel{
					Name:  types.StringValue(propertyName),
					Value: types.StringValue(propertyValue.(string)),
				})
			} else {
				if _, ok := userProperties[propertyName]; ok {
					stateProperties = append(stateProperties, PropertyModel{
						Name:  types.StringValue(propertyName),
						Value: types.StringValue(propertyValue.(string)),
					})
				}
			}
		}
	}

	state.Properties = stateProperties
	state.SensitiveProperties = stateSensitiveProperties

	return nil
}

func (r *ApplicationResource) populateStateFromAPI(ctx context.Context, state *ApplicationResourceModel) error {
	applicationID := state.ID.ValueString()

	application, err := r.client.GetApplication(applicationID)
	if err != nil {
		return err
	}

	if state.ApplicationType.IsNull() {
		state.ApplicationType = types.StringValue(application.CatalogAppName)
	}
	state.CatalogAppID = types.Int64Value(int64(application.CatalogAppId))
	state.Version = types.StringValue(application.Properties.Version)

	// Handle root environment group ID
	allowedEnvGroupApps := map[int]string{
		2: "AWS Standalone",
		8: "Okta",
		9: "Snowflake Standalone",
	}
	if _, ok := allowedEnvGroupApps[application.CatalogAppId]; ok {
		rootGroupID := ""
		if application.RootEnvironmentGroup != nil && application.RootEnvironmentGroup.EnvironmentGroups != nil {
			for _, envGroup := range application.RootEnvironmentGroup.EnvironmentGroups {
				if envGroup.Name == "root" {
					rootGroupID = envGroup.ID
					break
				}
			}
		}
		if rootGroupID != "" {
			state.EntityRootEnvironmentGroupID = types.StringValue(rootGroupID)
		}
	}

	// For non-eligible apps, entity_root_environment_group_id is not applicable; set to empty string.
	// Empty string (not null) is used so UseStateForUnknown plan modifier can preserve the value on subsequent plans.
	if state.EntityRootEnvironmentGroupID.IsUnknown() {
		state.EntityRootEnvironmentGroupID = types.StringValue("")
	}

	return r.mapPropertiesFromAPI(ctx, state, application)
}

// Utility functions

func getLatestAppVersion(systemApps []britive.SystemApp, appType string) (string, []string) {
	latestVersionParts := []string{"0", "0", "0", "0", "0"}
	var latestVersion string
	var allAppVersions []string

	for _, app := range systemApps {
		if strings.ToLower(app.Name) == appType {
			appVersionStr := strings.TrimPrefix(app.Version, "Custom-")
			appVersionParts := strings.Split(appVersionStr, ".")

			size := len(appVersionParts)
			if len(latestVersionParts) < size {
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

func validateAppPropertyValueType(val string, typ string, name string) error {
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
	}
	return nil
}

func getAppHash(val string) string {
	hash := argon2.IDKey([]byte(val), []byte{}, 1, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(hash)
}

func isAppHashValue(val string, hash string) bool {
	return hash == getAppHash(val)
}

// propertySliceEqual compares two PropertyModel slices for equality
func propertySliceEqual(a, b []PropertyModel) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]string, len(a))
	for _, p := range a {
		aMap[p.Name.ValueString()] = p.Value.ValueString()
	}
	for _, p := range b {
		if aMap[p.Name.ValueString()] != p.Value.ValueString() {
			return false
		}
	}
	return true
}

// userMappingSliceEqual compares two UserAccountMappingModel slices for equality
func userMappingSliceEqual(a, b []UserAccountMappingModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name.ValueString() != b[i].Name.ValueString() ||
			a[i].Description.ValueString() != b[i].Description.ValueString() {
			return false
		}
	}
	return true
}
