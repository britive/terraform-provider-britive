package britive

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTagMember() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTagMemberCreate,
		ReadContext:   resourceTagMemberRead,
		DeleteContext: resourceTagMemberDelete,
		Schema: map[string]*schema.Schema{
			"tag_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the tag",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The username associate wit the tag",
			},
		},
	}
}

func resourceTagMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	tagID := d.Get("tag_id").(string)
	username := d.Get("username").(string)

	user, err := c.GetUserByName(username)
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.CreateTagMember(tagID, user.UserID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(generateTagMemberUniqueID(tagID, user.UserID))

	return diags
}

func generateTagMemberUniqueID(tagID string, userID string) string {
	return fmt.Sprintf("user-tags/%s/users/%s", tagID, userID)
}

func parseTagMemberUniqueID(ID string) (tagID string, userID string, err error) {
	tagMemberParts := strings.Split(ID, "/")
	if len(tagMemberParts) < 4 {
		err = fmt.Errorf("Invalid user tag member reference, please check the state for %s", ID)
		return
	}
	tagID = tagMemberParts[1]
	userID = tagMemberParts[3]
	return
}

func resourceTagMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	tagID, userID, err := parseTagMemberUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.GetTagMember(tagID, userID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(generateTagMemberUniqueID(tagID, userID))

	return diags
}

func resourceTagMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	tagID, userID, err := parseTagMemberUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteTagMember(tagID, userID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags
}
