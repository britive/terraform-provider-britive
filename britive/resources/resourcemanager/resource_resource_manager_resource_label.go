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

// ResourceResourceLabel - Terraform Resource for Resource Label
type ResourceResourceLabel struct {
	Resource     *schema.Resource
	helper       *ResourceResourceLabelHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceResourceLabelHelper - Helper for Resource Label Resource
type ResourceResourceLabelHelper struct{}

func NewResourceResourceLabelHelper() *ResourceResourceLabelHelper {
	return &ResourceResourceLabelHelper{}
}

func NewResourceResourceLabel(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResourceLabel {
	rl := &ResourceResourceLabel{
		helper:       NewResourceResourceLabelHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rl.Resource = &schema.Resource{
		CreateContext: rl.resourceCreate,
		UpdateContext: rl.resourceUpdate,
		ReadContext:   rl.resourceRead,
		DeleteContext: rl.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rl.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Resource Label Name",
				ValidateFunc: rl.validation.StringWithNoSpecialChar,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Resource Label Description",
			},
			"internal": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Resource Label Internal",
			},
			"label_color": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Resource Label Color",
				DiffSuppressFunc: func(key, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"values": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Resource Label Values",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Resource Label Value ID",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Resource Label Value Name",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Resource Label Value Description",
						},
						"created_by": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Resource Label Value CreatedBy",
						},
						"updated_by": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Resource Label Value CreatedBy",
						},
						"created_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Resource Label Value CreatedBy",
						},
						"updated_on": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Resource Label Value CreatedBy",
						},
					},
				},
			},
		},
	}
	return rl
}

func (rl *ResourceResourceLabel) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	log.Printf("[INFO] Mapping Resource Label Resource to Model")

	resourceLabel := &britive.ResourceLabel{}
	err := rl.helper.mapResourceToModel(d, resourceLabel)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating Resource Label Resource")
	resourceLabel, err = c.CreateUpdateResourceLabel(*resourceLabel, false)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("Resource Label Resource"))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rl.helper.generateUniqueID(resourceLabel.LabelId))
	log.Printf("[INFO] Created Resource Label Resource")

	return rl.resourceRead(ctx, d, m)
}

func (rl *ResourceResourceLabel) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	labelId, err := rl.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading Resource Label Resource of %s", labelId)
	resourceLabel, err := c.GetResourceLabel(labelId)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("Resource Label Resource", labelId))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	rl.helper.getAndMapModelToResource(d, *resourceLabel)
	log.Printf("[INFO] Received Resource Label Resource")

	return diags
}

func (rl *ResourceResourceLabel) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	labelId, err := rl.helper.parseUniqueID(d.Id())
	if err != nil {
		diag.FromErr(err)
	}

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("label_color") || d.HasChange("values") {
		resourceLabel := &britive.ResourceLabel{
			LabelId: labelId,
		}
		err := rl.helper.mapResourceToModel(d, resourceLabel)
		if err != nil {
			return diag.FromErr(err)
		}

		resourceLabel, err = c.CreateUpdateResourceLabel(*resourceLabel, true)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("Resource Label Resource", labelId))
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return rl.resourceRead(ctx, d, m)
}

func (rl *ResourceResourceLabel) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	labelId, err := rl.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting Resource Label Resource")
	err = c.DeleteResourceLabel(labelId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[INFO] Deleted Resource Label Resource")

	return diags
}

func (rl *ResourceResourceLabel) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	if err := rl.importHelper.ParseImportID([]string{"resource-manager/resource-labels/(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	labelId := d.Id()
	log.Printf("[INFO] Importing resource label: %s", labelId)

	resp, err := c.GetResourceLabel(labelId)
	if err != nil {
		return nil, err
	}

	d.SetId(rl.helper.generateUniqueID(resp.LabelId))
	rl.helper.getAndMapModelToResource(d, *resp)
	log.Printf("[INFO] Imported resource label: %s", labelId)
	return []*schema.ResourceData{d}, nil
}

func (helper *ResourceResourceLabelHelper) mapResourceToModel(d *schema.ResourceData, resourceLabel *britive.ResourceLabel) error {
	resourceLabel.Name = d.Get("name").(string)
	if v, ok := d.GetOk("description"); ok {
		resourceLabel.Description = v.(string)
	}
	if v, ok := d.GetOk("label_color"); ok {
		resourceLabel.LabelColor = v.(string)
	}
	var resourceLabelValues []interface{}
	if v, ok := d.GetOk("values"); ok {
		resourceLabelValuesSet := v.(*schema.Set)
		resourceLabelValues = resourceLabelValuesSet.List()
	}
	for _, val := range resourceLabelValues {
		value := val.(map[string]interface{})
		resourceLabelValue := &britive.ResourceLabelValue{
			Name: value["name"].(string),
		}
		if desc, ok := value["description"]; ok {
			resourceLabelValue.Description = desc.(string)
		}
		if createdBy, ok := value["created_by"]; ok {
			resourceLabelValue.CreatedBy = createdBy.(int)
		}
		if updatedBy, ok := value["updated_by"]; ok {
			resourceLabelValue.UpdatedBy = updatedBy.(int)
		}
		if createdOn, ok := value["created_on"]; ok {
			resourceLabelValue.CreatedOn = createdOn.(string)
		}
		if updatedOn, ok := value["updated_on"]; ok {
			resourceLabelValue.UpdatedOn = updatedOn.(string)
		}
		resourceLabel.Values = append(resourceLabel.Values, *resourceLabelValue)
	}

	return nil
}

func (helper *ResourceResourceLabelHelper) getAndMapModelToResource(d *schema.ResourceData, resourceLabel britive.ResourceLabel) {
	d.Set("name", resourceLabel.Name)
	d.Set("description", resourceLabel.Description)
	d.Set("internal", resourceLabel.Internal)
	d.Set("label_color", resourceLabel.LabelColor)
	var resourceLabelValues []map[string]interface{}
	for _, val := range resourceLabel.Values {
		resourceLabelValue := make(map[string]interface{})
		resourceLabelValue["value_id"] = val.ValueId
		resourceLabelValue["name"] = val.Name
		resourceLabelValue["description"] = val.Description
		resourceLabelValue["created_by"] = val.CreatedBy
		resourceLabelValue["updated_by"] = val.UpdatedBy
		resourceLabelValue["created_on"] = val.CreatedOn
		resourceLabelValue["updated_on"] = val.UpdatedOn

		resourceLabelValues = append(resourceLabelValues, resourceLabelValue)
	}
	d.Set("values", resourceLabelValues)
}

func (helper *ResourceResourceLabelHelper) generateUniqueID(labelId string) string {
	return fmt.Sprintf("resource-manager/resource-labels/%s", labelId)
}

func (helper *ResourceResourceLabelHelper) parseUniqueID(labelId string) (string, error) {
	labelIdArr := strings.Split(labelId, "/")
	if len(labelIdArr) != 3 {
		return "", errs.NewNotFoundErrorf("Resource Label Id")
	}
	labelId = labelIdArr[len(labelIdArr)-1]
	return labelId, nil
}
