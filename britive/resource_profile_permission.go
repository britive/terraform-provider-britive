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

	profilePermission := britive.ProfilePermission{
		Name: permission["name"].(string),
		Type: permission["type"].(string),
	}
	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation:  "add",
		Permission: profilePermission,
	}

	pp, err := c.PerformProfilePermissionRequest(profileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("paps/%s/permissions/%s/type/%s", pp.ProfileID, pp.Name, pp.Type))

	return diags
}

func resourceProfilePermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileMemberID := d.Id()
	profileMemberParts := strings.Split(profileMemberID, "/")
	if len(profileMemberParts) < 6 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Invalid user profile member reference, please check the state for %s", profileMemberID),
		})
		return diags
	}
	pp, err := c.GetProfilePermission(profileMemberParts[1], britive.ProfilePermission{
		Name: profileMemberParts[3],
		Type: profileMemberParts[5],
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("paps/%s/permissions/%s/type/%s", pp.ProfileID, pp.Name, pp.Type))

	return diags
}

func resourceProfilePermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileMemberID := d.Id()
	profileMemberParts := strings.Split(profileMemberID, "/")
	if len(profileMemberParts) < 6 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Invalid user profile member reference, please check the state for %s", profileMemberID),
		})
		return diags
	}
	profilePermission := britive.ProfilePermission{
		Name: profileMemberParts[3],
		Type: profileMemberParts[5],
	}
	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation:  "remove",
		Permission: profilePermission,
	}

	_, err := c.PerformProfilePermissionRequest(profileMemberParts[1], profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags

}
