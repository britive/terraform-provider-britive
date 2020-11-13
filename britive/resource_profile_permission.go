package britive

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceProfilePermission - Terraform Resource for Profile Permission
type ResourceProfilePermission struct {
	Resource     *schema.Resource
	helper       *ResourceProfilePermissionHelper
	importHelper *ImportHelper
}

//NewResourceProfilePermission - Initialisation of new profile permission resource
func NewResourceProfilePermission(importHelper *ImportHelper) *ResourceProfilePermission {
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
			"app_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The application name of the application, profile is assciated with",
			},
			"profile_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the profile",
			},
			"profile_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the profile",
			},
			"permission_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of permission",
			},
			"permission_type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The type of permission",
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
	if err := rpp.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/permissions/(?P<permission_name>[^/]+)/type/(?P<permission_type>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appName := d.Get("app_name").(string)
	profileName := d.Get("profile_name").(string)
	permissionName := d.Get("permission_name").(string)
	permissionType := d.Get("permission_type").(string)

	log.Printf("[INFO] Importing profile permission: %s/%s/%s/%s", appName, profileName, permissionName, permissionType)

	app, err := c.GetApplicationByName(appName)
	if err != nil {
		return nil, err
	}
	profile, err := c.GetProfileByName(app.AppContainerID, profileName)
	if err != nil {
		return nil, err
	}

	profilePermission, err := c.GetProfilePermission(profile.ProfileID, britive.ProfilePermission{Name: permissionName, Type: permissionType})
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

//ResourceProfilePermissionHelper - Terraform Resource for Profile Permission Helper
type ResourceProfilePermissionHelper struct {
}

//NewResourceProfilePermissionHelper - Initialisation of new profile tag resource helper
func NewResourceProfilePermissionHelper() *ResourceProfilePermissionHelper {
	return &ResourceProfilePermissionHelper{}
}

func (rpph *ResourceProfilePermissionHelper) generateUniqueID(profilePermission britive.ProfilePermission) string {
	return fmt.Sprintf("paps/%s/permissions/%s/type/%s", profilePermission.ProfileID, profilePermission.Name, profilePermission.Type)
}

func (rpph *ResourceProfilePermissionHelper) parseUniqueID(ID string) (*britive.ProfilePermission, error) {
	profileMemberParts := strings.Split(ID, "/")

	if len(profileMemberParts) < 6 {
		return nil, fmt.Errorf("Invalid user profile member reference, please check the state for %s", ID)

	}
	profilePermission := &britive.ProfilePermission{
		ProfileID: profileMemberParts[1],
		Name:      profileMemberParts[3],
		Type:      profileMemberParts[5],
	}
	return profilePermission, nil
}
