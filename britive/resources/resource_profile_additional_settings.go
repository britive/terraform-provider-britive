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

// ResourceProfileAdditionalSettings - Terraform Resource for Profile Additional Settings
type ResourceProfileAdditionalSettings struct {
	Resource     *schema.Resource
	helper       *ResourceProfileAdditionalSettingsHelper
	importHelper *imports.ImportHelper
}

// NewResourceProfileAdditionalSettings - Initialization of new profile additional settings resource
func NewResourceProfileAdditionalSettings(importHelper *imports.ImportHelper) *ResourceProfileAdditionalSettings {
	rpas := &ResourceProfileAdditionalSettings{
		helper:       NewResourceProfileAdditionalSettingsHelper(),
		importHelper: importHelper,
	}
	rpas.Resource = &schema.Resource{
		CreateContext: rpas.resourceCreate,
		ReadContext:   rpas.resourceRead,
		UpdateContext: rpas.resourceUpdate,
		DeleteContext: rpas.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rpas.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"use_app_credential_type": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Inherit the credential type settings from the application",
			},
			"console_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Provide the console access for the profile, overriden if use_app_credential_type is set to true",
			},
			"programmatic_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Provide the programmatic access for the profile, overriden if use_app_credential_type is set to true",
			},
			"project_id_for_service_account": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The project id for creating service accounts",
			},
		},
	}
	return rpas
}

//region Profile Additional Settings Resource Context Operations

func (rpas *ResourceProfileAdditionalSettings) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	profileAdditionalSettings := britive.ProfileAdditionalSettings{}

	err := rpas.helper.mapResourceToModel(d, m, &profileAdditionalSettings, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new profile additional settings: %#v", profileAdditionalSettings)

	pas, err := c.UpdateProfileAdditionalSettings(profileAdditionalSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile additional settings: %#v", pas)
	d.SetId(rpas.helper.generateUniqueID(profileAdditionalSettings.ProfileID))
	rpas.resourceRead(ctx, d, m)
	return diags
}

func (rpas *ResourceProfileAdditionalSettings) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	c := m.(*britive.Client)

	profileID, err := rpas.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading profile additional settings: %s", profileID)

	profileAdditionalSettings, err := c.GetProfileAdditionalSettings(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("profile additional settings for %s", profileID))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	err = rpas.helper.mapModelToResource(d, m, false, profileAdditionalSettings, profileID)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rpas *ResourceProfileAdditionalSettings) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var hasChanges bool
	if d.HasChange("profile_id") || d.HasChange("use_app_credential_type") || d.HasChange("console_access") || d.HasChange("programmatic_access") || d.HasChange("project_id_for_service_account") {
		hasChanges = true
		profileID, err := rpas.helper.parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		profileAdditionalSettings := britive.ProfileAdditionalSettings{}

		err = rpas.helper.mapResourceToModel(d, m, &profileAdditionalSettings, false)
		if err != nil {
			return diag.FromErr(err)
		}

		profileAdditionalSettings.ProfileID = profileID

		upas, err := c.UpdateProfileAdditionalSettings(profileAdditionalSettings)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated profile additional settings: %#v", upas)
		d.SetId(rpas.helper.generateUniqueID(profileAdditionalSettings.ProfileID))
	}
	if hasChanges {
		return rpas.resourceRead(ctx, d, m)
	}
	return nil
}

func (rpas *ResourceProfileAdditionalSettings) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, err := rpas.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	profileAdditionalSettings := britive.ProfileAdditionalSettings{}

	err = rpas.helper.mapResourceToModel(d, m, &profileAdditionalSettings, true)
	if err != nil {
		return diag.FromErr(err)
	}

	profileAdditionalSettings.ProfileID = profileID

	log.Printf("[INFO] Deleting profile additional settings for %s", profileID)

	_, err = c.UpdateProfileAdditionalSettings(profileAdditionalSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted profile additional settings for %s", profileID)
	d.SetId("")

	return diags
}

func (rpas *ResourceProfileAdditionalSettings) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpas.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/additional-settings"}, d); err != nil {
		return nil, err
	}

	profileID := d.Get("profile_id").(string)
	if strings.TrimSpace(profileID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_id")
	}

	log.Printf("[INFO] Importing profile additional settings for %s", profileID)

	profileAdditionalSettings, err := c.GetProfileAdditionalSettings(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("profile additional settings for %s", profileID)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rpas.helper.generateUniqueID(profileID))

	err = rpas.helper.mapModelToResource(d, m, true, profileAdditionalSettings, profileID)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported profile additional settings for %s", profileID)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceProfileAdditionalSettingsHelper - Terraform Resource for Profile Additional Settings Helper
type ResourceProfileAdditionalSettingsHelper struct {
}

// NewResourceProfileAdditionalSettingsHelper - Initialization of new profile additional settings resource helper
func NewResourceProfileAdditionalSettingsHelper() *ResourceProfileAdditionalSettingsHelper {
	return &ResourceProfileAdditionalSettingsHelper{}
}

//region ProfileAdditionalSettings Resource helper functions

func (rpash *ResourceProfileAdditionalSettingsHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, profileAdditionalSettings *britive.ProfileAdditionalSettings, isDelete bool) error {
	profileAdditionalSettings.ProfileID = d.Get("profile_id").(string)
	projectId, isProjectIdSet := d.GetOk("project_id_for_service_account")
	if isDelete {
		profileAdditionalSettings.UseApplicationCredentialType = true
		profileAdditionalSettings.ConsoleAccess = false
		profileAdditionalSettings.ProgrammaticAccess = false
		if isProjectIdSet == true {
			profileAdditionalSettings.ProjectIdForServiceAccount = ""
		}
	} else {
		profileAdditionalSettings.UseApplicationCredentialType = d.Get("use_app_credential_type").(bool)
		profileAdditionalSettings.ConsoleAccess = d.Get("console_access").(bool)
		profileAdditionalSettings.ProgrammaticAccess = d.Get("programmatic_access").(bool)
		if isProjectIdSet == true {
			profileAdditionalSettings.ProjectIdForServiceAccount = projectId.(string)
		}
	}

	return nil
}

func (rpash *ResourceProfileAdditionalSettingsHelper) mapModelToResource(d *schema.ResourceData, m interface{}, isImport bool, profileAdditionalSettings *britive.ProfileAdditionalSettings, profileID string) error {

	log.Printf("[INFO] Received profile additional settings: %#v", profileAdditionalSettings)

	if err := d.Set("profile_id", profileID); err != nil {
		return err
	}
	if err := d.Set("use_app_credential_type", profileAdditionalSettings.UseApplicationCredentialType); err != nil {
		return err
	}
	if err := d.Set("console_access", profileAdditionalSettings.ConsoleAccess); err != nil {
		return err
	}
	if err := d.Set("programmatic_access", profileAdditionalSettings.ProgrammaticAccess); err != nil {
		return err
	}
	_, isProjectIdSet := d.GetOk("project_id_for_service_account")
	if isProjectIdSet || isImport {
		if err := d.Set("project_id_for_service_account", profileAdditionalSettings.ProjectIdForServiceAccount); err != nil {
			return err
		}
	}

	return nil
}

func (resourceProfileAdditionalSettingsHelper *ResourceProfileAdditionalSettingsHelper) generateUniqueID(profileID string) string {
	return fmt.Sprintf("paps/%s/additional-settings", profileID)
}

func (resourceProfileAdditionalSettingsHelper *ResourceProfileAdditionalSettingsHelper) parseUniqueID(ID string) (profileID string, err error) {
	profileAdditionalSettings := strings.Split(ID, "/")
	if len(profileAdditionalSettings) < 3 {
		err = errs.NewInvalidResourceIDError("profile additional settings", ID)
		return
	}

	profileID = profileAdditionalSettings[1]
	return
}

//endregion
