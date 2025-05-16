package britive

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
			"catalog_app_display_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Britive application catalog name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"catalog_app_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Britive application catalog id",
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
			"user_account_mappings": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Application user account",
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

	application := britive.ApplicationRequest{}

	err := rt.helper.mapApplicationResourceToModel(d, m, &application, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new application: %#v", application)

	rto, err := c.CreateApplication(application)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new application: %#v", rto)

	d.SetId(rt.helper.generateUniqueID(rto.AppContainerId))

	log.Printf("[INFO] Updated state after application submission: %#v", rto)

	// Patch properties
	properties := britive.Properties{}
	err = rt.helper.mapPropertiesResourceToModel(d, m, &properties, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating application properties: %#v", properties)
	_, err = c.PatchApplicationPropertyTypes(rto.AppContainerId, properties)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Updated application properties: %#v", properties)

	// configer user account mappings
	userMappings := britive.UserMappings{}
	err = rt.helper.mapUserMappingsResourceToModel(d, m, &userMappings, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating user mappings: %#v", userMappings)
	err = c.ConfigureUserMappings(rto.AppContainerId, userMappings)
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
	if d.HasChange("user_account_mappings") {
		hasChanges = true
		// Update user mapping
	}
	if d.HasChange("properties") {
		hasChanges = true
		properties := britive.Properties{}
		err := rt.helper.mapPropertiesResourceToModel(d, m, &properties, false)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating application properties: %#v", properties)
		_, err = c.PatchApplicationPropertyTypes(applicationID, properties)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updated application properties: %#v", properties)

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

	err = rt.helper.getAndMapModelToResource(d, m)
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

//region Resource Type Resource helper functions

func (rrth *ResourceApplicationHelper) mapApplicationResourceToModel(d *schema.ResourceData, m interface{}, application *britive.ApplicationRequest, isUpdate bool) error {
	application.CatalogAppId = d.Get("catalog_app_id").(int)
	application.CatalogAppDisplayName = d.Get("catalog_app_display_name").(string)
	return nil
}

func (rrth *ResourceApplicationHelper) mapPropertiesResourceToModel(d *schema.ResourceData, m interface{}, properties *britive.Properties, isUpdate bool) error {
	propertyTypes := d.Get("properties").(*schema.Set)
	for _, property := range propertyTypes.List() {
		propertyType := britive.PropertyTypes{}
		propertyType.Name = property.(map[string]interface{})["name"].(string)
		propertyType.Value = property.(map[string]interface{})["value"].(string)
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
	if err := d.Set("catalog_app_display_name", application.CatalogAppDisplayName); err != nil {
		return err
	}

	applicationProperties := application.Properties.PropertyTypes
	propertiesMap := make(map[string]string)
	for _, property := range applicationProperties {
		propertiesMap[property.Name] = fmt.Sprintf("%v", property.Value)
	}

	var stateProperties []map[string]interface{}
	properties := d.Get("properties").(*schema.Set)
	for _, property := range properties.List() {
		propertyName := property.(map[string]interface{})["name"].(string)
		stateProperties = append(stateProperties, map[string]interface{}{
			"name":  propertyName,
			"value": propertiesMap[propertyName],
		})
	}
	if err := d.Set("properties", stateProperties); err != nil {
		return err
	}

	// paramMap := make(map[string]string)

	// for i := 0; i < len(stateParamameters.List()); i++ {
	// 	parameter := stateParamameters.List()[i]
	// 	paramName := parameter.(map[string]interface{})["param_name"].(string)
	// 	paramType := parameter.(map[string]interface{})["param_type"].(string)
	// 	paramMap[paramName] = paramType
	// }

	// var parameterList []map[string]interface{}
	// for _, parameter := range application.Parameters {
	// 	parameterList = append(parameterList, map[string]interface{}{
	// 		"param_name":   parameter.ParamName,
	// 		"param_type":   paramMap[parameter.ParamName],
	// 		"is_mandatory": parameter.IsMandatory,
	// 	})
	// }

	// if err := d.Set("parameters", parameterList); err != nil {
	// 	return err
	// }
	return nil
}

func (resourceApplicationHelper *ResourceApplicationHelper) generateUniqueID(applicationID string) string {
	return applicationID
}

func (resourceApplicationHelper *ResourceApplicationHelper) parseUniqueID(ID string) (applicationID string, err error) {
	return ID, nil
}
