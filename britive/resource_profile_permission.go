package britive

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProfilePermission() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProfilePermissionCreate,
		ReadContext:   resourceProfilePermissionRead,
		DeleteContext: resourceProfilePermissionDelete,
		Schema: map[string]*schema.Schema{
			"profile_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the profile",
			},
			"permission": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				ForceNew:    true,
				Description: "The permission associate wit the profile",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of permission",
						},
						"type": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of permission",
						},
					},
				},
			},
		},
	}
}

func resourceProfilePermissionCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID := d.Get("profile_id").(string)

	pd := d.Get("permission").([]interface{})[0]

	permission := pd.(map[string]interface{})

	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation: "add",
		Permission: britive.ProfilePermission{
			ProfileID: profileID,
			Name:      permission["name"].(string),
			Type:      permission["type"].(string),
		},
	}

	err := c.PerformProfilePermissionRequest(profileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(generateProfilePermissionUniqueID(profilePermissionRequest.Permission))

	return diags
}

func generateProfilePermissionUniqueID(profilePermission britive.ProfilePermission) string {
	return fmt.Sprintf("paps/%s/permissions/%s/type/%s", profilePermission.ProfileID, profilePermission.Name, profilePermission.Type)
}

func parsePerfilePermissionUniqueID(ID string) (*britive.ProfilePermission, error) {
	profileMemberParts := strings.Split(ID, "/")

	if len(profileMemberParts) < 6 {
		return nil, fmt.Errorf("Invalid user profile member reference, please check the state for %s", ID)

	}
	profilePermission := &britive.ProfilePermission{
		ProfileID: profileMemberParts[1],
		Name:      profileMemberParts[3],
		Type:      profileMemberParts[5],
	}
	return profilePermission, nil
}

func resourceProfilePermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profilePermission, err := parsePerfilePermissionUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	pp, err := c.GetProfilePermission(profilePermission.ProfileID, *profilePermission)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(generateProfilePermissionUniqueID(*pp))

	return diags
}

func resourceProfilePermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profilePermission, err := parsePerfilePermissionUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation:  "remove",
		Permission: *profilePermission,
	}

	err = c.PerformProfilePermissionRequest(profilePermission.ProfileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags

}
