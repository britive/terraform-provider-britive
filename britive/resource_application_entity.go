package britive

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceApplicationEntity - Terraform Resource for Application Entity
type ResourceApplicationEntity struct {
	Resource     *schema.Resource
	helper       *ResourceApplicationEntityHelper
	importHelper *ImportHelper
}

// NewResourceApplicationEntity - Initialization of new application entity resource
func NewResourceApplicationEntity(importHelper *ImportHelper) *ResourceApplicationEntity {
	rae := &ResourceApplicationEntity{
		helper:       NewResourceApplicationEntityHelper(),
		importHelper: importHelper,
	}
	rae.Resource = &schema.Resource{
		CreateContext: rae.resourceCreate,
		ReadContext:   rae.resourceRead,
		UpdateContext: rae.resourceUpdate,
		DeleteContext: rae.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rae.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"entity_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The identity of the Britive application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"application_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identity of the Britive application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			// "entity_type": {
			// 	Type:         schema.TypeString,
			// 	Required:     true,
			// 	ForceNew:     true,
			// 	Description:  "The entity type, should be one of [Environment, EnvironmentGroup]",
			// 	ValidateFunc: validation.StringInSlice([]string{"Environment", "EnvironmentGroup"}, true),
			// },
			// "entity_name": {
			// 	Type:         schema.TypeString,
			// 	Required:     true,
			// 	Description:  "The name of the entity",
			// 	ValidateFunc: validation.StringIsNotWhiteSpace,
			// },
			// "entity_description": {
			// 	Type:        schema.TypeString,
			// 	Optional:    true,
			// 	Default:     "",
			// 	Description: "The description of the entity",
			// },
			// "parent_id": {
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	ForceNew:     true,
			// 	Description:  "The parent id under which the environment group will be created",
			// 	ValidateFunc: validation.StringIsNotWhiteSpace,
			// },
			"parent_group_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The parent id under which the environment will be created",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
		},
		// CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
		// 	// envSet := strings.EqualFold("Environment", d.Get("entity_type").(string))
		// 	envGroupSet := strings.EqualFold("EnvironmentGroup", d.Get("entity_type").(string))
		// 	parentSet := d.Get("parent_id").(string) != ""
		// 	parentGroupSet := d.Get("parent_group_id").(string) != ""
		// 	if envGroupSet && parentGroupSet {
		// 		return fmt.Errorf("Use only `parent_id` when creating an entity of type EnvironmentGroup. Do not use `parent_group_id`.")
		// 	}
		// 	if envSet && parentSet {
		// 		return fmt.Errorf("Use only `parent_group_id` when creating an entity of type Environment. Do not use `parent_id`.")
		// 	}
		// 	return nil
		// },
	}
	return rae
}

//region Application Entity Resource Context Operations

func (rae *ResourceApplicationEntity) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	applicationEntity := britive.ApplicationEntity{}

	err := rae.helper.mapResourceToModel(d, m, &applicationEntity, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new application entity: %#v", applicationEntity)

	applicationID := d.Get("application_id").(string)

	ae, err := c.CreateApplicationEntity(applicationEntity, applicationID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new application entity: %#v", ae)
	d.SetId(rae.helper.generateUniqueID(applicationID, ae.Type, ae.EntityID))

	// Get application Environment for entity with type Environment
	appEnvDetails, err := c.GetApplicationEnvironment(applicationID, ae.EntityID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Patch properties
	properties := britive.Properties{}
	err = rae.helper.mapPropertiesResourceToModel(d, m, &properties, appEnvDetails)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating application environment properties: %+v", properties)
	_, err = c.PatchApplicationEnvPropertyTypes(applicationID, ae.EntityID, properties)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Updated application environment properties")

	rae.resourceRead(ctx, d, m)
	return diags
}

func (rae *ResourceApplicationEntity) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rae.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	err = rae.helper.getAndMapPropertiesModelToResource(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rae *ResourceApplicationEntity) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var hasChanges bool
	if d.HasChange("properties") || d.HasChange("sensitive_properties") {
		hasChanges = true
		applicationID, _, entityID, err := rae.helper.parseUniqueID(d.Id())
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
		err = rae.helper.mapPropertiesResourceToModel(d, m, &properties, appEnvDetails)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating application environment properties: %+v", properties)
		_, err = c.PatchApplicationEnvPropertyTypes(applicationID, ntityID, properties)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updated application environment properties")
	}
	if hasChanges {
		return rae.resourceRead(ctx, d, m)
	}
	return nil
}

func (rae *ResourceApplicationEntity) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	applicationID, entityType, entityID, err := rae.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting entity %s for application %s", entityID, applicationID)
	err = c.DeleteApplicationEntity(applicationID, entityType, entityID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleted entity %s for application %s", entityID, applicationID)
	d.SetId("")

	return diags
}

func (rae *ResourceApplicationEntity) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	var err error
	if err := rae.importHelper.ParseImportID([]string{"apps/(?P<application_id>[^/]+)/root-environment-group/(?P<entity_type>[^/]+)/(?P<entity_id>[^/]+)", "(?P<application_id>[^/]+)/(?P<entity_type>[^/]+)/(?P<entity_id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	applicationID := d.Get("application_id").(string)
	entityID := d.Get("entity_id").(string)
	if strings.TrimSpace(applicationID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("application_id")
	}
	if strings.TrimSpace(entityID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("entity_id")
	}

	appEnvs := make([]britive.ApplicationEnvironment, 0)

	log.Printf("[INFO] Importing entity %s of type %s for application %s", entityID, "Environment", applicationID)

	envIdList, err := c.GetEnvDetails(appEnvs, "id")
	if err != nil {
		return nil, err
	}

	for _, id := range envIdList {
		if id == entityID {
			d.SetId(rae.helper.generateUniqueID(applicationID, "Environment", entityID))

			err = rae.helper.getAndMapModelToResource(d, m)
			if err != nil {
				return nil, err
			}
			log.Printf("[INFO] Imported entity %s for application %s", entityID, applicationID)
			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, NewNotFoundErrorf("entity %s for application %s", entityID, applicationID)

}

//endregion

// ResourceApplicationEntityHelper - Terraform Resource for Application Entity Helper
type ResourceApplicationEntityHelper struct {
}

// NewResourceApplicationEntityHelper - Initialization of new application entity resource helper
func NewResourceApplicationEntityHelper() *ResourceApplicationEntityHelper {
	return &ResourceApplicationEntityHelper{}
}

//region ApplicationEntity Resource helper functions

func (raeh *ResourceApplicationEntityHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, applicationEntity *britive.ApplicationEntity, isUpdate bool) error {
	entityName, err := raeh.getPropertyByKey(d, "displayName")
	if err != nil {
		return err
	}
	description, err := raeh.getPropertyByKey(d, "description")
	if err != nil {
		return err
	}
	applicationEntity.Name = entityName
	applicationEntity.Type = "Environment"
	applicationEntity.Description = description
	// applicationEntity.ParentID = d.Get("parent_id").(string)
	applicationEntity.ParentGroupID = d.Get("parent_group_id").(string)
	applicationEntity.EntityID = d.Get("entity_id").(string)
	return nil
}

func (raeh *ResourceApplicationEntityHelper) getPropertyByKey(d *schema.ResourceData, key string) (string, error) {
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

func (raeh *ResourceApplicationEntityHelper) mapPropertiesResourceToModel(d *schema.ResourceData, m interface{}, properties *britive.Properties, appResponse *britive.ApplicationResponse) error {
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
				log.Printf("Hash====== Key %s prePropertyValue %s propertyValue %s", propertyName, prePropertyValue, propertyValue)
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

func (raeh *ResourceApplicationEntityHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	applicationID, _, entityID, err := raeh.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading entity %s for application %s", entityID, applicationID)

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil || appRootEnvironmentGroup == nil {
		return err
	}

	for _, association := range appRootEnvironmentGroup.Environments {
		if association.ID == entityID {
			log.Printf("[INFO] Received entity: %#v", association)
			if err := d.Set("entity_id", entityID); err != nil {
				return err
			}
			if err := d.Set("parent_group_id", association.ParentGroupID); err != nil {
				return err
			}
			return nil
		}
	}
	return NewNotFoundErrorf("entity %s for application %s", entityID, applicationID)
}

func (raeh *ResourceApplicationEntityHelper) getAndMapPropertiesModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	applicationID, entityType, entityID, err := raeh.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	if strings.EqualFold(entityType, "Environment") {
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
	}
	return nil
}

func (resourceApplicationEntityHelper *ResourceApplicationEntityHelper) generateUniqueID(applicationID, entityType, entityID string) string {
	return fmt.Sprintf("apps/%s/root-environment-group/%s/%s", applicationID, entityType, entityID)
}

func (resourceApplicationEntityHelper *ResourceApplicationEntityHelper) parseUniqueID(ID string) (applicationID, entityType, entityID string, err error) {
	applicationEntityParts := strings.Split(ID, "/")
	if len(applicationEntityParts) < 5 {
		err = NewInvalidResourceIDError("application entity", ID)
		return
	}

	applicationID = applicationEntityParts[1]
	entityType = applicationEntityParts[3]
	entityID = applicationEntityParts[4]
	return
}

//endregion
