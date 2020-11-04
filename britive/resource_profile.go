package britive

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceProfile - Terraform Resource for Profile
type ResourceProfile struct {
	Resource     *schema.Resource
	helper       *ResourceProfileHelper
	validation   *Validation
	importHelper *ImportHelper
}

//NewResourceProfile - Initialisation of new profile resource
func NewResourceProfile(validation *Validation, importHelper *ImportHelper) *ResourceProfile {
	rp := &ResourceProfile{
		helper:       NewResourceProfileHelper(),
		validation:   validation,
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
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The application id to associate the profile",
			},
			"profile_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Description: "The id of the profile",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the profile",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the profile",
			},
			"status": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The status of the profile",
			},
			"associations": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Associations for the profile",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of association",
						},
						"value": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The value of association",
						},
					},
				},
			},
			"expiration_duration": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The profile expiration time out as duration",
			},
			"extendable": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The flag whether profile expiration is extendable or not",
			},
			"notification_prior_to_expiration": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The profile expiration notification as duration",
			},
			"extension_duration": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The profile expiration extenstion as duration",
			},
			"extension_limit": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The profile expiration extension repeat limit",
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

	rp.helper.mapResourceToModel(d, m, &profile, false)

	log.Printf("[INFO] Creating new profile: %#v", profile)

	p, err := c.CreateProfile(profile.AppContainerID, profile)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile: %#v", p)
	d.SetId(rp.helper.generateUniqueID(p.AppContainerID, p.ProfileID))

	rp.helper.saveProfileAssociations(p.AppContainerID, p.ProfileID, d, m)

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
	if d.HasChange("name") ||
		d.HasChange("description") ||
		d.HasChange("associations") ||
		d.HasChange("expiration_duration") ||
		d.HasChange("extendable") ||
		d.HasChange("notification_prior_to_expiration") ||
		d.HasChange("extension_duration") ||
		d.HasChange("extension_limit") {
		c := m.(*britive.Client)
		appContainerID, profileID, err := rp.helper.parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		profile := britive.Profile{}
		rp.helper.mapResourceToModel(d, m, &profile, true)
		_, err = c.UpdateProfile(appContainerID, profileID, profile)
		if err != nil {
			return diag.FromErr(err)
		}
		rp.helper.saveProfileAssociations(appContainerID, profileID, d, m)
		return rp.resourceRead(ctx, d, m)
	}
	return nil
}

func (rp *ResourceProfile) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	appContainerID, profileID, err := rp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteProfile(appContainerID, profileID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags
}

func (rp *ResourceProfile) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := rp.importHelper.ParseImportID([]string{"apps/(?P<app_container_id>[^/]+)/paps/(?P<profile_id>[^/]+)", "(?P<app_container_id>[^/]+)/(?P<profile_id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appContainerID := d.Get("app_container_id").(string)
	profileID := d.Get("profile_id").(string)
	d.SetId(rp.helper.generateUniqueID(appContainerID, profileID))
	err := rp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceProfileHelper - Resource Profile helper functions
type ResourceProfileHelper struct {
	Resource *schema.Resource
}

//NewResourceProfileHelper - Initialisation of new profile resource helper
func NewResourceProfileHelper() *ResourceProfileHelper {
	return &ResourceProfileHelper{}
}

//region Profile Helper functions

func (rph *ResourceProfileHelper) generateUniqueID(appContainerID string, profileID string) string {
	return fmt.Sprintf("apps/%s/paps/%s", appContainerID, profileID)
}

func (rph *ResourceProfileHelper) parseUniqueID(ID string) (appContainerID string, profileID string, err error) {
	idParts := strings.Split(ID, "/")
	if len(idParts) < 4 {
		return "", "", fmt.Errorf("Invalid application profile ID reference, please check the state for %s", ID)
	}
	appContainerID = idParts[1]
	profileID = idParts[3]
	return
}

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
	// if len(as) == 0 {
	// 	for _, daeg := range appRootEnvironmentGroup.EnvironmentGroups {
	// 		if daeg.ParentID == "" {
	// 			associations = rph.appendProfileAssociations(associations, "EnvironmentGroup", daeg.ID)
	// 			break
	// 		}
	// 	}
	// } else {
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
	// }
	if len(associations) > 0 {
		err = c.SaveProfileAssociations(profileID, associations)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rph *ResourceProfileHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, profile *britive.Profile, isUpdate bool) error {
	profile.AppContainerID = d.Get("app_container_id").(string)
	profile.Name = d.Get("name").(string)
	profile.Description = d.Get("description").(string)
	if !isUpdate {
		profile.Status = d.Get("status").(string)
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
			return fmt.Errorf("Mising required variable notification_prior_to_expiration")
		}
		notificationPriorToExpiration, err := time.ParseDuration(notificationPriorToExpirationString)
		if err != nil {
			return err
		}
		nullableNotificationPriorToExpiration := int64(notificationPriorToExpiration / time.Millisecond)
		profile.NotificationPriorToExpiration = &nullableNotificationPriorToExpiration

		extensionDurationString := d.Get("extension_duration").(string)
		if extensionDurationString == "" {
			return fmt.Errorf("Mising required variable extension_duration")
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

	ID := d.Id()

	appContainerID, profileID, err := rph.parseUniqueID(ID)
	if err != nil {
		return err
	}

	profile, err := c.GetProfile(profileID)
	if err != nil {
		return err
	}
	if err := d.Set("app_container_id", profile.AppContainerID); err != nil {
		return err
	}
	if err := d.Set("profile_id", profile.ProfileID); err != nil {
		return err
	}
	if err := d.Set("name", profile.Name); err != nil {
		return err
	}
	if err := d.Set("description", profile.Description); err != nil {
		return err
	}
	if err := d.Set("status", profile.Status); err != nil {
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
	associations, err := rph.mapProfileAssociationsModelToResource(appContainerID, profile.Associations, m)
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
			return nil, fmt.Errorf("Unable to get association related to ID %s in root environment", association.Value)
		}
		profileAssociation := make(map[string]interface{})
		profileAssociation["type"] = association.Type
		profileAssociation["value"] = a.Name
		profileAssociations = append(profileAssociations, profileAssociation)
	}
	return profileAssociations, nil

}

//endregion
