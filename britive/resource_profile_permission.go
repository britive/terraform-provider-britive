package britive

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceProfilePermission - Terraform Resource for Profile Permission
type ResourceProfilePermission struct {
	Resource *schema.Resource
	helper   *ResourceProfilePermissionHelper
}

//NewResourceProfilePermission - Initialisation of new profile permission resource
func NewResourceProfilePermission() *ResourceProfilePermission {
	rpp := &ResourceProfilePermission{
		helper: NewResourceProfilePermissionHelper(),
	}
	rpp.Resource = &schema.Resource{
		CreateContext: rpp.resourceCreate,
		ReadContext:   rpp.resourceRead,
		DeleteContext: rpp.resourceDelete,
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
				Description: "The permission associate with the profile",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "The name of permission",
						},
						"type": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "The type of permission",
						},
					},
				},
			},
		},
	}
	return rpp
}

func (rpp *ResourceProfilePermission) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	err := c.ExecuteProfilePermissionRequest(profileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rpp.helper.generateUniqueID(profilePermissionRequest.Permission))

	return diags
}

func (rpp *ResourceProfilePermission) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profilePermission, err := rpp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	pp, err := c.GetProfilePermission(profilePermission.ProfileID, *profilePermission)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(rpp.helper.generateUniqueID(*pp))

	return diags
}

func (rpp *ResourceProfilePermission) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profilePermission, err := rpp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation:  "remove",
		Permission: *profilePermission,
	}

	err = c.ExecuteProfilePermissionRequest(profilePermission.ProfileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags

}

//ResourceProfilePermissionHelper - Terraform Resource for Profile Permission Helper
type ResourceProfilePermissionHelper struct {
}

//NewResourceProfilePermissionHelper - Initialisation of new profile tag resource helper
func NewResourceProfilePermissionHelper() *ResourceProfilePermissionHelper {
	return &ResourceProfilePermissionHelper{}
}

func (rpph *ResourceProfilePermissionHelper) generateUniqueID(profilePermission britive.ProfilePermission) string {
	return fmt.Sprintf("paps/%s/permissions/%s/type/%s", profilePermission.ProfileID, profilePermission.Name, profilePermission.Type)
}

func (rpph *ResourceProfilePermissionHelper) parseUniqueID(ID string) (*britive.ProfilePermission, error) {
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
