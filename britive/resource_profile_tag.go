package britive

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProfileTag() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProfileTagCreate,
		ReadContext:   resourceProfileTagRead,
		DeleteContext: resourceProfileTagDelete,
		Schema: map[string]*schema.Schema{
			"profile_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the profile",
			},
			"tag": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The tag associate wit the profile",
			},
		},
	}
}

func resourceProfileTagCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID := d.Get("profile_id").(string)
	tagName := d.Get("tag").(string)

	tag, err := c.GetTagByName(tagName)
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.CreateProfileTag(profileID, tag.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(generateProfileTagUniqueID(profileID, tag.ID))

	return diags
}

func generateProfileTagUniqueID(profileID string, tagID string) string {
	return fmt.Sprintf("paps/%s/user-tags/%s", profileID, tagID)
}

func parseProfileTagUniqueID(ID string) (profileID string, tagID string, err error) {
	profileTagParts := strings.Split(ID, "/")
	if len(profileTagParts) < 4 {
		err = fmt.Errorf("Invalid profile tag reference, please check the state for %s", ID)
		return
	}
	profileID = profileTagParts[1]
	tagID = profileTagParts[3]
	return
}

func resourceProfileTagRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profileID, tagID, err := parseProfileTagUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.GetProfileTag(profileID, tagID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(generateProfileTagUniqueID(profileID, tagID))

	return diags
}

func resourceProfileTagDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, tagID, err := parseProfileTagUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteProfileTag(profileID, tagID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags
}
