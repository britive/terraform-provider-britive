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

// DataSourceApplication - Terraform Application DataSource
type DataSourceApplication struct {
	Resource *schema.Resource
}

// NewDataSourceApplication - Initializes new DataSourceApplication
func NewDataSourceApplication() *DataSourceApplication {
	dataSourceApplication := &DataSourceApplication{}
	dataSourceApplication.Resource = &schema.Resource{
		ReadContext: dataSourceApplication.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"environment_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "A set of environment ids for the application",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"environment_group_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "A set of environment group ids for the application",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"environment_ids_names": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "A set of map of environment ids and names for the application",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The environment id",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The environment name",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
			},
			"environment_group_ids_names": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "A set of map of environment group ids and names for the application",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The environment group id",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The environment group name",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
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
		return diag.FromErr(errs.NewNotFoundErrorf("application %s", applicationName))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(application.AppContainerID)

	if err := d.Set("name", application.CatalogAppDisplayName); err != nil {
		return diag.FromErr(err)
	}

	appEnvs, err := c.GetAppEnvs(d.Id(), "environments")
	if err != nil {
		return diag.FromErr(err)
	}

	appEnvGroups, err := c.GetAppEnvs(d.Id(), "environmentGroups")
	if err != nil {
		return diag.FromErr(err)
	}

	envIdList, err := c.GetEnvDetails(appEnvs, "id")
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("environment_ids", envIdList)

	envGrpIdList, err := c.GetEnvDetails(appEnvGroups, "id")
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("environment_group_ids", envGrpIdList)

	envIdNameList, err := c.GetEnvFullDetails(appEnvs)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("environment_ids_names", envIdNameList)

	envGrpIdNameList, err := c.GetEnvFullDetails(appEnvGroups)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("environment_group_ids_names", envGrpIdNameList)

	return nil
}
