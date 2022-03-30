package britive

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourcePermissions - Terraform Resource for Permissions
type ResourcePermissions struct {
	Resource     *schema.Resource
	helper       *ResourcePermissionsHelper
	validation   *Validation
	importHelper *ImportHelper
}

//NewResourcePermissions - Initializes new permission resource
func NewResourcePermissions(v *Validation, importHelper *ImportHelper) *ResourcePermissions {
	rp := &ResourcePermissions{
		helper:       NewResourcePermissionsHelper(),
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
				Type:        schema.TypeList,
				Required:    true,
				Description: "Comma separated list of resources",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"actions": {
				Type:        schema.TypeList,
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

func (rp *ResourcePermissions) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	permissions := britive.Permissions{}

	err := rp.helper.mapResourceToModel(d, m, &permissions, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new permission: %#v", permissions)

	pm, err := c.AddPermission(permissions)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new permission: %#v", pm)
	d.SetId(pm.PermissionID)

	rp.resourceRead(ctx, d, m)

	return diags
}

func (rp *ResourcePermissions) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rp.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rp *ResourcePermissions) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permissionID := d.Id()
	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("consumer") || d.HasChange("resources") || d.HasChange("actions") {
		hasChanges = true
		permissions := britive.Permissions{}

		err := rp.helper.mapResourceToModel(d, m, &permissions, true)
		if err != nil {
			return diag.FromErr(err)
		}

		up, err := c.UpdatePermission(permissionID, permissions)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated permission: %#v", up)
		d.SetId(permissionID)
	}
	if hasChanges {
		return rp.resourceRead(ctx, d, m)
	}
	return nil
}

func (rp *ResourcePermissions) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	permissionID := d.Id()

	log.Printf("[INFO] Deleting permission: %s", permissionID)
	err := c.DeletePermission(permissionID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Permission %s deleted", permissionID)
	d.SetId("")

	return diags
}

func (rp *ResourcePermissions) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rp.importHelper.ParseImportID([]string{"api/v1/policy-admin/policies/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	permissionName := d.Get("name").(string)
	if strings.TrimSpace(permissionName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing permission: %s", permissionName)

	permissions, err := c.GetPermissionByName(permissionName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("permission %s", permissionName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(permissions.PermissionID)

	err = rp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported permission: %s", permissionName)
	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourcePermissionsHelper - Resource Permission helper functions
type ResourcePermissionsHelper struct {
}

//NewResourcePermissionsHelper - Initializes new permission resource helper
func NewResourcePermissionsHelper() *ResourcePermissionsHelper {
	return &ResourcePermissionsHelper{}
}

//region Permissions Resource helper functions

func (rph *ResourcePermissionsHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, permissions *britive.Permissions, isUpdate bool) error {

	permissions.Name = d.Get("name").(string)
	permissions.Description = d.Get("description").(string)
	permissions.Consumer = d.Get("consumer").(string)

	res := d.Get("resources").([]interface{})
	permissions.Resources = append(permissions.Resources, res...)

	act := d.Get("actions").([]interface{})
	permissions.Actions = append(permissions.Actions, act...)

	return nil
}

func (rph *ResourcePermissionsHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	permissionID := d.Id()

	log.Printf("[INFO] Reading permission %s", permissionID)

	permissions, err := c.GetPermission(permissionID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("permission %s", permissionID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received permission %#v", permissions)

	if err := d.Set("name", permissions.Name); err != nil {
		return err
	}
	if err := d.Set("description", permissions.Description); err != nil {
		return err
	}
	if err := d.Set("consumer", permissions.Consumer); err != nil {
		return err
	}
	if err := d.Set("resources", permissions.Resources); err != nil {
		return err
	}
	if err := d.Set("actions", permissions.Actions); err != nil {
		return err
	}
	return nil
}

//endregion
