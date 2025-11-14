package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceResourceManagerProfilePermission - Terraform Resource for Profile Permission Management
type ResourceResourceManagerProfilePermission struct {
	Resource     *schema.Resource
	helper       *ResourceResourceManagerProfilePermissionHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceResourceManagerProfilePermissionHelper - Helper for Profile Permission Management Resource
type ResourceResourceManagerProfilePermissionHelper struct{}

func NewResourceResourceManagerProfilePermissionHelper() *ResourceResourceManagerProfilePermissionHelper {
	return &ResourceResourceManagerProfilePermissionHelper{}
}

func NewResourceResourceManagerProfilePermission(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResourceManagerProfilePermission {
	rrmppr := &ResourceResourceManagerProfilePermission{
		helper:       NewResourceResourceManagerProfilePermissionHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rrmppr.Resource = &schema.Resource{
		CreateContext: rrmppr.resourceCreate,
		UpdateContext: rrmppr.resourceUpdate,
		ReadContext:   rrmppr.resourceRead,
		DeleteContext: rrmppr.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rrmppr.resourceStateImporter,
		},
		CustomizeDiff: rrmppr.validation.ValidateImmutableFields([]string{
			"profile_id",
			"name",
		}),
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Profile Id",
			},
			"permission_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Profile permission Id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the permission.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of permission",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Version of the permission.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"resource_type_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of ResourceType associated with this permission",
			},
			"resource_type_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of ResourceType associated with this permission",
			},
			"variables": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Variables of permission",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of variable associated with permission",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Value of variable",
						},
						"is_system_defined": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "State value is system designed or not",
						},
					},
				},
			},
		},
	}
	return rrmppr
}

func (rrmppr *ResourceResourceManagerProfilePermission) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	resourceManagerProfilePermission := &britive.ResourceManagerProfilePermission{}

	log.Printf("[INFO] Mapping resource to permission model")

	resourceManagerProfilePermission, err := rrmppr.helper.mapResourceToModel(d, c, resourceManagerProfilePermission)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating profile permission %#v", resourceManagerProfilePermission)

	resourceManagerProfilePermission, err = c.CreateUpdateResourceManagerProfilePermission(*resourceManagerProfilePermission, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rrmppr.helper.generateUniqueID(resourceManagerProfilePermission.ProfilID, resourceManagerProfilePermission.PermissionID))

	log.Printf("[INFO] Created profile permission %#v", resourceManagerProfilePermission)
	return rrmppr.resourceRead(ctx, d, m)
}

func (rrmppr *ResourceResourceManagerProfilePermission) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	profileID, permissionID := rrmppr.helper.parseUniqueID(d.Id())

	log.Printf("[INFO] Reading profile permission with profile: %s and permission: %s", profileID, permissionID)

	resourceManagerPermissions, err := c.GetResourceManagerProfilePermission(profileID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Finding permiision from list of perissions: %#v", resourceManagerPermissions)

	err = rrmppr.helper.getAndMapModelToResource(d, *resourceManagerPermissions, permissionID)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rrmppr *ResourceResourceManagerProfilePermission) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	if d.HasChange("version") {
		oldVer, newVer := d.GetChange("version")
		if !strings.EqualFold(oldVer.(string), newVer.(string)) {
			return diag.FromErr(fmt.Errorf("field 'version' is immutable and cannot be changed (from '%v' to '%v')", oldVer.(string), newVer.(string)))
		}
	}

	if d.HasChange("name") || d.HasChange("profile_id") || d.HasChange("description") || d.HasChange("version") || d.HasChange("variables") {

		profileID, permissionID := rrmppr.helper.parseUniqueID(d.Id())
		resourceManagerProfilePermission := &britive.ResourceManagerProfilePermission{
			ProfilID:     profileID,
			PermissionID: permissionID,
		}

		resourceManagerProfilePermission, err := rrmppr.helper.mapResourceToModel(d, c, resourceManagerProfilePermission)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating resource manager profile permission: %#v", resourceManagerProfilePermission)

		resourceManagerProfilePermission, err = c.CreateUpdateResourceManagerProfilePermission(*resourceManagerProfilePermission, true)
		if err != nil {
			return diag.FromErr(err)
		}

	}

	log.Printf("[INFO] Updated resource manager profile permission: %s", d.Id())

	return rrmppr.resourceRead(ctx, d, m)
}

func (rrmppr *ResourceResourceManagerProfilePermission) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	profileID, permissionID := rrmppr.helper.parseUniqueID(d.Id())

	log.Printf("[INFO] Deleting resource manager profile permission with profile: %s, permission: %s", profileID, permissionID)

	err := c.DeleteResourceManagerProfilePermission(profileID, permissionID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	log.Printf("[INFO] Deleted resource manager profile permission with profile: %s, permission: %s", profileID, permissionID)

	return diags
}

func (rrmppr *ResourceResourceManagerProfilePermission) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	if err := rrmppr.importHelper.ParseImportID([]string{"resource-manager/profile/(?P<profile_id>[^/]+)/permission/(?P<permission_id>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	profileID := d.Get("profile_id").(string)
	permissionID := d.Get("permission_id").(string)
	if strings.TrimSpace(profileID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_id")
	}
	if strings.TrimSpace(permissionID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_id")
	}

	log.Printf("[INFO] Importing resource manager profile permission: %s/%s", profileID, permissionID)

	permission, err := c.GetResourceManagerProfilePermission(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("permission %s for profile %s", permissionID, profileID)
	}
	if err != nil {
		return nil, err
	}

	isFoundPermission := false
	for _, val := range permission.Permissions {
		if permissionID == val["permissionId"].(string) {
			isFoundPermission = true
			break
		}
	}

	if isFoundPermission {
		d.SetId(rrmppr.helper.generateUniqueID(profileID, permissionID))
		log.Printf("[INFO] Imported resource manager profile permission: %s/%s", profileID, permissionID)
		return []*schema.ResourceData{d}, nil
	} else {
		return nil, errs.NewNotFoundErrorf("permission with id : %s", permissionID)
	}

}

func (helper *ResourceResourceManagerProfilePermissionHelper) mapResourceToModel(d *schema.ResourceData, c *britive.Client, resourceManagerProfilePermission *britive.ResourceManagerProfilePermission) (*britive.ResourceManagerProfilePermission, error) {
	rawProfileID := d.Get("profile_id").(string)

	profArr := strings.Split(rawProfileID, "/")

	profileID := profArr[len(profArr)-1]

	resourceManagerProfilePermission.ProfilID = profileID

	rawPermissions, err := c.GetAvailablePermissions(profileID)
	if err != nil {
		return nil, err
	}

	permissionName := d.Get("name").(string)
	for _, permission := range rawPermissions.Permissions {
		if permission["name"].(string) == permissionName {
			resourceManagerProfilePermission.PermissionID = permission["permissionId"].(string)
			break
		}
	}

	if resourceManagerProfilePermission.PermissionID == "" {
		return nil, fmt.Errorf("permission '%s' is invalid or already associated with the profile", permissionName)
	}

	version := d.Get("version").(string)
	if strings.EqualFold(version, "latest") || strings.EqualFold(version, "local") {
		version = strings.ToLower(version)
	}
	resourceTypePermission, err := c.GetSpecifiedVersionPermission(resourceManagerProfilePermission.PermissionID, version)
	if err != nil {
		return nil, errs.NewNotFoundErrorf("permission with version: %s", version)
	}

	resourceManagerProfilePermission.Version = version
	resourceManagerProfilePermission.ResourceTypeId = resourceTypePermission.ResourceTypeID
	resourceManagerProfilePermission.ResourceTypeName = resourceTypePermission.ResourceTypeName

	userVariables := d.Get("variables").(*schema.Set).List()
	permissionVariableMap := make(map[string]bool)
	for _, v := range resourceTypePermission.Variables {
		permissionVariableMap[v.(string)] = true
	}
	for _, v := range userVariables {
		variable := v.(map[string]interface{})
		varName := variable["name"].(string)
		if _, ok := permissionVariableMap[varName]; !ok {
			return nil, fmt.Errorf("the variable '%s' is not valid for the '%s' permission", varName, permissionName)
		}
	}
	if len(userVariables) < len(resourceTypePermission.Variables) {
		return nil, fmt.Errorf("missing required variables: all variables defined in the '%s' permission are mandatory and must be provided", permissionName)
	}

	for _, v := range userVariables {
		vMap := v.(map[string]interface{})
		vMap["isSystemDefined"] = vMap["is_system_defined"]
		resourceManagerProfilePermission.Variables = append(resourceManagerProfilePermission.Variables, vMap)
	}

	return resourceManagerProfilePermission, nil

}

func (helper *ResourceResourceManagerProfilePermissionHelper) getAndMapModelToResource(d *schema.ResourceData, resourceManagerPermissions britive.ResourceManagerPermissions, permissionID string) error {
	var permission map[string]interface{}
	for _, perm := range resourceManagerPermissions.Permissions {
		if perm["permissionId"].(string) == permissionID {
			permission = perm
			break
		}
	}

	if permission == nil {
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Setting resource manager profile permission %#v", permission)

	if err := d.Set("permission_id", permission["permissionId"]); err != nil {
		return err
	}
	if err := d.Set("name", permission["permissionName"].(string)); err != nil {
		return err
	}
	if err := d.Set("description", permission["description"].(string)); err != nil {
		return err
	}
	if err := d.Set("version", permission["version"].(string)); err != nil {
		return err
	}
	if err := d.Set("resource_type_id", permission["resourceTypeId"].(string)); err != nil {
		return err
	}
	if err := d.Set("resource_type_name", permission["resourceTypeName"].(string)); err != nil {
		return err
	}
	var stateVariables []map[string]interface{}
	if variables, ok := permission["variables"].([]interface{}); ok {
		for _, v := range variables {
			if permMap, ok := v.(map[string]interface{}); ok {
				newVar := map[string]interface{}{
					"name":              permMap["name"],
					"value":             permMap["value"],
					"is_system_defined": permMap["isSystemDefined"],
				}
				stateVariables = append(stateVariables, newVar)
			}
		}
	}
	if err := d.Set("variables", stateVariables); err != nil {
		return err
	}

	log.Printf("[INFO] Read resource manager profile permisiion : %#v", permission)
	return nil
}

func (helper *ResourceResourceManagerProfilePermissionHelper) generateUniqueID(profileID, permissionID string) string {
	return fmt.Sprintf("resource-manager/profile/%s/permission/%s", profileID, permissionID)
}

func (helper *ResourceResourceManagerProfilePermissionHelper) parseUniqueID(ID string) (string, string) {
	idArr := strings.Split(ID, "/")
	profileID := idArr[len(idArr)-3]
	permissionID := idArr[len(idArr)-1]
	return profileID, permissionID
}
