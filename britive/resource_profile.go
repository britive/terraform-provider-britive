package britive

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProfile() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProfileCreate,
		ReadContext:   resourceProfileRead,
		UpdateContext: resourceProfileUpdate,
		DeleteContext: resourceProfileDelete,
		Importer: &schema.ResourceImporter{
			State: resourceProfileStateImporter,
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
				ValidateFunc: durationSchemaValidateFunc,
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
				ValidateFunc: durationSchemaValidateFunc,
				Description:  "The profile expiration notification as duration",
			},
			"extension_duration": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: durationSchemaValidateFunc,
				Description:  "The profile expiration extenstion as duration",
			},
			"extension_limit": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The profile expiration extension repeat limit",
			},
		},
	}
}

func durationSchemaValidateFunc(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	_, err := time.ParseDuration(v)
	if err != nil {
		errs = append(errs, fmt.Errorf("%s must be duration. [e.g 1s, 10m, 2h, 5d], got: %s", key, v))
	}
	return
}

func appendAssociations(associations []britive.ProfileAssociation, associationType string, associationID string) []britive.ProfileAssociation {
	associations = append(associations, britive.ProfileAssociation{
		Type:  associationType,
		Value: associationID,
	})
	return associations
}

func saveProfileAssociations(appContainerID string, profileID string, d *schema.ResourceData, m interface{}) error {
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
	if len(as) == 0 {
		for _, daeg := range appRootEnvironmentGroup.EnvironmentGroups {
			if daeg.ParentID == "" {
				associations = appendAssociations(associations, "EnvironmentGroup", daeg.ID)
				break
			}
		}
	} else {
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
					associations = appendAssociations(associations, associationType, aeg.ID)
					break
				}
			}
		}
	}
	if len(associations) > 0 {
		err = c.SaveProfileAssociations(profileID, associations)
		if err != nil {
			return err
		}
	}
	return nil
}

func mapResourceDataToProfile(d *schema.ResourceData, m interface{}, profile *britive.Profile, isUpdate bool) error {
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

func resourceProfileCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profile := britive.Profile{}

	mapResourceDataToProfile(d, m, &profile, false)

	p, err := c.CreateProfile(profile.AppContainerID, profile)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(generateUniqueID(p.AppContainerID, p.ProfileID))

	saveProfileAssociations(p.AppContainerID, p.ProfileID, d, m)

	resourceProfileRead(ctx, d, m)

	return diags
}

func generateUniqueID(appContainerID string, profileID string) string {
	return fmt.Sprintf("apps/%s/paps/%s", appContainerID, profileID)
}

func parseUniqueID(ID string) (appContainerID string, profileID string, err error) {
	idParts := strings.Split(ID, "/")
	if len(idParts) < 4 {
		return "", "", fmt.Errorf("Invalid application profile ID reference, please check the state for %s", ID)
	}
	appContainerID = idParts[1]
	profileID = idParts[3]
	return
}

func resourceProfileRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := getAndSetProfileToState(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func getAndSetProfileToState(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	ID := d.Id()

	appContainerID, profileID, err := parseUniqueID(ID)
	if err != nil {
		return err
	}

	profile, err := c.GetProfile(appContainerID, profileID)
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
	associations, err := flattenProfileAssociations(appContainerID, profile.Associations, m)
	if err != nil {
		return err
	}
	if err := d.Set("associations", associations); err != nil {
		return err
	}
	return nil
}

func resourceProfileUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("name") ||
		d.HasChange("description") ||
		d.HasChange("associations") ||
		d.HasChange("expiration_duration") ||
		d.HasChange("extendable") ||
		d.HasChange("notification_prior_to_expiration") ||
		d.HasChange("extension_duration") ||
		d.HasChange("extension_limit") {
		c := m.(*britive.Client)
		appContainerID, profileID, err := parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		profile := britive.Profile{}
		mapResourceDataToProfile(d, m, &profile, true)
		_, err = c.UpdateProfile(appContainerID, profileID, profile)
		if err != nil {
			return diag.FromErr(err)
		}
		saveProfileAssociations(appContainerID, profileID, d, m)
		return resourceProfileRead(ctx, d, m)
	}
	return nil
}

func resourceProfileDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	appContainerID, profileID, err := parseUniqueID(d.Id())
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

func resourceProfileStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := parseImportID([]string{"apps/(?P<app_container_id>[^/]+)/paps/(?P<profile_id>[^/]+)", "(?P<app_container_id>[^/]+)/(?P<profile_id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	d.SetId(generateUniqueID(d.Get("app_container_id").(string), d.Get("profile_id").(string)))
	err := getAndSetProfileToState(d, m)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func flattenProfileAssociations(appContainerID string, associations []britive.ProfileAssociation, m interface{}) ([]interface{}, error) {
	c := m.(*britive.Client)
	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(appContainerID)
	if err != nil {
		return nil, err
	}

	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return make([]interface{}, 0), nil
	}
	profileAssociations := make([]interface{}, len(associations), len(associations))
	for i, association := range associations {
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
		profileAssociations[i] = profileAssociation
	}
	return profileAssociations, nil

}
