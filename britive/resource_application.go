package britive

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/crypto/argon2"
)

// ResourceApplication - Terraform Resource for Application
type ResourceApplication struct {
	Resource     *schema.Resource
	helper       *ResourceApplicationHelper
	validation   *Validation
	importHelper *ImportHelper
}

// NewResourceApplication - Initializes new application resource
func NewResourceApplication(v *Validation, importHelper *ImportHelper) *ResourceApplication {
	rt := &ResourceApplication{
		helper:       NewResourceApplicationHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rt.Resource = &schema.Resource{
		CreateContext: rt.resourceCreate,
		ReadContext:   rt.resourceRead,
		UpdateContext: rt.resourceUpdate,
		DeleteContext: rt.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rt.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"application_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Britive application type. Suppotted types 'Snowflake', 'Snowflake Standalone', 'GCP', 'GCP Standalone' and 'Google Workspace'",
				ValidateFunc: validation.StringInSlice([]string{"Snowflake", "Snowflake Standalone", "GCP", "GCP Standalone", "Google Workspace"}, true),
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Britive application version",
			},
			"catalog_app_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Britive application base catalog id.",
			},
			"entity_root_environment_group_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Britive application root environment ID for Snowflake Standalone applications.",
			},
			"properties": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Britive application overwrite properties.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Britive application property name.",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Britive application property value.",
						},
					},
				},
			},
			"sensitive_properties": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Britive application overwrite sensitive properties.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							StateFunc: func(val interface{}) string {
								return val.(string)
							},
							Description: "Britive application property name.",
						},
						"value": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
							StateFunc: func(val interface{}) string {
								return getHash(val.(string))
							},
							Description: "Britive application property value.",
						},
					},
				},
			},
			"user_account_mappings": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Application user account",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Application user account name",
						},
						"description": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Application user account description",
						},
					},
				},
			},
		},
	}
	return rt
}

func (rt *ResourceApplication) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	// Validate properties and sensitive_properties
	err, appCatalogDetails := rt.helper.validatePropertiesAgainstSystemApps(d, c)
	if err != nil {
		return diag.FromErr(err)
	}

	applicationName, err := rt.helper.getApplicationName(d)
	if err != nil {
		return diag.FromErr(err)
	}

	application := britive.ApplicationRequest{}
	err = rt.helper.mapApplicationResourceToModel(d, m, &application, applicationName, appCatalogDetails, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new application")

	appResponse, err := c.CreateApplication(application)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new application: %#v", appResponse)

	d.SetId(rt.helper.generateUniqueID(appResponse.AppContainerId))

	log.Printf("[INFO] Updated state after application submission: %#v", appResponse)

	// Patch properties
	properties := britive.Properties{}
	err = rt.helper.mapPropertiesResourceToModel(d, m, &properties, appResponse, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating application properties")
	_, err = c.PatchApplicationPropertyTypes(appResponse.AppContainerId, properties)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Updated application properties")

	// configer user account mappings
	userMappings := britive.UserMappings{}
	err = rt.helper.mapUserMappingsResourceToModel(d, m, &userMappings, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating user mappings: %#v", userMappings)
	err = c.ConfigureUserMappings(appResponse.AppContainerId, userMappings)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Updated user mappings: %#v", userMappings)

	//The root environment group creation can be skipped when PAB-20648 is fixed
	if application.CatalogAppId == 9 {
		log.Printf("[INFO] Creating root environment group")
		err = c.CreateRootEnvironmentGroup(appResponse.AppContainerId, application.CatalogAppId)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Created root environment group")
		rootEnvID, err := c.GetRootEnvID(appResponse.AppContainerId)
		if err != nil {
			return diag.FromErr(err)
		}
		if err = d.Set("entity_root_environment_group_id", rootEnvID); err != nil {
			return diag.FromErr(err)
		}
	}

	rt.resourceRead(ctx, d, m)

	return diags
}

func (rt *ResourceApplication) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	err := rt.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rt *ResourceApplication) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	// Validate properties and sensitive_properties
	err, foundApp := rt.helper.validatePropertiesAgainstSystemApps(d, c)
	if err != nil {
		return diag.FromErr(err)
	}

	applicationID, err := rt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var hasChanges bool
	if d.HasChange("properties") || d.HasChange("sensitive_properties") {
		hasChanges = true

		log.Printf("[INFO] Reading application %s", applicationID)
		application, err := c.GetApplication(applicationID)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(NewNotFoundErrorf("application %s", applicationID))
		}
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Received application %#v", application)

		properties := britive.Properties{}

		oldProps, newProps := d.GetChange("properties")
		oldSprops, newSprops := d.GetChange("sensitive_properties")

		getRemovedProperties(c, foundApp, &properties, oldProps, newProps, oldSprops, newSprops)
		err = rt.helper.mapPropertiesResourceToModel(d, m, &properties, application, false)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating application properties")
		_, err = c.PatchApplicationPropertyTypes(applicationID, properties)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updated application properties")

	}
	if d.HasChange("user_account_mappings") {
		hasChanges = true
		userMappings := britive.UserMappings{}
		err = rt.helper.mapUserMappingsResourceToModel(d, m, &userMappings, false)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating user mappings: %#v", userMappings)
		err = c.ConfigureUserMappings(applicationID, userMappings)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updated user mappings: %#v", userMappings)
	}

	if hasChanges {
		return rt.resourceRead(ctx, d, m)
	}
	return nil
}

func (rt *ResourceApplication) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	applicationID, err := rt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting application %s", applicationID)
	err = c.DeleteApplication(applicationID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Application %s deleted", applicationID)
	d.SetId("")

	return diags
}

func getRemovedProperties(c *britive.Client, application *britive.SystemApp, properties *britive.Properties, oldProps, newProps, oldSprops, newSprops interface{}) {
	var oldPropertiesList, newPropertiesList, oldSecPropertiesList, newSecPropertiesList []interface{}

	oldPropertiesList = oldProps.(*schema.Set).List()
	newPropertiesList = newProps.(*schema.Set).List()
	oldSecPropertiesList = oldSprops.(*schema.Set).List()
	newSecPropertiesList = newSprops.(*schema.Set).List()

	oldPropertiesList = append(oldPropertiesList, oldSecPropertiesList...)
	newPropertiesList = append(newPropertiesList, newSecPropertiesList...)

	newPropertyNames := make(map[string]interface{})
	for _, item := range newPropertiesList {
		if prop, ok := item.(map[string]interface{}); ok {
			if name, ok := prop["name"].(string); ok {
				newPropertyNames[name] = nil
			}
		}
	}

	propertyTypeMap := make(map[string]interface{})
	for _, foundProp := range application.PropertyTypes {
		prop := map[string]interface{}{
			"value": foundProp.Value,
			"type":  foundProp.Type,
		}
		propertyTypeMap[foundProp.Name] = prop
	}

	for _, propRaw := range oldPropertiesList {
		prop, _ := propRaw.(map[string]interface{})

		propName := prop["name"].(string)

		if _, found := newPropertyNames[propName]; !found {
			var property britive.PropertyTypes
			property.Name = propName

			valueAndType, _ := propertyTypeMap[propName]
			propValue := valueAndType.(map[string]interface{})["value"]
			property.Value = propValue
			properties.PropertyTypes = append(properties.PropertyTypes, property)
		}
	}
}

func (rt *ResourceApplication) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rt.importHelper.ParseImportID([]string{"apps/(?P<id>[^/]+)", "(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	applicationID := d.Id()
	if strings.TrimSpace(applicationID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("id")
	}

	log.Printf("[INFO] Importing resource type: %s", applicationID)

	application, err := c.GetApplication(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("application %s", applicationID)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rt.helper.generateUniqueID(application.AppContainerId))

	log.Printf("[INFO] Imported application: %s", applicationID)
	return []*schema.ResourceData{d}, nil
}

// ResourceApplicationHelper - Resource Resource Type helper functions
type ResourceApplicationHelper struct {
}

// NewResourceApplicationHelper - Initializes new resource type resource helper
func NewResourceApplicationHelper() *ResourceApplicationHelper {
	return &ResourceApplicationHelper{}
}

func (rrth *ResourceApplicationHelper) mapApplicationResourceToModel(d *schema.ResourceData, m interface{}, application *britive.ApplicationRequest, applicationName string, appCatalogDetails *britive.SystemApp, isUpdate bool) error {
	application.CatalogAppId = appCatalogDetails.CatalogAppId
	application.CatalogAppDisplayName = applicationName
	return nil
}

func (rrth *ResourceApplicationHelper) getApplicationName(d *schema.ResourceData) (string, error) {
	propertyTypes := d.Get("properties").(*schema.Set)
	for _, property := range propertyTypes.List() {
		propertyName := property.(map[string]interface{})["name"].(string)
		propertyValue := property.(map[string]interface{})["value"].(string)
		if propertyName == "displayName" {
			return propertyValue, nil
		}
	}
	return "", errors.New("Missing mandatory property displayName")
}

func getHash(val string) string {
	hash := argon2.IDKey([]byte(val), []byte{}, 1, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(hash)
}

func isHashValue(val string, hash string) bool {
	return hash == getHash(val)
}

func (rrth *ResourceApplicationHelper) mapPropertiesResourceToModel(d *schema.ResourceData, m interface{}, properties *britive.Properties, appResponse *britive.ApplicationResponse, isUpdate bool) error {
	propertyTypes := d.Get("properties").(*schema.Set)
	sensitiveProperties := d.Get("sensitive_properties").(*schema.Set)

	applicationProperties := appResponse.Properties.PropertyTypes
	propertiesMap := make(map[string]string)
	for _, property := range applicationProperties {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Type)
	}

	for _, property := range propertyTypes.List() {
		propertyType := britive.PropertyTypes{}
		propertyType.Name = property.(map[string]interface{})["name"].(string)

		if propertyType.Name == "iconUrl" {
			continue
		}

		if propertiesMap[propertyType.Name] == "java.lang.Boolean" {
			propertyValue, err := strconv.ParseBool(property.(map[string]interface{})["value"].(string))
			if err != nil {
				return err
			}
			propertyType.Value = propertyValue
		} else {
			propertyValue := property.(map[string]interface{})["value"].(string)
			propertyType.Value = propertyValue
		}
		properties.PropertyTypes = append(properties.PropertyTypes, propertyType)
	}

	sensitivePropertiesMap := make(map[string]string)

	for _, property := range sensitiveProperties.List() {
		propertyName := property.(map[string]interface{})["name"].(string)
		propertyValue := property.(map[string]interface{})["value"].(string)
		if prePropertyValue, ok := sensitivePropertiesMap[propertyName]; ok {
			if isHashValue(prePropertyValue, propertyValue) {
				continue
			} else if isHashValue(propertyValue, prePropertyValue) {
				sensitivePropertiesMap[propertyName] = propertyValue
				continue
			} else {
				return errors.New("Something wrong with sensitive properties")
			}
		} else {
			sensitivePropertiesMap[propertyName] = propertyValue
		}
	}

	for sensitivePropertyName, sensitivePropertyValue := range sensitivePropertiesMap {
		propertyType := britive.PropertyTypes{}
		propertyType.Name = sensitivePropertyName
		propertyType.Value = sensitivePropertyValue
		properties.PropertyTypes = append(properties.PropertyTypes, propertyType)
	}

	return nil
}

func (rrth *ResourceApplicationHelper) mapUserMappingsResourceToModel(d *schema.ResourceData, m interface{}, userMappings *britive.UserMappings, isUpdate bool) error {
	inputUserMappings := d.Get("user_account_mappings").(*schema.Set)
	for _, user := range inputUserMappings.List() {
		userMapping := britive.UserMapping{}
		userMapping.Name = user.(map[string]interface{})["name"].(string)
		userMapping.Description = user.(map[string]interface{})["description"].(string)
		userMappings.UserAccountMappings = append(userMappings.UserAccountMappings, userMapping)
	}
	return nil
}

func (rrth *ResourceApplicationHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	applicationID, err := rrth.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading application %s", applicationID)

	application, err := c.GetApplication(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("application %s", applicationID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received application %#v", application)

	if _, ok := d.GetOk("application_type"); !ok {
		if err := d.Set("application_type", application.CatalogAppName); err != nil {
			return err
		}
	}

	if err := d.Set("catalog_app_id", application.CatalogAppId); err != nil {
		return err
	}

	if err := d.Set("user_account_mappings", application.UserAccountMappings); err != nil {
		return err
	}

	if err := d.Set("version", application.Properties.Version); err != nil {
		return err
	}

	systemApps, err := c.GetSystemApps()
	if err != nil {
		return fmt.Errorf("Failed to fetch system apps: %v", err)
	}

	var foundApp *britive.SystemApp
	for _, app := range systemApps {
		if app.CatalogAppId == application.CatalogAppId {
			foundApp = &app
			break
		}
	}
	if foundApp == nil {
		return fmt.Errorf("Failed to found the system app with catalog ID: %v", application.CatalogAppId)
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
	sensitiveProperties := d.Get("sensitive_properties").(*schema.Set)
	properties := d.Get("properties").(*schema.Set)
	userProperties := make(map[string]interface{})
	for _, prop := range properties.List() {
		propName := prop.(map[string]interface{})["name"].(string)
		propValue := prop.(map[string]interface{})["value"]
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
				for _, sp := range sensitiveProperties.List() {
					existing := sp.(map[string]interface{})
					if existing["name"] == propertyName {
						propertyValue = existing["value"].(string)
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
	if err := d.Set("properties", stateProperties); err != nil {
		return err
	}
	if err := d.Set("sensitive_properties", stateSensitiveProperties); err != nil {
		return err
	}
	return nil
}

func (resourceApplicationHelper *ResourceApplicationHelper) generateUniqueID(applicationID string) string {
	return applicationID
}

func (resourceApplicationHelper *ResourceApplicationHelper) parseUniqueID(ID string) (applicationID string, err error) {
	return ID, nil
}

// validatePropertiesAgainstSystemApps validates properties and sensitive_properties against system apps
func (rrth *ResourceApplicationHelper) validatePropertiesAgainstSystemApps(d *schema.ResourceData, c *britive.Client) (error, *britive.SystemApp) {
	appTypeRaw, ok := d.GetOk("application_type")
	if !ok {
		// Return error
		return fmt.Errorf("Required application_type is not provided"), nil
	}
	appType := strings.ToLower(appTypeRaw.(string))

	systemApps, err := c.GetSystemApps()
	if err != nil {
		return fmt.Errorf("Failed to fetch system apps: %v", err), nil
	}

	latestVersion, allAppVersions := getLatestVersion(systemApps, appType)

	var appVersion string
	appVersionRaw, ok := d.GetOk("version")
	if ok {
		appVersion = appVersionRaw.(string)
	} else {
		log.Printf("Selected latest version %s for application with type %s", latestVersion, appTypeRaw)
		appVersion = latestVersion
	}

	var foundApp *britive.SystemApp
	for _, app := range systemApps {
		if strings.ToLower(app.Name) == appType && app.Version == appVersion {
			foundApp = &app
			break
		}
	}
	if foundApp == nil {
		return fmt.Errorf("application_type '%s' with version '%s' not supportted by britive. \nTry %v versions", appType, appVersion, allAppVersions), nil
	}
	log.Printf("[Info] Selecting catalog with id %d for application of type %s.", foundApp.CatalogAppId, appTypeRaw)

	allowedProps := map[string]britive.SystemAppPropertyType{}
	allowedSensitive := map[string]britive.SystemAppPropertyType{}
	for _, pt := range foundApp.PropertyTypes {
		if pt.Type == "com.britive.pab.api.Secret" || pt.Type == "com.britive.pab.api.SecretFile" {
			allowedSensitive[pt.Name] = pt
		} else {
			allowedProps[pt.Name] = pt
		}
	}
	// Validate properties
	props := d.Get("properties").(*schema.Set)
	userProps := map[string]bool{}
	for _, prop := range props.List() {
		propMap := prop.(map[string]interface{})
		name := propMap["name"].(string)
		val := propMap["value"].(string)
		userProps[name] = true
		pt, ok := allowedProps[name]
		if !ok {
			return fmt.Errorf("Property '%s' is not supported for application type '%s'", name, foundApp.Name), nil
		}
		// Type validation for non-sensitive properties
		if err := validatePropertyValueType(val, pt.Type, name); err != nil {
			return err, nil
		}
	}
	// Validate sensitive_properties
	sprops := d.Get("sensitive_properties").(*schema.Set)
	userSensitive := map[string]bool{}
	for _, prop := range sprops.List() {
		propMap := prop.(map[string]interface{})
		name := propMap["name"].(string)
		userSensitive[name] = true
		if _, ok := allowedSensitive[name]; !ok {
			return fmt.Errorf("sensitive property '%s' is not supported for application type '%s'", name, foundApp.Name), nil
		}
	}
	return nil, foundApp
}

func getLatestVersion(systemApps []britive.SystemApp, appType string) (string, []string) {
	var foundApps []britive.SystemApp
	latestVersionParts := []string{"0", "0", "0", "0", "0"}
	var latestVersion string
	var allAppVersions []string
	for _, app := range systemApps {
		if strings.ToLower(app.Name) == appType {
			foundApps = append(foundApps, app)
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

func validatePropertyValueType(val string, typ string, name string) error {
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
