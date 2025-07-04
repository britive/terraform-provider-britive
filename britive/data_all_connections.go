package britive

import (
	"context"
	"errors"

	"github.com/britive/terraform-provider-britive/britive-client-go"
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

	allConnections, err := c.GetAllConnections()
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(NewNotFoundErrorf("connections not found"))
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

	if err := d.Set("connections", results); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("all-connections")

	return nil
}
