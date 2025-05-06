package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceTypePermissions - Terraform Resource for Resource Type Permissions
type ResourceTypePermissions struct {
	Resource     *schema.Resource
	helper       *ResourceTypePermissionsHelper
	importHelper *imports.ImportHelper
}

// ResourceTypePermissionsHelper - Helper for Resource Type Permissions Resource
type ResourceTypePermissionsHelper struct{}

func NewResourceTypePermissionsHelper() *ResourceTypePermissionsHelper {
	return &ResourceTypePermissionsHelper{}
}

func NewResourceTypePermissions(importHelper *imports.ImportHelper) *ResourceTypePermissions {
	rtp := &ResourceTypePermissions{
		helper:       NewResourceTypePermissionsHelper(),
		importHelper: importHelper,
	}
	rtp.Resource = &schema.Resource{
		CreateContext: rtp.resourceCreate,
		ReadContext:   rtp.resourceRead,
		UpdateContext: rtp.resourceUpdate,
		DeleteContext: rtp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rtp.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"permission_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the permission.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the permission.",
			},
			"versions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of versions for the permission.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"resource_type_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the associated resource type.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the permission.",
			},
			"checkin_time_limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     60,
				Description: "The check-in time limit in minutes.",
			},
			"checkout_time_limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     60,
				Description: "The check-out time limit in minutes.",
			},
			"is_draft": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicates if the permission is a draft.",
			},
			"inline_file_exists": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates if an inline file exists.",
			},
			"response_templates": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Default:     []string{},
				Description: "List of response template names.",
			},
			"show_orig_creds": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicates if original credentials should be shown.",
			},
			"checkin_code_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The file path for check-in code.",
			},
			"checkout_code_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The file path for check-out code.",
			},
			"checkin_code": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The inline check-in code.",
			},
			"checkout_code": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The inline check-out code.",
			},
			"checkin_file_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the check-in file.",
			},
			"checkout_file_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the check-out file.",
			},
			"variables": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of variables.",
			},
		},
	}
	return rtp
}

func (rtp *ResourceTypePermissions) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permission := &britive.ResourceTypePermission{}
	rtp.helper.mapResourceToModel(d, permission)

	log.Printf("[INFO] Creating resource type permission: %#v", permission)

	resp, err := c.CreateResourceTypePermission(*permission)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rtp.helper.generateUniqueID(resp.PermissionID))
	log.Printf("[INFO] Created resource type permission: %s", resp.PermissionID)
	return rtp.resourceRead(ctx, d, m)
}

func (rtp *ResourceTypePermissions) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permissionID, err := rtp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading resource type permission: %s", permissionID)

	permission, err := c.GetResourceTypePermission(permissionID)
	if errors.Is(err, britive.ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	rtp.helper.getAndMapModelToResource(d, permission)
	log.Printf("[INFO] Retrieved resource type permission: %s", permissionID)
	return nil
}

func (rtp *ResourceTypePermissions) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permissionID, err := rtp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	permission := &britive.ResourceTypePermission{}
	rtp.helper.mapResourceToModel(d, permission)
	permission.PermissionID = permissionID

	log.Printf("[INFO] Updating resource type permission: %s", permissionID)

	_, err = c.UpdateResourceTypePermission(*permission)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated resource type permission: %s", permissionID)
	return rtp.resourceRead(ctx, d, m)
}

func (rtp *ResourceTypePermissions) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permissionID, err := rtp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting resource type permission: %s", permissionID)

	err = c.DeleteResourceTypePermission(permissionID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted resource type permission: %s", permissionID)
	d.SetId("")
	return nil
}

func (helper *ResourceTypePermissionsHelper) mapResourceToModel(d *schema.ResourceData, permission *britive.ResourceTypePermission) {
	permission.Name = d.Get("name").(string)
	permission.Description = d.Get("description").(string)
	permission.ResourceTypeID = d.Get("resource_type_id").(string)
	permission.IsDraft = d.Get("is_draft").(bool)
	permission.CheckinTimeLimit = d.Get("checkin_time_limit").(int)
	permission.CheckoutTimeLimit = d.Get("checkout_time_limit").(int)
	permission.ResponseTemplates = d.Get("response_templates").([]string)
	permission.ShowOrigCreds = d.Get("show_orig_creds").(bool)
}

func (helper *ResourceTypePermissionsHelper) getAndMapModelToResource(d *schema.ResourceData, permission *britive.ResourceTypePermission) {
	d.Set("permission_id", permission.PermissionID)
	d.Set("name", permission.Name)
	d.Set("description", permission.Description)
	d.Set("resource_type_id", permission.ResourceTypeID)
	d.Set("is_draft", permission.IsDraft)
	d.Set("checkin_time_limit", permission.CheckinTimeLimit)
	d.Set("checkout_time_limit", permission.CheckoutTimeLimit)
	d.Set("inline_file_exists", permission.InlineFileExists)
	d.Set("response_templates", permission.ResponseTemplates)
	d.Set("show_orig_creds", permission.ShowOrigCreds)
	d.Set("checkin_file_name", permission.CheckinFileName)
	d.Set("checkout_file_name", permission.CheckoutFileName)
}

func (helper *ResourceTypePermissionsHelper) generateUniqueID(permissionID string) string {
	return fmt.Sprintf("resource-manager/permissions/%s", permissionID)
}

func (helper *ResourceTypePermissionsHelper) parseUniqueID(ID string) (string, error) {
	parts := strings.Split(ID, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid resource ID format: %s", ID)
	}
	return parts[2], nil
}

func (rtp *ResourceTypePermissions) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	if err := rtp.importHelper.ParseImportID([]string{"resource-manager/permissions/(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	permissionID := d.Id()
	log.Printf("[INFO] Importing resource type permission: %s", permissionID)

	permission, err := c.GetResourceTypePermission(permissionID)
	if err != nil {
		return nil, err
	}

	d.SetId(rtp.helper.generateUniqueID(permission.PermissionID))
	rtp.helper.getAndMapModelToResource(d, permission)
	log.Printf("[INFO] Imported resource type permission: %s", permissionID)
	return []*schema.ResourceData{d}, nil
}
