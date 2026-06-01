package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//ResourceTagMember - Terraform Resource for Tag member
type ResourceTagMember struct {
	Resource     *schema.Resource
	helper       *ResourceTagMemberHelper
	importHelper *imports.ImportHelper
}

//NewResourceTagMember - Initializes new tag member resource
func NewResourceTagMember(importHelper *imports.ImportHelper) *ResourceTagMember {
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
			"tag_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the Britive tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"tag_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the Britive tag",
			},
			"username": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The username of the user added to the Britive tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"user_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The identifier of the user added to the Britive tag",
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
	userID := strings.TrimSpace(d.Get("user_id").(string))
	if userID == "" {
		log.Printf("[WARN] Deprecated behavior: resolving user_id from username for britive_tag_member %s/%s. Set user_id explicitly to reduce API calls and avoid this fallback in a future release.", tagID, username)
		user, err := c.GetUserByName(username)
		if err != nil {
			return diag.FromErr(err)
		}
		userID = user.UserID
	}

	log.Printf("[INFO] Creating new tag member: %s/%s", tagID, userID)
	err := c.CreateTagMember(tagID, userID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new tag member: %s/%s", tagID, userID)
	d.SetId(rtm.helper.generateUniqueID(tagID, userID))

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

	if err := rtm.importHelper.ParseImportID([]string{"tags/(?P<tag_id>[^/]+)/users/(?P<user_id>[^/]+)", "tag-name/(?P<tag_name>[^/]+)/username/(?P<username>[^/]+)", "(?P<tag_name>[^/]+)/(?P<username>[^/]+)"}, d); err != nil {
		return nil, err
	}

	tagID := d.Get("tag_id").(string)
	userID := d.Get("user_id").(string)
	tagName := d.Get("tag_name").(string)
	username := d.Get("username").(string)

	if strings.TrimSpace(tagID) == "" && strings.TrimSpace(tagName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("tag_id or tag_name")
	}

	if strings.TrimSpace(userID) == "" && strings.TrimSpace(username) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("user_id or username")
	}

	if strings.TrimSpace(tagID) == "" {
		log.Printf("[WARN] Deprecated behavior: resolving tag_id from tag_name for imported britive_tag_member %s/%s. Use import format with IDs to reduce API calls.", tagName, username)
		tag, err := c.GetTagByName(tagName)
		if errors.Is(err, britive.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("tag %s", tagName)
		}
		if err != nil {
			return nil, err
		}
		tagID = tag.ID
	}
	if strings.TrimSpace(userID) == "" {
		log.Printf("[WARN] Deprecated behavior: resolving user_id from username for imported britive_tag_member %s/%s. Use import format with IDs to reduce API calls.", tagID, username)
		user, err := c.GetUserByName(username)
		if errors.Is(err, britive.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("member %s", username)
		}
		if err != nil {
			return nil, err
		}
		userID = user.UserID
	}

	log.Printf("[INFO] Importing tag member %s/%s", tagID, userID)

	d.SetId(rtm.helper.generateUniqueID(tagID, userID))
	d.Set("tag_name", "")

	err := rtm.helper.getAndMapModelToResource(tagID, userID, d, m)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Imported tag member %s/%s", tagID, userID)

	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceTagMemberHelper - Resource Tag member helper functions
type ResourceTagMemberHelper struct {
}

//NewResourceTagMemberHelper - Initializes new tag member resource helper
func NewResourceTagMemberHelper() *ResourceTagMemberHelper {
	return &ResourceTagMemberHelper{}
}

//region Tag member Resource helper functions

func (resourceTagMemberHelper *ResourceTagMemberHelper) generateUniqueID(tagID string, userID string) string {
	return fmt.Sprintf("tags/%s/users/%s", tagID, userID)
}

func (resourceTagMemberHelper *ResourceTagMemberHelper) parseUniqueID(ID string) (tagID string, userID string, err error) {
	tagMemberParts := strings.Split(ID, "/")
	if len(tagMemberParts) < 4 {
		err = errs.NewInvalidResourceIDError("tag member", ID)
		return
	}
	tagID = tagMemberParts[1]
	userID = tagMemberParts[3]
	return
}

func (resourceTagMemberHelper *ResourceTagMemberHelper) getAndMapModelToResource(tagID string, userID string, d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	log.Printf("[INFO] Reading tag member %s/%s", tagID, userID)
	u, err := c.GetTagMember(tagID, userID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("member %s in tag %s", userID, tagID)
	}
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
	if err := d.Set("user_id", userID); err != nil {
		return err
	}

	return nil
}

//endregion
