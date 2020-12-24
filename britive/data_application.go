package britive

import (
	"context"
	"errors"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//DataSourceApplication - Terraform Application DataSource
type DataSourceApplication struct {
	Resource *schema.Resource
}

//NewDataSourceApplication - Initializes new DataSourceApplication
func NewDataSourceApplication() *DataSourceApplication {
	dataSourceApplication := &DataSourceApplication{}
	dataSourceApplication.Resource = &schema.Resource{
		ReadContext: dataSourceApplication.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
	return dataSourceApplication
}

func (dataSourceApplication *DataSourceApplication) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	applicationName := d.Get("name").(string)

	application, err := c.GetApplicationByName(applicationName)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(NewNotFoundErrorf("application %s", applicationName))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(application.AppContainerID)

	if err := d.Set("name", application.CatalogAppDisplayName); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
