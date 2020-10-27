package britive

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProfileIdentity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProfileIdentityCreate,
		ReadContext:   resourceProfileIdentityRead,
		DeleteContext: resourceProfileIdentityDelete,
		Schema: map[string]*schema.Schema{
			"profile_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the profile",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The username associate wit the profile",
			},
		},
	}
}

func resourceProfileIdentityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID := d.Get("profile_id").(string)
	username := d.Get("username").(string)

	user, err := c.GetUserByName(username)
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.CreateProfileIdentity(profileID, user.UserID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(generateProfileIdentityUniqueID(profileID, user.UserID))

	return diags
}

func generateProfileIdentityUniqueID(profileID string, userID string) string {
	return fmt.Sprintf("paps/%s/users/%s", profileID, userID)
}

func parseProfileIdentityUniqueID(ID string) (profileID string, userID string, err error) {
	profileIdentityParts := strings.Split(ID, "/")
	if len(profileIdentityParts) < 4 {
		err = fmt.Errorf("Invalid profile identity reference, please check the state for %s", ID)
		return
	}
	profileID = profileIdentityParts[1]
	userID = profileIdentityParts[3]
	return
}

func resourceProfileIdentityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profileID, userID, err := parseProfileIdentityUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.GetProfileIdentity(profileID, userID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(generateProfileIdentityUniqueID(profileID, userID))

	return diags
}

func resourceProfileIdentityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, userID, err := parseProfileIdentityUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteProfileIdentity(profileID, userID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags
}
