package britive

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//ResourceProfileIdentity - Terraform Resource for Profile Identity
type ResourceProfileIdentity struct {
	Resource *schema.Resource
	helper   *ResourceProfileIdentityHelper
}

//NewResourceProfileIdentity - Initialisation of new profile identity resource
func NewResourceProfileIdentity() *ResourceProfileIdentity {
	rpt := &ResourceProfileIdentity{
		helper: NewResourceProfileIdentityHelper(),
	}
	rpt.Resource = &schema.Resource{
		CreateContext: rpt.resourceCreate,
		ReadContext:   rpt.resourceRead,
		UpdateContext: rpt.resourceUpdate,
		DeleteContext: rpt.resourceDelete,
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
				Description: "The identity associate with the profile",
			},
			"access_period": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "The access period for the associated identity",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
							Description:  "The start of the access period for the associated identity",
						},
						"end": &schema.Schema{
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

	_, err = c.CreateProfileIdentity(*profileIdentity)
	if err != nil {
		return diag.FromErr(err)
	}

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

	pt, err := c.GetProfileIdentity(profileID, userID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(rpt.helper.generateUniqueID(profileID, userID))

	if pt.AccessPeriod != nil {
		accessPeriods := make([]interface{}, 1, 1)
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

	_, err = c.UpdateProfileIdentity(*profileIdentity)
	if err != nil {
		return diag.FromErr(err)
	}

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

	err = c.DeleteProfileIdentity(profileID, userID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags
}

//endregion

//ResourceProfileIdentityHelper - Terraform Resource for Profile Identity Helper
type ResourceProfileIdentityHelper struct {
}

//NewResourceProfileIdentityHelper - Initialisation of new profile identity resource helper
func NewResourceProfileIdentityHelper() *ResourceProfileIdentityHelper {
	return &ResourceProfileIdentityHelper{}
}

//region Profile Identity Helper functions

func (rpth *ResourceProfileIdentityHelper) getAndMapResourceToModel(d *schema.ResourceData, m interface{}) (*britive.ProfileIdentity, error) {
	c := m.(*britive.Client)
	profileID := d.Get("profile_id").(string)
	username := d.Get("username").(string)
	identity, err := c.GetUserByName(username)
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
	profileIdentity := britive.ProfileIdentity{
		ProfileID:    profileID,
		UserID:       identity.UserID,
		AccessPeriod: accessPeriod,
	}
	return &profileIdentity, nil
}

func (rpth *ResourceProfileIdentityHelper) generateUniqueID(profileID string, userID string) string {
	return fmt.Sprintf("paps/%s/users/%s", profileID, userID)
}

func (rpth *ResourceProfileIdentityHelper) parseUniqueID(ID string) (profileID string, userID string, err error) {
	profileIdentityParts := strings.Split(ID, "/")
	if len(profileIdentityParts) < 4 {
		err = fmt.Errorf("Invalid profile identity reference, please check the state for %s", ID)
		return
	}
	profileID = profileIdentityParts[1]
	userID = profileIdentityParts[3]
	return
}

//endregion
