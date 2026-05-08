package datasources

import (
	"context"
	"errors"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceUserAttribute - Terraform User Attribute DataSource
type DataSourceUserAttribute struct {
	Resource *schema.Resource
}

// NewDataSourceUserAttribute - Initializes new DataSourceUserAttribute
func NewDataSourceUserAttribute() *DataSourceUserAttribute {
	dataSourceUserAttribute := &DataSourceUserAttribute{}
	dataSourceUserAttribute.Resource = &schema.Resource{
		ReadContext: dataSourceUserAttribute.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the user attribute",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"name", "attribute_schema_id"},
			},
			"attribute_schema_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The unique identifier of the user attribute",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"name", "attribute_schema_id"},
			},
		},
	}
	return dataSourceUserAttribute
}

func (dataSourceUserAttribute *DataSourceUserAttribute) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var attribute *britive.UserAttribute
	var err error

	if attributeSchemaID, ok := d.GetOk("attribute_schema_id"); ok {
		attribute, err = c.GetAttribute(attributeSchemaID.(string))
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("attribute with id %s", attributeSchemaID.(string)))
		}
	} else {
		attributeName := d.Get("name").(string)
		attribute, err = c.GetAttributeByName(attributeName)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("attribute %s", attributeName))
		}
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(attribute.ID)

	if err := d.Set("name", attribute.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("attribute_schema_id", attribute.ID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
