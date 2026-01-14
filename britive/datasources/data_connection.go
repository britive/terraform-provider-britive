package datasources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
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
			"setting_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "ITSM",
				Description: "Advanced Setting Type",
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
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

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

	connectionName := d.Get("name").(string)

	isConnectionFound := false
	allConnectionNames := make([]string, 0)
	for _, conn := range allConnections {
		if strings.EqualFold(conn.Name, connectionName) {
			d.SetId(conn.ID)
			d.Set("name", connectionName)
			d.Set("type", conn.Type)
			d.Set("auth_type", conn.AuthType)
			d.Set("setting_type", settingType)
			isConnectionFound = true
		}
		allConnectionNames = append(allConnectionNames, conn.Name+",")
	}

	if !isConnectionFound {
		totalConnections := len(allConnectionNames) - 1
		var errMsg error
		if totalConnections >= 0 {
			allConnectionNames[totalConnections] = allConnectionNames[totalConnections][:len(allConnectionNames[totalConnections])-1]
			errMsg = fmt.Errorf("Invalid connection name.\nTry with %v", allConnectionNames)
		} else {
			errMsg = fmt.Errorf("Invalid connection name.")
		}
		return diag.FromErr(errMsg)
	}

	return nil
}
