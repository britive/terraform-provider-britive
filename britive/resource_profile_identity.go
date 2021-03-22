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

//ResourceProfileIdentity - Terraform Resource for Profile Identity
type ResourceProfileIdentity struct {
	Resource     *schema.Resource
	helper       *ResourceProfileIdentityHelper
	importHelper *ImportHelper
}

//NewResourceProfileIdentity - Initialization of new profile identity resource
func NewResourceProfileIdentity(importHelper *ImportHelper) *ResourceProfileIdentity {
	rpt := &ResourceProfileIdentity{
		helper:       NewResourceProfileIdentityHelper(),
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
				Description: "The name of the application, profile is assciated with",
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
			"username": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identity associate with the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"access_period": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "The access period for the associated identity",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
							Description:  "The start of the access period for the associated identity",
						},
						"end": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
							Description:  "The end of the access period for the associated identity",
						},
					},
				},
			},
		},
	}
	return rpt
}

//region Profile Identity Resource Context Operations

func (rpt *ResourceProfileIdentity) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	profileIdentity, err := rpt.helper.getAndMapResourceToModel(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new profile identity: %#v", *profileIdentity)

	_, err = c.CreateProfileIdentity(*profileIdentity)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile identity: %#v", *profileIdentity)
	d.SetId(rpt.helper.generateUniqueID(profileIdentity.ProfileID, profileIdentity.UserID))

	return rpt.resourceRead(ctx, d, m)
}

func (rpt *ResourceProfileIdentity) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profileID, userID, err := rpt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading profile identity: %s/%s", profileID, userID)

	pt, err := c.GetProfileIdentity(profileID, userID)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(NewNotFoundErrorf("identity %s in profile %s", userID, profileID))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received profile identity: %#v", pt)
	d.SetId(rpt.helper.generateUniqueID(profileID, userID))

	if pt.AccessPeriod != nil {
		accessPeriods := make([]interface{}, 1)
		accessPeriod := make(map[string]interface{})
		accessPeriod["start"] = pt.AccessPeriod.Start.Format(time.RFC3339)
		accessPeriod["end"] = pt.AccessPeriod.End.Format(time.RFC3339)
		accessPeriods[0] = accessPeriod
		if err := d.Set("access_period", accessPeriods); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func (rpt *ResourceProfileIdentity) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("access_period") {
		return nil
	}
	c := m.(*britive.Client)
	profileIdentity, err := rpt.helper.getAndMapResourceToModel(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating profile identity: %#v", *profileIdentity)

	pi, err := c.UpdateProfileIdentity(*profileIdentity)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted updated profile identity: %#v", pi)

	d.SetId(rpt.helper.generateUniqueID(profileIdentity.ProfileID, profileIdentity.UserID))

	return rpt.resourceRead(ctx, d, m)
}

func (rpt *ResourceProfileIdentity) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, userID, err := rpt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting profile identity: %s/%s", profileID, userID)

	err = c.DeleteProfileIdentity(profileID, userID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted profile identity: %s/%s", profileID, userID)
	d.SetId("")

	return diags
}

func (rpt *ResourceProfileIdentity) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpt.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/users/(?P<username>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<username>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appName := d.Get("app_name").(string)
	profileName := d.Get("profile_name").(string)
	username := d.Get("username").(string)
	if strings.TrimSpace(appName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("app_name")
	}
	if strings.TrimSpace(profileName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("profile_name")
	}
	if strings.TrimSpace(username) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("username")
	}

	log.Printf("[INFO] Importing profile identity: %s/%s/%s", appName, profileName, username)

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

	user, err := c.GetUserByName(username)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("identity %s", username)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rpt.helper.generateUniqueID(profile.ProfileID, user.UserID))
	d.Set("profile_id", profile.ProfileID)

	d.Set("app_name", "")
	d.Set("profile_name", "")

	log.Printf("[INFO] Imported profile tag: %s/%s/%s", appName, profileName, username)
	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceProfileIdentityHelper - Terraform Resource for Profile Identity Helper
type ResourceProfileIdentityHelper struct {
}

//NewResourceProfileIdentityHelper - Initialization of new profile identity resource helper
func NewResourceProfileIdentityHelper() *ResourceProfileIdentityHelper {
	return &ResourceProfileIdentityHelper{}
}

//region Profile Identity Helper functions

func (resourceProfileIdentityHelper *ResourceProfileIdentityHelper) getAndMapResourceToModel(d *schema.ResourceData, m interface{}) (*britive.ProfileIdentity, error) {
	c := m.(*britive.Client)
	profileID := d.Get("profile_id").(string)
	username := d.Get("username").(string)
	identity, err := c.GetUserByName(username)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("identity %s", username)
	}
	if err != nil {
		return nil, err
	}

	aps := d.Get("access_period").([]interface{})
	var accessPeriod *britive.TimePeriod
	if len(aps) > 0 {
		ap := aps[0].(map[string]interface{})
		start := ap["start"].(string)
		end := ap["end"].(string)

		if start == "" && end == "" {
			accessPeriod = nil
		} else {
			startTime, err := time.Parse(time.RFC3339, start)
			if err != nil {
				return nil, err
			}
			endTime, err := time.Parse(time.RFC3339, end)
			if err != nil {
				return nil, err
			}
			accessPeriod = &britive.TimePeriod{
				Start: startTime,
				End:   endTime,
			}
		}
	}
	profileIdentity := britive.ProfileIdentity{
		ProfileID:    profileID,
		UserID:       identity.UserID,
		AccessPeriod: accessPeriod,
	}
	return &profileIdentity, nil
}

func (resourceProfileIdentityHelper *ResourceProfileIdentityHelper) generateUniqueID(profileID string, userID string) string {
	return fmt.Sprintf("paps/%s/users/%s", profileID, userID)
}

func (resourceProfileIdentityHelper *ResourceProfileIdentityHelper) parseUniqueID(ID string) (profileID string, userID string, err error) {
	profileIdentityParts := strings.Split(ID, "/")
	if len(profileIdentityParts) < 4 {
		err = NewInvalidResourceIDError("profile identity", ID)
		return
	}
	profileID = profileIdentityParts[1]
	userID = profileIdentityParts[3]
	return
}

//endregion
