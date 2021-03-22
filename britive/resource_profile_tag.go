package britive

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//ResourceProfileTag - Terraform Resource for Profile Tag
type ResourceProfileTag struct {
	Resource     *schema.Resource
	helper       *ResourceProfileTagHelper
	importHelper *ImportHelper
}

//NewResourceProfileTag - Initialization of new profile tag resource
func NewResourceProfileTag(importHelper *ImportHelper) *ResourceProfileTag {
	rpt := &ResourceProfileTag{
		helper:       NewResourceProfileTagHelper(),
		importHelper: importHelper,
	}
	rpt.Resource = &schema.Resource{
		CreateContext: rpt.resourceCreate,
		ReadContext:   rpt.resourceRead,
		UpdateContext: rpt.resourceUpdate,
		DeleteContext: rpt.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rpt.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"app_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The application name of the application, profile is assciated with",
			},
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"profile_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the profile",
			},
			"tag_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The tag associate with the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"access_period": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "The access period for the associated tag",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
							Description:  "The start of the access period for the associated tag",
						},
						"end": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
							Description:  "The end of the access period for the associated tag",
						},
					},
				},
			},
		},
	}
	return rpt
}

//region Profile Tag Resource Context Operations

func (rpt *ResourceProfileTag) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	profileTag, err := rpt.helper.getAndMapResourceToModel(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new profile tag: %#v", *profileTag)

	pt, err := c.CreateProfileTag(*profileTag)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile tag: %#v", *pt)

	d.SetId(rpt.helper.generateUniqueID(profileTag.ProfileID, profileTag.TagID))

	return rpt.resourceRead(ctx, d, m)
}

func (rpt *ResourceProfileTag) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	err := rpt.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rpt *ResourceProfileTag) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("access_period") {
		return nil
	}
	c := m.(*britive.Client)
	profileTag, err := rpt.helper.getAndMapResourceToModel(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating profile tag: %#v", *profileTag)

	upt, err := c.UpdateProfileTag(*profileTag)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted Updated profile tag: %#v", upt)

	d.SetId(rpt.helper.generateUniqueID(profileTag.ProfileID, profileTag.TagID))

	return rpt.resourceRead(ctx, d, m)
}

func (rpt *ResourceProfileTag) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, tagID, err := rpt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting profile tag: %s/%s", profileID, tagID)

	err = c.DeleteProfileTag(profileID, tagID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted profile tag: %s/%s", profileID, tagID)

	d.SetId("")

	return diags
}

func (rpt *ResourceProfileTag) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpt.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/tags/(?P<tag_name>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<tag_name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appName := d.Get("app_name").(string)
	profileName := d.Get("profile_name").(string)
	tagName := d.Get("tag_name").(string)
	if strings.TrimSpace(appName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("app_name")
	}
	if strings.TrimSpace(profileName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("profile_name")
	}
	if strings.TrimSpace(tagName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("tag_name")
	}

	log.Printf("[INFO] Importing profile tag: %s/%s/%s", appName, profileName, tagName)

	app, err := c.GetApplicationByName(appName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("application %s", appName)
	}
	if err != nil {
		return nil, err
	}
	profile, err := c.GetProfileByName(app.AppContainerID, profileName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("profile %s", profileName)
	}
	if err != nil {
		return nil, err
	}

	tag, err := c.GetTagByName(tagName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("tag %s", tagName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rpt.helper.generateUniqueID(profile.ProfileID, tag.ID))
	d.Set("app_name", "")
	d.Set("profile_name", "")

	err = rpt.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported profile tag: %s/%s/%s", appName, profileName, tagName)
	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceProfileTagHelper - Terraform Resource for Profile Tag Helper
type ResourceProfileTagHelper struct {
}

//NewResourceProfileTagHelper - Initialization of new profile tag resource helper
func NewResourceProfileTagHelper() *ResourceProfileTagHelper {
	return &ResourceProfileTagHelper{}
}

//region Profile Tag Helper functions

func (resourceProfileTagHelper *ResourceProfileTagHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)
	profileID, tagID, err := resourceProfileTagHelper.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading profile tag: %s/%s", profileID, tagID)

	pt, err := c.GetProfileTag(profileID, tagID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("tag %s in profile %s", tagID, profileID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received profile tag: %#v", pt)

	d.Set("profile_id", profileID)

	if pt.AccessPeriod != nil {
		accessPeriods := make([]interface{}, 1)
		accessPeriod := make(map[string]interface{})
		accessPeriod["start"] = pt.AccessPeriod.Start.Format(time.RFC3339)
		accessPeriod["end"] = pt.AccessPeriod.End.Format(time.RFC3339)
		accessPeriods[0] = accessPeriod
		if err := d.Set("access_period", accessPeriods); err != nil {
			return err
		}
	}
	return nil
}

func (resourceProfileTagHelper *ResourceProfileTagHelper) getAndMapResourceToModel(d *schema.ResourceData, m interface{}) (*britive.ProfileTag, error) {
	c := m.(*britive.Client)
	profileID := d.Get("profile_id").(string)
	tagName := d.Get("tag_name").(string)
	tag, err := c.GetTagByName(tagName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("tag %s", tagName)
	}
	if err != nil {
		return nil, err
	}

	aps := d.Get("access_period").([]interface{})
	var accessPeriod *britive.TimePeriod
	if len(aps) > 0 {
		ap := aps[0].(map[string]interface{})
		startTime, err := time.Parse(time.RFC3339, ap["start"].(string))
		if err != nil {
			return nil, err
		}
		endTime, err := time.Parse(time.RFC3339, ap["end"].(string))
		if err != nil {
			return nil, err
		}
		accessPeriod = &britive.TimePeriod{
			Start: startTime,
			End:   endTime,
		}
	}
	profileTag := britive.ProfileTag{
		ProfileID:    profileID,
		TagID:        tag.ID,
		AccessPeriod: accessPeriod,
	}
	return &profileTag, nil
}

func (resourceProfileTagHelper *ResourceProfileTagHelper) generateUniqueID(profileID string, tagID string) string {
	return fmt.Sprintf("paps/%s/tags/%s", profileID, tagID)
}

func (resourceProfileTagHelper *ResourceProfileTagHelper) parseUniqueID(ID string) (profileID string, tagID string, err error) {
	profileTagParts := strings.Split(ID, "/")
	if len(profileTagParts) < 4 {
		err = NewInvalidResourceIDError("profile tag", ID)
		return
	}
	profileID = profileTagParts[1]
	tagID = profileTagParts[3]
	return
}

//endregion
