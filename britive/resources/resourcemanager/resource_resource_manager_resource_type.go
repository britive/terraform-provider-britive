package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceResourceType - Terraform Resource for Resource Type
type ResourceResourceType struct {
	Resource     *schema.Resource
	helper       *ResourceResourceTypeHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceResourceType - Initializes new resource type resource
func NewResourceResourceType(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResourceType {
	rt := &ResourceResourceType{
		helper:       NewResourceResourceTypeHelper(),
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
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of Britive resource type",
				ValidateFunc: rt.validation.StringWithNoSpecialChar,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive resource type",
			},
			"icon": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Icon of Britive resource type",
				ValidateFunc: rt.validation.ValidateSVGString,
			},
			"parameters": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Parameters/Fields of the resource type",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"param_name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: rt.validation.StringWithNoSpecialChar,
						},
						"param_type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								paramType := val.(string)
								if !strings.EqualFold(paramType, "string") && !strings.EqualFold(paramType, "password") {
									errs = append(errs, fmt.Errorf("paramater type '%s' is not supported, try with 'string' or 'password'", val))
								}
								return
							},
						},
						"is_mandatory": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
		},
	}
	return rt
}

//region Resource Type Resource Context Operations

func (rt *ResourceResourceType) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	resourceType := britive.ResourceType{}

	err := rt.helper.mapResourceToModel(d, m, &resourceType, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new resource type: %#v", resourceType)

	rto, err := c.CreateResourceType(resourceType)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new resource type: %#v", rto)

	log.Printf("[INFO] Adding icon to resource type: %#v", rto)
	userSVG := d.Get("icon").(string)
	err = c.AddRemoveIcon(rto.ResourceTypeID, userSVG)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		if err := c.DeleteResourceType(rto.ResourceTypeID); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
		return diags
	}

	log.Printf("[INFO] Added icon to resource type: %#v", rto)

	d.SetId(rt.helper.generateUniqueID(rto.ResourceTypeID))

	rt.resourceRead(ctx, d, m)

	return diags
}

func (rt *ResourceResourceType) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rt.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rt *ResourceResourceType) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	resourceTypeID, err := rt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var hasChanges bool
	if d.HasChange("description") || d.HasChange("parameters") || d.HasChange("name") {
		hasChanges = true
		resourceType := britive.ResourceType{}

		err := rt.helper.mapResourceToModel(d, m, &resourceType, true)
		if err != nil {
			return diag.FromErr(err)
		}

		ur, err := c.UpdateResourceType(resourceType, resourceTypeID)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] updated resource type: %#v", ur)

		d.SetId(rt.helper.generateUniqueID(resourceTypeID))
	}
	if d.HasChange("icon") {
		hasChanges = true
		log.Printf("[INFO] Updating icon to resource type: %#v", resourceTypeID)
		userSVG := d.Get("icon").(string)
		err = c.AddRemoveIcon(resourceTypeID, userSVG)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Added icon to resource type: %#v", resourceTypeID)
	}
	if hasChanges {
		return rt.resourceRead(ctx, d, m)
	}
	return nil
}

func (rt *ResourceResourceType) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	resourceTypeID, err := rt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting resource type: %s", resourceTypeID)
	err = c.DeleteResourceType(resourceTypeID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Resource type %s deleted", resourceTypeID)
	d.SetId("")

	return diags
}

func (rt *ResourceResourceType) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rt.importHelper.ParseImportID([]string{"resource-manager/resource-types/(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	resourceTypeID := d.Id()
	if strings.TrimSpace(resourceTypeID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("id")
	}

	log.Printf("[INFO] Importing resource type: %s", resourceTypeID)

	resourceType, err := c.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("resource type %s", resourceTypeID)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rt.helper.generateUniqueID(resourceType.ResourceTypeID))

	err = rt.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported resource type: %s", resourceTypeID)
	return []*schema.ResourceData{d}, nil
}

// ResourceResourceTypeHelper - Resource Resource Type helper functions
type ResourceResourceTypeHelper struct {
}

// NewResourceResourceTypeHelper - Initializes new resource type resource helper
func NewResourceResourceTypeHelper() *ResourceResourceTypeHelper {
	return &ResourceResourceTypeHelper{}
}

//region Resource Type Resource helper functions

func (rrth *ResourceResourceTypeHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, resourceType *britive.ResourceType, isUpdate bool) error {
	resourceType.Name = d.Get("name").(string)
	resourceType.Description = d.Get("description").(string)
	parameters := d.Get("parameters").(*schema.Set)

	for _, param := range parameters.List() {
		parameter := param.(map[string]interface{})
		paramName := parameter["param_name"].(string)
		paramType := parameter["param_type"].(string)

		resourceType.Parameters = append(resourceType.Parameters,
			britive.Parameter{
				ParamName:   paramName,
				ParamType:   strings.ToLower(paramType),
				IsMandatory: parameter["is_mandatory"].(bool),
			})
	}
	return nil
}

func (rrth *ResourceResourceTypeHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	resourceTypeID, err := rrth.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading resource type %s", resourceTypeID)

	resourceType, err := c.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("resourceType %s", resourceTypeID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received resource type %#v", resourceType)

	if err := d.Set("name", resourceType.Name); err != nil {
		return err
	}
	if err := d.Set("description", resourceType.Description); err != nil {
		return err
	}

	stateParamameters := d.Get("parameters").(*schema.Set)
	paramMap := make(map[string]string)

	for i := 0; i < len(stateParamameters.List()); i++ {
		parameter := stateParamameters.List()[i]
		paramName := parameter.(map[string]interface{})["param_name"].(string)
		paramType := parameter.(map[string]interface{})["param_type"].(string)
		paramMap[paramName] = paramType
	}

	var parameterList []map[string]interface{}
	for _, parameter := range resourceType.Parameters {
		parameterList = append(parameterList, map[string]interface{}{
			"param_name":   parameter.ParamName,
			"param_type":   paramMap[parameter.ParamName],
			"is_mandatory": parameter.IsMandatory,
		})
	}

	if err := d.Set("parameters", parameterList); err != nil {
		return err
	}
	return nil
}

func (resourceResourceTypeHelper *ResourceResourceTypeHelper) generateUniqueID(resourceTypeID string) string {
	return fmt.Sprintf("resource-manager/resource-types/%s", resourceTypeID)
}

func (resourceResourceTypeHelper *ResourceResourceTypeHelper) parseUniqueID(ID string) (resourceTypeID string, err error) {
	resourceTypeParts := strings.Split(ID, "/")
	if len(resourceTypeParts) < 3 {
		err = errs.NewInvalidResourceIDError("resourceType", ID)
		return
	}

	resourceTypeID = resourceTypeParts[2]
	return
}
