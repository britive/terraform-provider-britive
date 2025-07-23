package resources

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceProfile - Terraform Resource for Profile
type ResourceProfile struct {
	Resource     *schema.Resource
	helper       *ResourceProfileHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceProfile - Initialization of new profile resource
func NewResourceProfile(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceProfile {
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
			"app_container_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identity of the Britive application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"app_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the Britive application",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the Britive profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive profile",
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "To disable the Britive profile",
			},
			"associations": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "The list of associations for the Britive profile",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The type of association, should be one of [Environment, EnvironmentGroup, ApplicationResource]",
							ValidateFunc: validation.StringInSlice([]string{"Environment", "EnvironmentGroup", "ApplicationResource"}, false),
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The association value",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"parent_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "The parent name of the resource. Required only if the association type is ApplicationResource",
						},
					},
				},
			},
			"expiration_duration": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The expiration time for the Britive profile",
			},
			"extendable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The Boolean flag that indicates whether profile expiry is extendable or not",
			},
			"notification_prior_to_expiration": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "he profile expiry notification as a time value",
			},
			"extension_duration": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: rp.validation.DurationValidateFunc,
				Description:  "The profile expiry extension as a time value",
			},
			"extension_limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The repetition limit for extending the profile expiry",
			},
			"destination_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The destination url to redirect user after checkout",
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
		d.HasChange("extension_limit") ||
		d.HasChange("destination_url") {

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
		return nil, errs.NewNotEmptyOrWhiteSpaceError("app_name")
	}
	if strings.TrimSpace(profileName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing profile: %s/%s", appName, profileName)

	app, err := c.GetApplicationByName(appName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("application %s", appName)
	}
	if err != nil {
		return nil, err
	}
	profile, err := c.GetProfileByName(app.AppContainerID, profileName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("profile %s", profileName)
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

// ResourceProfileHelper - Resource Profile helper functions
type ResourceProfileHelper struct {
	Resource *schema.Resource
}

// NewResourceProfileHelper - Initialization of new profile resource helper
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
	applicationType, err := c.GetApplicationType(appContainerID)
	if err != nil {
		return err
	}
	appType := applicationType.ApplicationType
	associationScopes := make([]britive.ProfileAssociation, 0)
	associationResources := make([]britive.ProfileAssociation, 0)
	as := d.Get("associations").(*schema.Set)
	unmatchedAssociations := make([]interface{}, 0)
	for _, a := range as.List() {
		s := a.(map[string]interface{})
		associationType := s["type"].(string)
		associationValue := s["value"].(string)
		var rootAssociations []britive.Association
		isAssociationExists := false
		switch associationType {
		case "EnvironmentGroup", "Environment":
			if associationType == "EnvironmentGroup" {
				rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
				if appType == "AWS" && strings.EqualFold("root", associationValue) {
					associationValue = "Root"
				} else if appType == "AWS Standalone" && strings.EqualFold("root", associationValue) {
					associationValue = "root"
				}
			} else {
				rootAssociations = appRootEnvironmentGroup.Environments
			}
			for _, aeg := range rootAssociations {
				if aeg.Name == associationValue || aeg.ID == associationValue {
					isAssociationExists = true
					associationScopes = rph.appendProfileAssociations(associationScopes, associationType, aeg.ID)
					break
				} else if associationType == "Environment" && appType == "AWS Standalone" {
					newAssociationValue := c.GetEnvId(appContainerID, associationValue)
					if aeg.ID == newAssociationValue {
						isAssociationExists = true
						associationScopes = rph.appendProfileAssociations(associationScopes, associationType, aeg.ID)
						break
					}
				}
			}
		case "ApplicationResource":
			associationParentName := s["parent_name"].(string)
			if strings.TrimSpace(associationParentName) == "" {
				return errs.NewNotEmptyOrWhiteSpaceError("associations.parent_name")
			}
			r, err := c.GetProfileAssociationResource(profileID, associationValue, associationParentName)
			if errors.Is(err, britive.ErrNotFound) {
				isAssociationExists = false
			} else if err != nil {
				return err
			} else if r != nil {
				isAssociationExists = true
				associationResources = rph.appendProfileAssociations(associationResources, associationType, r.NativeID)
			}

		}
		if !isAssociationExists {
			unmatchedAssociations = append(unmatchedAssociations, s)
		}

	}
	if len(unmatchedAssociations) > 0 {
		return errs.NewNotFoundErrorf("associations %v", unmatchedAssociations)
	}
	log.Printf("[INFO] Updating profile %s associations: %#v", profileID, associationScopes)
	err = c.SaveProfileAssociationScopes(profileID, associationScopes)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted Update profile %s associations: %#v", profileID, associationScopes)
	if len(associationResources) > 0 {
		log.Printf("[INFO] Updating profile %s association resources: %#v", profileID, associationResources)
		err = c.SaveProfileAssociationResourceScopes(profileID, associationResources)
		if err != nil {
			return err
		}
		log.Printf("[INFO] Submitted Update profile %s association resources: %#v", profileID, associationResources)
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
	profile.DestinationUrl = d.Get("destination_url").(string)
	extendable := d.Get("extendable").(bool)
	if extendable {
		profile.Extendable = extendable
		notificationPriorToExpirationString := d.Get("notification_prior_to_expiration").(string)
		if notificationPriorToExpirationString == "" {
			return errs.NewNotEmptyOrWhiteSpaceError("notification_prior_to_expiration")
		}
		notificationPriorToExpiration, err := time.ParseDuration(notificationPriorToExpirationString)
		if err != nil {
			return err
		}
		nullableNotificationPriorToExpiration := int64(notificationPriorToExpiration / time.Millisecond)
		profile.NotificationPriorToExpiration = &nullableNotificationPriorToExpiration

		extensionDurationString := d.Get("extension_duration").(string)
		if extensionDurationString == "" {
			return errs.NewNotEmptyOrWhiteSpaceError("extension_duration")
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
		return errs.NewNotFoundErrorf("profile %s", profileID)
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
	if err := d.Set("destination_url", profile.DestinationUrl); err != nil {
		return err
	}
	associations, err := rph.mapProfileAssociationsModelToResource(profile.AppContainerID, profile.ProfileID, profile.Associations, d, m)
	if err != nil {
		return err
	}
	if err := d.Set("associations", associations); err != nil {
		return err
	}
	return nil
}

func (rph *ResourceProfileHelper) mapProfileAssociationsModelToResource(appContainerID string, profileID string, associations []britive.ProfileAssociation, d *schema.ResourceData, m interface{}) ([]interface{}, error) {
	c := m.(*britive.Client)
	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(appContainerID)
	if err != nil {
		return nil, err
	}
	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return make([]interface{}, 0), nil
	}
	inputAssociations := d.Get("associations").(*schema.Set)
	applicationType, err := c.GetApplicationType(appContainerID)
	if err != nil {
		return nil, err
	}
	appType := applicationType.ApplicationType
	profileAssociations := make([]interface{}, 0)
	for _, association := range associations {
		var rootAssociations []britive.Association
		switch association.Type {
		case "EnvironmentGroup", "Environment":
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
				return nil, errs.NewNotFoundErrorf("association %s", association.Value)
			}
			profileAssociation := make(map[string]interface{})
			associationValue := a.Name
			for _, inputAssociation := range inputAssociations.List() {
				ia := inputAssociation.(map[string]interface{})
				iat := ia["type"].(string)
				iav := ia["value"].(string)
				if association.Type == "EnvironmentGroup" && (appType == "AWS" || appType == "AWS Standalone") && strings.EqualFold("root", a.Name) && strings.EqualFold("root", iav) {
					associationValue = iav
				}
				if association.Type == iat && a.ID == iav {
					associationValue = a.ID
					break
				} else if association.Type == "Environment" && appType == "AWS Standalone" {
					envId := c.GetEnvId(appContainerID, iav)
					if association.Type == iat && a.ID == envId {
						associationValue = iav
						break
					}
				}
			}
			profileAssociation["type"] = association.Type
			profileAssociation["value"] = associationValue
			profileAssociations = append(profileAssociations, profileAssociation)
		case "ApplicationResource":
			par, err := c.GetProfileAssociationResourceByNativeID(profileID, association.Value)
			if errors.Is(err, britive.ErrNotFound) {
				return nil, errs.NewNotFoundErrorf("application resource %s", association.Value)
			} else if err != nil {
				return nil, err
			} else if par == nil {
				return nil, errs.NewNotFoundErrorf("application resource %s", association.Value)
			}
			profileAssociation := make(map[string]interface{})
			profileAssociation["type"] = association.Type
			profileAssociation["value"] = par.Name
			profileAssociation["parent_name"] = par.ParentName
			profileAssociations = append(profileAssociations, profileAssociation)
		}

	}
	return profileAssociations, nil

}

//endregion
