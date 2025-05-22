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
				Description:  "Britive application type. Suppotted types 'Snowflake', 'Snowflake Standalone'",
				ValidateFunc: validation.StringInSlice([]string{"Snowflake", "Snowflake Standalone"}, true),
			},
			"catalog_app_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Britive application base catalog id.",
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
							Required:  true,
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

	applicationName, err := rt.helper.getApplicationName(d)
	if err != nil {
		return diag.FromErr(err)
	}

	application := britive.ApplicationRequest{}
	err = rt.helper.mapApplicationResourceToModel(d, m, &application, applicationName, false)
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

func (rt *ResourceApplication) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rt.importHelper.ParseImportID([]string{"application/(?P<id>[^/]+)", "applications/(?P<id>[^/]+)", "(?P<id>[^/]+)"}, d); err != nil {
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

	err = rt.helper.importAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
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

func (rrth *ResourceApplicationHelper) mapApplicationResourceToModel(d *schema.ResourceData, m interface{}, application *britive.ApplicationRequest, applicationName string, isUpdate bool) error {
	catalogApps := map[string]int{
		"snowflake standalone": 9,
		"snowflake":            10,
	}
	catalogAppName := d.Get("application_type").(string)
	catalogAppId, ok := catalogApps[strings.ToLower(catalogAppName)]
	if !ok {
		return errors.New("Application with type %s not supportted")
	}
	application.CatalogAppId = catalogAppId
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
		if propertiesMap[propertyType.Name] == "java.lang.Boolean" {
			propertyValue, err := strconv.ParseBool(property.(map[string]interface{})["value"].(string))
			if err != nil {
				return err
			}
			propertyType.Value = propertyValue
		} else {
			propertyType.Value = property.(map[string]interface{})["value"].(string)
		}
		properties.PropertyTypes = append(properties.PropertyTypes, propertyType)
	}

	sensitivePropertiesMap := make(map[string]string)

	for _, property := range sensitiveProperties.List() {
		propertyType := britive.PropertyTypes{}
		propertyName := property.(map[string]interface{})["name"].(string)
		propertyValue := property.(map[string]interface{})["value"].(string)
		if prePropertyValue, ok := sensitivePropertiesMap[propertyName]; ok {
			if isHashValue(prePropertyValue, propertyValue) {
				continue
			}
		}
		propertyType.Name = propertyName
		propertyType.Value = propertyValue
		sensitivePropertiesMap[propertyName] = propertyValue
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

	if err := d.Set("catalog_app_id", application.CatalogAppId); err != nil {
		return err
	}

	if err := d.Set("user_account_mappings", application.UserAccountMappings); err != nil {
		return err
	}

	applicationProperties := application.Properties.PropertyTypes
	propertiesMap := make(map[string]string)
	for _, property := range applicationProperties {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Value)
	}

	var stateProperties []map[string]interface{}
	var stateSensitiveProperties []map[string]interface{}
	properties := d.Get("properties").(*schema.Set)
	sensitiveProperties := d.Get("sensitive_properties").(*schema.Set)

	for _, property := range properties.List() {
		propertyName := property.(map[string]interface{})["name"].(string)
		stateProperties = append(stateProperties, map[string]interface{}{
			"name":  propertyName,
			"value": propertiesMap[propertyName],
		})
	}
	for _, property := range sensitiveProperties.List() {
		propertyName := property.(map[string]interface{})["name"].(string)
		if propertiesMap[propertyName] == "*" {
			for _, sp := range sensitiveProperties.List() {
				existing := sp.(map[string]interface{})
				if existing["name"] == propertyName {
					propertiesMap[propertyName] = existing["value"].(string)
					break
				}
			}
		}
		stateSensitiveProperties = append(stateSensitiveProperties, map[string]interface{}{
			"name":  propertyName,
			"value": propertiesMap[propertyName],
		})
	}

	if err := d.Set("properties", stateProperties); err != nil {
		return err
	}
	if err := d.Set("sensitive_properties", stateSensitiveProperties); err != nil {
		return err
	}
	return nil
}

func (rrth *ResourceApplicationHelper) importAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	applicationID, err := rrth.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Importing application %s", applicationID)

	application, err := c.GetApplication(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("application %s", applicationID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received application %#v", application)

	catalogApps := map[int]string{
		9:  "snowflake standalone",
		10: "snowflake",
	}
	catalogAppId := application.CatalogAppId
	catalogAppType, ok := catalogApps[catalogAppId]
	if !ok {
		return errors.New(fmt.Sprintf("Catlog with id %d not supportted", catalogAppId))
	}
	if err := d.Set("catalog_app_id", catalogAppId); err != nil {
		return err
	}

	if err := d.Set("application_type", catalogAppType); err != nil {
		return err
	}

	var stateProperties []map[string]interface{}
	var stateSensitiveProperties []map[string]interface{}
	applicationProperties := application.Properties.PropertyTypes
	for _, property := range applicationProperties {
		propertyName := property.Name

		if property.Type == "com.britive.pab.api.Secret" || property.Type == "com.britive.pab.api.SecretFile" {
			stateSensitiveProperties = append(stateSensitiveProperties, map[string]interface{}{
				"name":  propertyName,
				"value": fmt.Sprintf("%v", property.Value),
			})
		} else {
			stateProperties = append(stateProperties, map[string]interface{}{
				"name":  propertyName,
				"value": fmt.Sprintf("%v", property.Value),
			})
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
