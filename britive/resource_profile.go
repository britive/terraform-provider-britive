package britive

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//ResourceProfile - Terraform Resource for Profile
type ResourceProfile struct {
	Resource     *schema.Resource
	helper       *ResourceProfileHelper
	validation   *Validation
	importHelper *ImportHelper
}

//NewResourceProfile - Initialization of new profile resource
func NewResourceProfile(v *Validation, importHelper *ImportHelper) *ResourceProfile {
	rp := &ResourceProfile{
		helper:       NewResourceProfileHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rp.Resource = &schema.Resource{
		CreateContext: rp.resourceCreate,
		ReadContext:   rp.resourceRead,
		UpdateContext: rp.resourceUpdate,
		DeleteContext: rp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rp.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"app_container_id": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identity of the Britive application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"app_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the Britive application",
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the Britive profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive profile",
			},
			"disabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "To disable the Britive profile",
			},
			"associations": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: "The list of associations for the Britive profile",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The type of association, can be any one from the list Environment, EnvironmentGroup, ApplicationResource",
							ValidateFunc: validation.StringInSlice([]string{"Environment", "EnvironmentGroup", "ApplicationResource"}, false),
						},
						"value": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The association value",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"resource_parent_name": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The parent name of resource. Required only if the association type is ApplicationResource",
						},
					},
				},
			},
			"expiration_duration": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The expiration time for the Britive profile",
			},
			"extendable": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The Boolean flag that indicates whether profile expiry is extendable or not",
			},
			"notification_prior_to_expiration": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "he profile expiry notification as a time value",
			},
			"extension_duration": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The profile expiry extension as a time value",
			},
			"extension_limit": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The repetition limit for extending the profile expiry",
			},
		},
	}
	return rp
}

//region Profile Resource Context Operations

func (rp *ResourceProfile) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profile := britive.Profile{}

	err := rp.helper.mapResourceToModel(d, m, &profile, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new profile: %#v", profile)

	p, err := c.CreateProfile(profile.AppContainerID, profile)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile: %#v", p)
	d.SetId(p.ProfileID)

	err = rp.helper.saveProfileAssociations(p.AppContainerID, p.ProfileID, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	rp.resourceRead(ctx, d, m)

	return diags
}

func (rp *ResourceProfile) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rp.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rp *ResourceProfile) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	profileID := d.Id()
	appContainerID := d.Get("app_container_id").(string)

	var hasChanges bool
	if d.HasChange("name") ||
		d.HasChange("description") ||
		d.HasChange("associations") ||
		d.HasChange("expiration_duration") ||
		d.HasChange("extendable") ||
		d.HasChange("notification_prior_to_expiration") ||
		d.HasChange("extension_duration") ||
		d.HasChange("extension_limit") {

		hasChanges = true

		profile := britive.Profile{}
		err := rp.helper.mapResourceToModel(d, m, &profile, true)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updating profile: %#v", profile)

		up, err := c.UpdateProfile(appContainerID, profileID, profile)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated profile: %#v", up)

		err = rp.helper.saveProfileAssociations(appContainerID, profileID, d, m)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("disabled") {

		hasChanges = true
		disabled := d.Get("disabled").(bool)

		log.Printf("[INFO] Updating status disabled: %t of profile: %s", disabled, profileID)
		up, err := c.EnableOrDisableProfile(appContainerID, profileID, disabled)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated status of profile: %#v", up)
	}
	if hasChanges {
		return rp.resourceRead(ctx, d, m)
	}
	return nil
}

func (rp *ResourceProfile) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID := d.Id()
	appContainerID := d.Get("app_container_id").(string)

	log.Printf("[INFO] Deleting profile: %s/%s", appContainerID, profileID)

	err := c.DeleteProfile(appContainerID, profileID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted profile: %s/%s", appContainerID, profileID)
	d.SetId("")

	return diags
}

func (rp *ResourceProfile) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rp.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<name>[^/]+)", "(?P<app_name>[^/]+)/(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appName := d.Get("app_name").(string)
	profileName := d.Get("name").(string)
	if strings.TrimSpace(appName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("app_name")
	}
	if strings.TrimSpace(profileName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing profile: %s/%s", appName, profileName)

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

	d.SetId(profile.ProfileID)
	d.Set("app_name", "")

	err = rp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported profile: %s/%s", appName, profileName)
	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceProfileHelper - Resource Profile helper functions
type ResourceProfileHelper struct {
	Resource *schema.Resource
}

//NewResourceProfileHelper - Initialization of new profile resource helper
func NewResourceProfileHelper() *ResourceProfileHelper {
	return &ResourceProfileHelper{}
}

//region Profile Helper functions

func (rph *ResourceProfileHelper) appendProfileAssociations(associations []britive.ProfileAssociation, associationType string, associationID string) []britive.ProfileAssociation {
	associations = append(associations, britive.ProfileAssociation{
		Type:  associationType,
		Value: associationID,
	})
	return associations
}

func (rph *ResourceProfileHelper) saveProfileAssociations(appContainerID string, profileID string, d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)
	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(appContainerID)
	if err != nil {
		return err
	}
	if appRootEnvironmentGroup == nil {
		return nil
	}
	associations := make([]britive.ProfileAssociation, 0)
	as := d.Get("associations").([]interface{})
	for _, a := range as {
		s := a.(map[string]interface{})
		associationType := s["type"].(string)
		associationName := s["value"].(string)
		var rootAssociations []britive.Association
		if associationType == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}
		for _, aeg := range rootAssociations {
			if aeg.Name == associationName {
				associations = rph.appendProfileAssociations(associations, associationType, aeg.ID)
				break
			}
		}
	}
	if len(associations) > 0 {
		log.Printf("[INFO] Updating profile %s associations: %#v", profileID, associations)
		err = c.SaveProfileAssociations(profileID, associations)
		if err != nil {
			return err
		}
		log.Printf("[INFO] Submitted Update profile %s associations: %#v", profileID, associations)
	} else {
		return NewNotFoundErrorf("associations %v", as)
	}
	return nil
}

func (rph *ResourceProfileHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, profile *britive.Profile, isUpdate bool) error {
	profile.AppContainerID = d.Get("app_container_id").(string)
	profile.Name = d.Get("name").(string)
	profile.Description = d.Get("description").(string)
	if !isUpdate {
		if d.Get("disabled").(bool) {
			profile.Status = "inactive"
		} else {
			profile.Status = "active"
		}
	}
	expirationDuration, err := time.ParseDuration(d.Get("expiration_duration").(string))
	if err != nil {
		return err
	}
	profile.ExpirationDuration = int64(expirationDuration / time.Millisecond)
	extendable := d.Get("extendable").(bool)
	if extendable {
		profile.Extendable = extendable
		notificationPriorToExpirationString := d.Get("notification_prior_to_expiration").(string)
		if notificationPriorToExpirationString == "" {
			return NewNotEmptyOrWhiteSpaceError("notification_prior_to_expiration")
		}
		notificationPriorToExpiration, err := time.ParseDuration(notificationPriorToExpirationString)
		if err != nil {
			return err
		}
		nullableNotificationPriorToExpiration := int64(notificationPriorToExpiration / time.Millisecond)
		profile.NotificationPriorToExpiration = &nullableNotificationPriorToExpiration

		extensionDurationString := d.Get("extension_duration").(string)
		if extensionDurationString == "" {
			return NewNotEmptyOrWhiteSpaceError("extension_duration")
		}
		extensionDuration, err := time.ParseDuration(extensionDurationString)
		if err != nil {
			return err
		}
		nullableExtensionDuration := int64(extensionDuration / time.Millisecond)
		profile.ExtensionDuration = &nullableExtensionDuration
		profile.ExtensionLimit = d.Get("extension_limit").(int)
	}

	return nil
}

func (rph *ResourceProfileHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	profileID := d.Id()

	log.Printf("[INFO] Reading profile %s", profileID)

	profile, err := c.GetProfile(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("profile %s", profileID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received profile %#v", profile)

	if err := d.Set("app_container_id", profile.AppContainerID); err != nil {
		return err
	}
	if err := d.Set("name", profile.Name); err != nil {
		return err
	}
	if err := d.Set("description", profile.Description); err != nil {
		return err
	}
	if err := d.Set("disabled", strings.EqualFold(profile.Status, "inactive")); err != nil {
		return err
	}
	if err := d.Set("expiration_duration", time.Duration(profile.ExpirationDuration*int64(time.Millisecond)).String()); err != nil {
		return err
	}
	if err := d.Set("extendable", profile.Extendable); err != nil {
		return err
	}
	if profile.Extendable {
		if profile.NotificationPriorToExpiration != nil {
			notificationPriorToExpiration := *profile.NotificationPriorToExpiration
			if err := d.Set("notification_prior_to_expiration", time.Duration(notificationPriorToExpiration*int64(time.Millisecond)).String()); err != nil {
				return err
			}
		}
		if profile.ExtensionDuration != nil {
			extensionDuration := *profile.ExtensionDuration
			if err := d.Set("extension_duration", time.Duration(extensionDuration*int64(time.Millisecond)).String()); err != nil {
				return err
			}
		}
		if err := d.Set("extension_limit", profile.ExtensionLimit); err != nil {
			return err
		}
	}
	associations, err := rph.mapProfileAssociationsModelToResource(profile.AppContainerID, profile.Associations, m)
	if err != nil {
		return err
	}
	if err := d.Set("associations", associations); err != nil {
		return err
	}
	return nil
}

func (rph *ResourceProfileHelper) mapProfileAssociationsModelToResource(appContainerID string, associations []britive.ProfileAssociation, m interface{}) ([]interface{}, error) {
	c := m.(*britive.Client)
	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(appContainerID)
	if err != nil {
		return nil, err
	}

	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return make([]interface{}, 0), nil
	}
	profileAssociations := make([]interface{}, 0)
	for _, association := range associations {
		var rootAssociations []britive.Association
		if association.Type == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}
		var a *britive.Association
		for _, aeg := range rootAssociations {
			if aeg.ID == association.Value {
				a = &aeg
				break
			}
		}
		if a == nil {
			return nil, NewNotFoundErrorf("association %s", association.Value)
		}
		profileAssociation := make(map[string]interface{})
		profileAssociation["type"] = association.Type
		profileAssociation["value"] = a.Name
		profileAssociations = append(profileAssociations, profileAssociation)
	}
	return profileAssociations, nil

}

//endregion
