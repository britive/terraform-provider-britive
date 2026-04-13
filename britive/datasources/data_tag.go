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

// DataSourceTag - Terraform Tag DataSource
type DataSourceTag struct {
	Resource *schema.Resource
}

// NewDataSourceTag - Initializes new DataSourceTag
func NewDataSourceTag() *DataSourceTag {
	dataSourceTag := &DataSourceTag{}
	dataSourceTag.Resource = &schema.Resource{
		ReadContext: dataSourceTag.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"name", "tag_id"},
			},
			"tag_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The unique identifier of the tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"name", "tag_id"},
			},
		},
	}
	return dataSourceTag
}

func (dataSourceTag *DataSourceTag) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var tag *britive.Tag
	var err error

	if tagID, ok := d.GetOk("tag_id"); ok {
		tag, err = c.GetTag(tagID.(string))
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("tag with id %s", tagID.(string)))
		}
	} else {
		tagName := d.Get("name").(string)
		tag, err = c.GetTagByName(tagName)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("tag %s", tagName))
		}
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tag.ID)

	if err := d.Set("name", tag.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tag_id", tag.ID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
