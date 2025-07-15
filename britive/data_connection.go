package britive

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DataSourceConnection struct {
	Resource *schema.Resource
}

func NewDataSourceConnection() *DataSourceConnection {
	dataSourceConnection := &DataSourceConnection{}
	dataSourceConnection.Resource = &schema.Resource{
		ReadContext: dataSourceConnection.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
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
	}
	return dataSourceConnection
}

func (dataSourceConnections *DataSourceConnection) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	allConnections, err := c.GetAllConnections()
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(NewNotFoundErrorf("connections not found"))
	}

	connectionName := d.Get("name").(string)

	isConnectionFound := false
	allConnectionNames := make([]string, 0)
	for _, conn := range allConnections {
		if strings.EqualFold(conn.Name, connectionName) {
			d.SetId(conn.ID)
			d.Set("name", connectionName)
			d.Set("type", conn.Type)
			d.Set("auth_type", conn.AuthType)
			isConnectionFound = true
		}
		allConnectionNames = append(allConnectionNames, conn.Name)
	}

	if !isConnectionFound {
		return diag.FromErr(fmt.Errorf("Invalid connection name.\nTry with %v", allConnectionNames))
	}

	return nil
}
