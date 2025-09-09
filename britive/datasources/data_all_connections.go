package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DataSourceAllConnections struct {
	Resource *schema.Resource
}

func NewDataSourceAllConnections() *DataSourceAllConnections {
	dataSourceAllConnections := &DataSourceAllConnections{}
	dataSourceAllConnections.Resource = &schema.Resource{
		ReadContext: dataSourceAllConnections.resourceRead,
		Schema: map[string]*schema.Schema{
			"setting_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "ITSM",
				Description: "Advanced Setting Type",
			},
			"connections": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "all connections",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Id of connection",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of connection",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Type of connection",
						},
						"auth_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Auth type of connection",
						},
					},
				},
			},
		},
	}
	return dataSourceAllConnections
}

func (dataSourceAllConnections *DataSourceAllConnections) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	settingType := d.Get("setting_type").(string)

	allConnections, err := c.GetAllConnections(settingType)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("connections not found"))
	} else if errors.Is(err, britive.ErrNotSupported) {
		return diag.FromErr(errs.NewNotSupportedError(fmt.Sprintf("%s setting type is ", settingType)))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	var results []map[string]interface{}
	for _, conn := range allConnections {
		results = append(results, map[string]interface{}{
			"id":        conn.ID,
			"name":      conn.Name,
			"type":      conn.Type,
			"auth_type": conn.AuthType,
		})
	}

	if err := d.Set("setting_type", settingType); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("connections", results); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("all-connections")

	return nil
}
