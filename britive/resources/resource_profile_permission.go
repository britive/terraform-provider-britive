package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceProfilePermission - Terraform Resource for Profile Permission
type ResourceProfilePermission struct {
	Resource     *schema.Resource
	helper       *ResourceProfilePermissionHelper
	importHelper *imports.ImportHelper
}

// NewResourceProfilePermission - Initialization of new profile permission resource
func NewResourceProfilePermission(importHelper *imports.ImportHelper) *ResourceProfilePermission {
	rpp := &ResourceProfilePermission{
		helper:       NewResourceProfilePermissionHelper(),
		importHelper: importHelper,
	}
	rpp.Resource = &schema.Resource{
		CreateContext: rpp.resourceCreate,
		ReadContext:   rpp.resourceRead,
		DeleteContext: rpp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rpp.resourceStateImporter,
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
			"permission_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of permission",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permission_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The type of permission",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
	return rpp
}

func (rpp *ResourceProfilePermission) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID := d.Get("profile_id").(string)

	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation: "add",
		Permission: britive.ProfilePermission{
			ProfileID: profileID,
			Name:      d.Get("permission_name").(string),
			Type:      d.Get("permission_type").(string),
		},
	}

	log.Printf("[INFO] Creating new profile permission:  %s, %#v", profileID, profilePermissionRequest)

	err := c.ExecuteProfilePermissionRequest(profileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile permission:  %s, %#v", profileID, profilePermissionRequest)

	d.SetId(rpp.helper.generateUniqueID(profilePermissionRequest.Permission))

	return diags
}

func (rpp *ResourceProfilePermission) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profilePermission, err := rpp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading profile permission:  %s, %#v", profilePermission.ProfileID, *profilePermission)

	pp, err := c.GetProfilePermission(profilePermission.ProfileID, *profilePermission)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("permission %s of type %s in profile %s", profilePermission.Name, profilePermission.Type, profilePermission.ProfileID))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading profile permission:  %s, %#v", profilePermission.ProfileID, pp)

	d.SetId(rpp.helper.generateUniqueID(*pp))

	return diags
}

func (rpp *ResourceProfilePermission) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
	profilePermission, err := rpp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation:  "remove",
		Permission: *profilePermission,
	}

	log.Printf("[INFO] Deleting profile permission: %s, %#v", profilePermission.ProfileID, profilePermissionRequest)

	err = c.ExecuteProfilePermissionRequest(profilePermission.ProfileID, profilePermissionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted profile permission: %s, %#v", profilePermission.ProfileID, profilePermissionRequest)

	d.SetId("")

	return diags
}

func (rpp *ResourceProfilePermission) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpp.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/permissions/(?P<permission_name>.+)/type/(?P<permission_type>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<permission_name>.+)/(?P<permission_type>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appName := d.Get("app_name").(string)
	profileName := d.Get("profile_name").(string)
	permissionName := d.Get("permission_name").(string)
	permissionType := d.Get("permission_type").(string)
	if strings.TrimSpace(appName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("app_name")
	}
	if strings.TrimSpace(profileName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_name")
	}
	if strings.TrimSpace(permissionName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_name")
	}
	if strings.TrimSpace(permissionType) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_type")
	}

	log.Printf("[INFO] Importing profile permission: %s/%s/%s/%s", appName, profileName, permissionName, permissionType)

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

	profilePermission, err := c.GetProfilePermission(profile.ProfileID, britive.ProfilePermission{Name: permissionName, Type: permissionType})
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("permission %s of type %s in profile %s", permissionName, permissionType, profileName)
	}
	if err != nil {
		return nil, err
	}
	profilePermission.ProfileID = profile.ProfileID
	d.SetId(rpp.helper.generateUniqueID(*profilePermission))
	d.Set("profile_id", profile.ProfileID)

	d.Set("app_name", "")
	d.Set("profile_name", "")

	log.Printf("[INFO] Imported profile permission: %s/%s/%s/%s", appName, profileName, permissionName, permissionType)
	return []*schema.ResourceData{d}, nil
}

// ResourceProfilePermissionHelper - Terraform Resource for Profile Permission Helper
type ResourceProfilePermissionHelper struct {
}

// NewResourceProfilePermissionHelper - Initialization of new profile tag resource helper
func NewResourceProfilePermissionHelper() *ResourceProfilePermissionHelper {
	return &ResourceProfilePermissionHelper{}
}

func (resourceProfilePermissionHelper *ResourceProfilePermissionHelper) generateUniqueID(profilePermission britive.ProfilePermission) string {
	return fmt.Sprintf("paps/%s/permissions/%s/type/%s", profilePermission.ProfileID, profilePermission.Name, profilePermission.Type)
}

func (resourceProfilePermissionHelper *ResourceProfilePermissionHelper) parseUniqueID(ID string) (*britive.ProfilePermission, error) {
	idFormat := "paps/([^/]+)/permissions/(.+)/type/([^/]+)"

	re, err := regexp.Compile(idFormat)
	if err != nil {
		return nil, err
	}

	fieldValues := re.FindStringSubmatch(ID)
	if fieldValues != nil {
		profilePermission := &britive.ProfilePermission{
			ProfileID: fieldValues[1],
			Name:      fieldValues[2],
			Type:      fieldValues[3],
		}
		return profilePermission, nil
	} else {
		return nil, errs.NewInvalidResourceIDError("profile permission", ID)
	}
}
