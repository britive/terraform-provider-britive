package datasources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DataSourceResourceManagerProfilePermissions struct {
	Resource *schema.Resource
}

func NewDataSourceResourceManagerProfilePermissions() *DataSourceResourceManagerProfilePermissions {
	dsrmpp := &DataSourceResourceManagerProfilePermissions{}
	dsrmpp.Resource = &schema.Resource{
		ReadContext: dsrmpp.resourceRead,
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of connection",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Available Permissions",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of permission",
						},
						"permission_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Permission ID",
						},
						"version": {
							Type:        schema.TypeSet,
							Computed:    true,
							Description: "Versions of the permission",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
	return dsrmpp
}

func (dsrmpp *DataSourceResourceManagerProfilePermissions) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	profileIDArr := strings.Split(d.Get("profile_id").(string), "/")
	profileID := profileIDArr[len(profileIDArr)-1]

	log.Printf("[INFO] Reading all available permissions for profile: %s", profileID)

	allAvailablePermissions, err := c.GetAvailablePermissions(profileID)
	if err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("permissions not found"))
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Read all available permissions: %#v", allAvailablePermissions)

	if allAvailablePermissions == nil || allAvailablePermissions.Permissions == nil {
		return diag.Errorf("received nil response or nil permissions list from GetAvailablePermissions")
	}

	var permissions []map[string]interface{}

	for _, val := range allAvailablePermissions.Permissions {
		perm := make(map[string]interface{})

		if name, ok := val["name"].(string); ok {
			perm["name"] = name
		}
		if pid, ok := val["permissionId"].(string); ok {
			perm["permission_id"] = pid
		}

		permissionVersions, err := c.GetPermissionVersions(perm["permission_id"].(string))
		if err != nil {
			return diag.FromErr(err)
		}

		var version []interface{}
		for _, v := range permissionVersions {
			version = append(version, v["version"])
		}
		version = append(version, "local")
		version = append(version, "latest")

		perm["version"] = version

		permissions = append(permissions, perm)
	}

	d.SetId(fmt.Sprintf("profile/%s/available-permissions", profileID))
	if err := d.Set("permissions", permissions); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Set all available permissions: %#v", permissions)

	return nil
}
