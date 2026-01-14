package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceEntityEnvironment - Terraform Resource for Application Entity Environment
type ResourceEntityEnvironment struct {
	Resource     *schema.Resource
	helper       *ResourceEntityEnvironmentHelper
	importHelper *imports.ImportHelper
}

// NewResourceEntityEnvironment - Initialization of new application entity environment resource
func NewResourceEntityEnvironment(importHelper *imports.ImportHelper) *ResourceEntityEnvironment {
	ree := &ResourceEntityEnvironment{
		helper:       NewResourceEntityEnvironmentHelper(),
		importHelper: importHelper,
	}
	ree.Resource = &schema.Resource{
		CreateContext: ree.resourceCreate,
		ReadContext:   ree.resourceRead,
		UpdateContext: ree.resourceUpdate,
		DeleteContext: ree.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: ree.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"entity_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The identity of the application entity of type environment",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"application_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identity of the Britive application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"parent_group_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The parent group id under which the environment will be created",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"properties": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Britive application entity environment overwrite properties.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Britive application entity environment property name.",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Britive application entity environment property value.",
						},
					},
				},
			},
			"sensitive_properties": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Britive application entity environment overwrite sensitive properties.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							StateFunc: func(val interface{}) string {
								return val.(string)
							},
							Description: "Britive application entity environment property name.",
						},
						"value": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
							StateFunc: func(val interface{}) string {
								return getHash(val.(string))
							},
							Description: "Britive application entity environment property value.",
						},
					},
				},
			},
		},
	}
	return ree
}

//region Application Entity Resource Context Operations

func (ree *ResourceEntityEnvironment) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client
	var diags diag.Diagnostics

	applicationEntity := britive.ApplicationEntityEnvironment{}

	err := ree.helper.mapResourceToModel(d, m, &applicationEntity, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new application entity environment: %#v", applicationEntity)

	applicationID := d.Get("application_id").(string)

	ae, err := c.CreateEntityEnvironment(applicationEntity, applicationID, m)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new application entity environment: %#v", ae)
	d.SetId(ree.helper.generateUniqueID(applicationID, ae.EntityID))

	// Get application environment for entity with type environment
	appEnvDetails, err := c.GetApplicationEnvironment(applicationID, ae.EntityID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Patch properties
	properties := britive.Properties{}
	err = ree.helper.mapPropertiesResourceToModel(d, m, &properties, appEnvDetails)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating application environment properties")
	_, err = c.PatchApplicationEnvPropertyTypes(applicationID, ae.EntityID, properties, m)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Updated application environment properties")

	ree.resourceRead(ctx, d, m)
	return diags
}

func (ree *ResourceEntityEnvironment) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := ree.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	err = ree.helper.getAndMapPropertiesModelToResource(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (ree *ResourceEntityEnvironment) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var hasChanges bool
	if d.HasChange("properties") || d.HasChange("sensitive_properties") {
		hasChanges = true
		applicationID, entityID, err := ree.helper.parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		// Get application Environment for entity with type Environment
		appEnvDetails, err := c.GetApplicationEnvironment(applicationID, entityID)
		if err != nil {
			return diag.FromErr(err)
		}

		// Patch properties
		properties := britive.Properties{}
		err = ree.helper.mapPropertiesResourceToModel(d, m, &properties, appEnvDetails)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating application entity environment properties")
		_, err = c.PatchApplicationEnvPropertyTypes(applicationID, entityID, properties, m)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updated application entity environment properties")
	}
	if hasChanges {
		return ree.resourceRead(ctx, d, m)
	}
	return nil
}

func (ree *ResourceEntityEnvironment) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	applicationID, entityID, err := ree.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting entity %s of type environment for application %s", entityID, applicationID)
	err = c.DeleteEntityEnvironment(applicationID, entityID, m)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleted entity %s of type environment for application %s", entityID, applicationID)
	d.SetId("")

	return diags
}

func (ree *ResourceEntityEnvironment) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client
	var err error
	if err := ree.importHelper.ParseImportID([]string{"apps/(?P<application_id>[^/]+)/root-environment-group/environments/(?P<entity_id>[^/]+)", "(?P<application_id>[^/]+)/environments/(?P<entity_id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	applicationID := d.Get("application_id").(string)
	entityID := d.Get("entity_id").(string)
	if strings.TrimSpace(applicationID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("application_id")
	}
	if strings.TrimSpace(entityID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("entity_id")
	}

	log.Printf("[INFO] Importing entity %s of type environment for application %s", entityID, applicationID)

	appEnvs, err := c.GetAppEnvs(applicationID, "environments")
	if err != nil {
		return nil, err
	}
	envIdList, err := c.GetEnvDetails(appEnvs, "id")
	if err != nil {
		return nil, err
	}

	for _, id := range envIdList {
		if id == entityID {
			d.SetId(ree.helper.generateUniqueID(applicationID, entityID))

			err = ree.helper.getAndMapModelToResource(d, m)
			if err != nil {
				return nil, err
			}

			err = ree.helper.importAndMapPropertiesToResource(d, m)
			if err != nil {
				return nil, err
			}
			log.Printf("[INFO] Imported entity %s of type environment for application %s", entityID, applicationID)
			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, errs.NewNotFoundErrorf("entity %s of type environment for application %s", entityID, applicationID)

}

//endregion

// ResourceEntityEnvironmentHelper - Terraform Resource for Application Entity Environment Helper
type ResourceEntityEnvironmentHelper struct {
}

// NewResourceEntityEnvironmentHelper - Initialization of new application entity environment resource helper
func NewResourceEntityEnvironmentHelper() *ResourceEntityEnvironmentHelper {
	return &ResourceEntityEnvironmentHelper{}
}

//region ApplicationEntityEnvironment Resource helper functions

func (reeh *ResourceEntityEnvironmentHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, applicationEntity *britive.ApplicationEntityEnvironment, isUpdate bool) error {
	entityName, err := reeh.getPropertyByKey(d, "displayName")
	if err != nil {
		return err
	}
	description, err := reeh.getPropertyByKey(d, "description")
	if err != nil {
		return err
	}
	applicationEntity.Name = entityName
	applicationEntity.Description = description
	applicationEntity.ParentGroupID = d.Get("parent_group_id").(string)
	applicationEntity.EntityID = d.Get("entity_id").(string)
	return nil
}

func (reeh *ResourceEntityEnvironmentHelper) getPropertyByKey(d *schema.ResourceData, key string) (string, error) {
	propertyTypes := d.Get("properties").(*schema.Set)
	for _, property := range propertyTypes.List() {
		propertyName := property.(map[string]interface{})["name"].(string)
		propertyValue := property.(map[string]interface{})["value"].(string)
		if propertyName == key {
			return propertyValue, nil
		}
	}
	return "", errors.New("Missing mandatory property " + key)
}

func (reeh *ResourceEntityEnvironmentHelper) mapPropertiesResourceToModel(d *schema.ResourceData, m interface{}, properties *britive.Properties, appResponse *britive.ApplicationResponse) error {
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

func (reeh *ResourceEntityEnvironmentHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	applicationID, entityID, err := reeh.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading entity environment %s for application %s", entityID, applicationID)

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(applicationID, m)
	if err != nil || appRootEnvironmentGroup == nil {
		return err
	}

	for _, association := range appRootEnvironmentGroup.Environments {
		if association.ID == entityID {
			log.Printf("[INFO] Received entity environment: %#v", association)
			if err := d.Set("entity_id", entityID); err != nil {
				return err
			}
			if err := d.Set("parent_group_id", association.ParentGroupID); err != nil {
				return err
			}
			return nil
		}
	}
	return errs.NewNotFoundErrorf("entity environment %s for application %s", entityID, applicationID)
}

func (reeh *ResourceEntityEnvironmentHelper) getAndMapPropertiesModelToResource(d *schema.ResourceData, m interface{}) error {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	applicationID, entityID, err := reeh.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	applicationEnvironmentDetails, err := c.GetApplicationEnvironment(applicationID, entityID)
	if err != nil {
		return err
	}

	applicationProperties := applicationEnvironmentDetails.Properties.PropertyTypes
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

func (rrth *ResourceEntityEnvironmentHelper) importAndMapPropertiesToResource(d *schema.ResourceData, m interface{}) error {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	applicationID, entityID, err := rrth.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Importing properties for entity %s for application %s", entityID, applicationID)

	// Get application Environment for entity with type Environment
	appEnvDetails, err := c.GetApplicationEnvironment(applicationID, entityID)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received entity env %#v", appEnvDetails)
	//ToDo: check env type should be envrionment

	var stateProperties []map[string]interface{}
	var stateSensitiveProperties []map[string]interface{}
	applicationProperties := appEnvDetails.Properties.PropertyTypes
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

//endregion
