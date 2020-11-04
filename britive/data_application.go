package britive

import (
	"context"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//DataSourceApplication - Terraform Application DataSource
type DataSourceApplication struct {
	Resource *schema.Resource
}

//NewDataSourceApplication - Initialises new DataSourceApplication
func NewDataSourceApplication() *DataSourceApplication {
	dsa := &DataSourceApplication{}
	dsa.Resource = &schema.Resource{
		ReadContext: dsa.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
	return dsa
}

func (dsa *DataSourceApplication) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	applicationName := d.Get("name").(string)

	if applicationName == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Name must be passed to get application"),
		})
		return diags
	}
	application, err := c.GetApplicationByName(applicationName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(application.AppContainerID)
	if err := d.Set("name", application.CatalogAppDisplayName); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
