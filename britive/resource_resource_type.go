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

// ResourceResourceType - Terraform Resource for Resource Type
type ResourceResourceType struct {
	Resource     *schema.Resource
	helper       *ResourceResourceTypeHelper
	validation   *Validation
	importHelper *ImportHelper
}

// NewResourceResourceType - Initializes new resource type resource
func NewResourceResourceType(v *Validation, importHelper *ImportHelper) *ResourceResourceType {
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
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive resource type",
			},
			"parameters": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Parameters/Fields of the resource type",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"param_type": {
							Type:     schema.TypeString,
							Required: true,
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
	if d.HasChange("description") || d.HasChange("parameters") {
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

		log.Printf("[INFO] Submitted updated resource type: %#v", ur)
		d.SetId(rt.helper.generateUniqueID(resourceTypeID))
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
	log.Printf("[INFO] resource type %s deleted", resourceTypeID)
	d.SetId("")

	return diags
}

func (rt *ResourceResourceType) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	resourceTypeID := d.Id()
	if strings.TrimSpace(resourceTypeID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("id")
	}

	log.Printf("[INFO] Importing resource type: %s", resourceTypeID)

	resourceType, err := c.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("resource type %s", resourceTypeID)
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

	for i := 0; i < len(parameters.List()); i++ {
		parameter := parameters.List()[i]
		resourceType.Parameters = append(resourceType.Parameters,
			britive.Parameter{
				Name:        parameter.(map[string]interface{})["name"].(string),
				ParamType:   parameter.(map[string]interface{})["param_type"].(string),
				IsMandatory: parameter.(map[string]interface{})["is_mandatory"].(bool),
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
		return NewNotFoundErrorf("resourceType %s", resourceTypeID)
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

	var parameterList []map[string]interface{}
	for _, parameter := range resourceType.Parameters {
		parameterList = append(parameterList, map[string]interface{}{
			"name":         parameter.Name,
			"param_type":   parameter.ParamType,
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
		err = NewInvalidResourceIDError("resourceType", ID)
		return
	}

	resourceTypeID = resourceTypeParts[2]
	return
}
