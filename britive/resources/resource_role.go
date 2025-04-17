package resources

import (
	"context"
	"encoding/json"
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceRole - Terraform Resource for Role
type ResourceRole struct {
	Resource     *schema.Resource
	helper       *ResourceRoleHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceRole - Initializes new role resource
func NewResourceRole(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceRole {
	rr := &ResourceRole{
		helper:       NewResourceRoleHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rr.Resource = &schema.Resource{
		CreateContext: rr.resourceCreate,
		ReadContext:   rr.resourceRead,
		UpdateContext: rr.resourceUpdate,
		DeleteContext: rr.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rr.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of Britive role",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive role",
			},
			"permissions": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Permissions of the role",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
	return rr
}

//region Role Resource Context Operations

func (rr *ResourceRole) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	role := britive.Role{}

	err := rr.helper.mapResourceToModel(d, m, &role, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new role: %#v", role)

	ro, err := c.AddRole(role)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new role: %#v", ro)
	d.SetId(rr.helper.generateUniqueID(ro.RoleID))

	rr.resourceRead(ctx, d, m)

	return diags
}

func (rr *ResourceRole) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rr.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rr *ResourceRole) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	roleID, err := rr.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("permissions") {
		hasChanges = true
		role := britive.Role{}

		err := rr.helper.mapResourceToModel(d, m, &role, true)
		if err != nil {
			return diag.FromErr(err)
		}

		old_name, _ := d.GetChange("name")
		oldPerm, _ := d.GetChange("permissions")
		ur, err := c.UpdateRole(role, old_name.(string))
		if err != nil {
			if errState := d.Set("permissions", oldPerm.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated role: %#v", ur)
		d.SetId(rr.helper.generateUniqueID(roleID))
	}
	if hasChanges {
		return rr.resourceRead(ctx, d, m)
	}
	return nil
}

func (rr *ResourceRole) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	roleID, err := rr.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting role: %s", roleID)
	err = c.DeleteRole(roleID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Role %s deleted", roleID)
	d.SetId("")

	return diags
}

func (rr *ResourceRole) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rr.importHelper.ParseImportID([]string{"roles/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	roleName := d.Get("name").(string)
	if strings.TrimSpace(roleName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing role: %s", roleName)

	role, err := c.GetRoleByName(roleName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("role %s", roleName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rr.helper.generateUniqueID(role.RoleID))

	err = rr.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported role: %s", roleName)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceRoleHelper - Resource Role helper functions
type ResourceRoleHelper struct {
}

// NewResourceRoleHelper - Initializes new role resource helper
func NewResourceRoleHelper() *ResourceRoleHelper {
	return &ResourceRoleHelper{}
}

//region Role Resource helper functions

func (rrh *ResourceRoleHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, role *britive.Role, isUpdate bool) error {
	role.Name = d.Get("name").(string)
	role.Description = d.Get("description").(string)
	json.Unmarshal([]byte(d.Get("permissions").(string)), &role.Permissions)

	return nil
}

func (rrh *ResourceRoleHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	roleID, err := rrh.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading role %s", roleID)

	roleRes, err := c.GetRole(roleID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("role %s", roleID)
	}
	if err != nil {
		return err
	}

	role, err := c.GetRoleByName(roleRes.Name)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("role %s", roleRes.Name)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received role %#v", role)

	if err := d.Set("name", role.Name); err != nil {
		return err
	}
	if err := d.Set("description", role.Description); err != nil {
		return err
	}
	perm, err := json.Marshal(role.Permissions)
	if err != nil {
		return err
	}

	newPerm := d.Get("permissions")
	if britive.ArrayOfMapsEqual(string(perm), newPerm.(string)) {
		if err := d.Set("permissions", newPerm.(string)); err != nil {
			return err
		}
	} else if err := d.Set("permissions", string(perm)); err != nil {
		return err
	}
	return nil
}

func (resourceRoleHelper *ResourceRoleHelper) generateUniqueID(roleID string) string {
	return fmt.Sprintf("roles/%s", roleID)
}

func (resourceRoleHelper *ResourceRoleHelper) parseUniqueID(ID string) (roleID string, err error) {
	roleParts := strings.Split(ID, "/")
	if len(roleParts) < 2 {
		err = errs.NewInvalidResourceIDError("role", ID)
		return
	}

	roleID = roleParts[1]
	return
}

//endregion
