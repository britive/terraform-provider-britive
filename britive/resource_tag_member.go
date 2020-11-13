package britive

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceTagMember - Terraform Resource for Tag member
type ResourceTagMember struct {
	Resource     *schema.Resource
	helper       *ResourceTagMemberHelper
	importHelper *ImportHelper
}

//NewResourceTagMember - Initialises new tag member resource
func NewResourceTagMember(importHelper *ImportHelper) *ResourceTagMember {
	rtm := &ResourceTagMember{
		helper:       NewResourceTagMemberHelper(),
		importHelper: importHelper,
	}
	rtm.Resource = &schema.Resource{
		CreateContext: rtm.resourceCreate,
		ReadContext:   rtm.resourceRead,
		DeleteContext: rtm.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rtm.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"tag_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the tag",
			},
			"tag_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the tag",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The username associate with the tag",
			},
		},
	}
	return rtm
}

//region Tag member Resource Context Operations

func (rtm *ResourceTagMember) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	tagID := d.Get("tag_id").(string)
	username := d.Get("username").(string)

	user, err := c.GetUserByName(username)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new tag member: %s/%s", tagID, user.UserID)
	err = c.CreateTagMember(tagID, user.UserID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new tag member: %s/%s", tagID, user.UserID)
	d.SetId(rtm.helper.generateUniqueID(tagID, user.UserID))

	return rtm.resourceRead(ctx, d, m)
}

func (rtm *ResourceTagMember) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	tagID, userID, err := rtm.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = rtm.helper.getAndMapModelToResource(tagID, userID, d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func (rtm *ResourceTagMember) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	tagID, userID, err := rtm.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting tag member %s/%s", tagID, userID)

	err = c.DeleteTagMember(tagID, userID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted tag member %s/%s", tagID, userID)
	d.SetId("")

	return diags
}

func (rtm *ResourceTagMember) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	if err := rtm.importHelper.ParseImportID([]string{"tags/(?P<tag_name>[^/]+)/users/(?P<username>[^/]+)", "(?P<tag_name>[^/]+)/(?P<username>[^/]+)"}, d); err != nil {
		return nil, err
	}

	tagName := d.Get("tag_name").(string)
	username := d.Get("username").(string)

	log.Printf("[INFO] Importing tag member %s/%s", tagName, username)

	tag, err := c.GetTagByName(tagName)
	if err != nil {
		return nil, err
	}
	user, err := c.GetUserByName(username)
	if err != nil {
		return nil, err
	}

	d.SetId(rtm.helper.generateUniqueID(tag.ID, user.UserID))
	d.Set("tag_name", "")

	err = rtm.helper.getAndMapModelToResource(tag.ID, user.UserID, d, m)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Imported tag member %s/%s", tagName, username)

	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceTagMemberHelper - Resource Tag member helper functions
type ResourceTagMemberHelper struct {
}

//NewResourceTagMemberHelper - Initialises new tag member resource helper
func NewResourceTagMemberHelper() *ResourceTagMemberHelper {
	return &ResourceTagMemberHelper{}
}

//region Tag member Resource helper functions

func (rtmh *ResourceTagMemberHelper) generateUniqueID(tagID string, userID string) string {
	return fmt.Sprintf("user-tags/%s/users/%s", tagID, userID)
}

func (rtmh *ResourceTagMemberHelper) parseUniqueID(ID string) (tagID string, userID string, err error) {
	tagMemberParts := strings.Split(ID, "/")
	if len(tagMemberParts) < 4 {
		err = fmt.Errorf("Invalid user tag member reference, please check the state for %s", ID)
		return
	}
	tagID = tagMemberParts[1]
	userID = tagMemberParts[3]
	return
}

func (rtmh *ResourceTagMemberHelper) getAndMapModelToResource(tagID string, userID string, d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	log.Printf("[INFO] Reading tag member %s/%s", tagID, userID)
	u, err := c.GetTagMember(tagID, userID)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received tag member %#v", u)

	if err := d.Set("tag_id", tagID); err != nil {
		return err
	}
	if err := d.Set("username", u.Username); err != nil {
		return err
	}

	return nil
}

//endregion
