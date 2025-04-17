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
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourcePermission - Terraform Resource for Permission
type ResourcePermission struct {
	Resource     *schema.Resource
	helper       *ResourcePermissionHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourcePermission - Initializes new permission resource
func NewResourcePermission(v *validate.Validation, importHelper *imports.ImportHelper) *ResourcePermission {
	rp := &ResourcePermission{
		helper:       NewResourcePermissionHelper(),
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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of Britive permission",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive permission",
			},
			"consumer": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The consumer service",
			},
			"resources": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Comma separated list of resources",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"actions": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Actions to be performed on the resource",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
	return rp
}

//region Permission Resource Context Operations

func (rp *ResourcePermission) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	permission := britive.Permission{}

	err := rp.helper.mapResourceToModel(d, m, &permission, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new permission: %#v", permission)

	pm, err := c.AddPermission(permission)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new permission: %#v", pm)
	d.SetId(rp.helper.generateUniqueID(pm.PermissionID))

	rp.resourceRead(ctx, d, m)

	return diags
}

func (rp *ResourcePermission) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rp.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rp *ResourcePermission) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permissionID, err := rp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("consumer") || d.HasChange("resources") || d.HasChange("actions") {
		hasChanges = true
		permission := britive.Permission{}

		err := rp.helper.mapResourceToModel(d, m, &permission, true)
		if err != nil {
			return diag.FromErr(err)
		}

		old_name, _ := d.GetChange("name")
		up, err := c.UpdatePermission(permission, old_name.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated permission: %#v", up)
		d.SetId(rp.helper.generateUniqueID(permissionID))
	}
	if hasChanges {
		return rp.resourceRead(ctx, d, m)
	}
	return nil
}

func (rp *ResourcePermission) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	permissionID, err := rp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting permission: %s", permissionID)
	err = c.DeletePermission(permissionID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Permission %s deleted", permissionID)
	d.SetId("")

	return diags
}

func (rp *ResourcePermission) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rp.importHelper.ParseImportID([]string{"permissions/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	permissionName := d.Get("name").(string)
	if strings.TrimSpace(permissionName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing permission: %s", permissionName)

	permission, err := c.GetPermissionByName(permissionName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("permission %s", permissionName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rp.helper.generateUniqueID(permission.PermissionID))

	err = rp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported permission: %s", permissionName)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourcePermissionHelper - Resource Permission helper functions
type ResourcePermissionHelper struct {
}

// NewResourcePermissionsHelper - Initializes new permission resource helper
func NewResourcePermissionHelper() *ResourcePermissionHelper {
	return &ResourcePermissionHelper{}
}

//region Permissions Resource helper functions

func (rph *ResourcePermissionHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, permission *britive.Permission, isUpdate bool) error {

	permission.Name = d.Get("name").(string)
	permission.Description = d.Get("description").(string)
	permission.Consumer = d.Get("consumer").(string)

	res := d.Get("resources").(*schema.Set)
	permission.Resources = append(permission.Resources, res.List()...)

	act := d.Get("actions").(*schema.Set)
	permission.Actions = append(permission.Actions, act.List()...)

	return nil
}

func (rph *ResourcePermissionHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	permissionID, err := rph.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading permission %s", permissionID)

	permissionRes, err := c.GetPermission(permissionID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("permission %s", permissionID)
	}
	if err != nil {
		return err
	}
	permission, err := c.GetPermissionByName(permissionRes.Name)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("role %s", permissionRes.Name)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received permission %#v", permission)

	if err := d.Set("name", permission.Name); err != nil {
		return err
	}
	if err := d.Set("description", permission.Description); err != nil {
		return err
	}
	if err := d.Set("consumer", permission.Consumer); err != nil {
		return err
	}
	if err := d.Set("resources", permission.Resources); err != nil {
		return err
	}
	if err := d.Set("actions", permission.Actions); err != nil {
		return err
	}
	return nil
}

func (resourcePermissionHelper *ResourcePermissionHelper) generateUniqueID(permissionID string) string {
	return fmt.Sprintf("permissions/%s", permissionID)
}

func (resourcePermissionHelper *ResourcePermissionHelper) parseUniqueID(ID string) (permissionID string, err error) {
	permissionParts := strings.Split(ID, "/")
	if len(permissionParts) < 2 {
		err = errs.NewInvalidResourceIDError("permission", ID)
		return
	}

	permissionID = permissionParts[1]
	return
}

//endregion
