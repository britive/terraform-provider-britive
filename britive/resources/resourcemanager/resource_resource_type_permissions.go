package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceResourceTypePermissions - Terraform Resource for Resource Type Permissions
type ResourceResourceTypePermissions struct {
	Resource     *schema.Resource
	helper       *ResourceResourceTypePermissionsHelper
	importHelper *imports.ImportHelper
}

// ResourceResourceTypePermissionsHelper - Helper for Resource Type Permissions Resource
type ResourceResourceTypePermissionsHelper struct{}

func NewResourceResourceTypePermissionsHelper() *ResourceResourceTypePermissionsHelper {
	return &ResourceResourceTypePermissionsHelper{}
}

func NewResourceResourceTypePermissions(importHelper *imports.ImportHelper) *ResourceResourceTypePermissions {
	rtp := &ResourceResourceTypePermissions{
		helper:       NewResourceResourceTypePermissionsHelper(),
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
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version for the permission.",
				// Elem:        &schema.Schema{Type: schema.TypeString},
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
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of response template names.",
			},
			"show_orig_creds": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicates if original credentials should be shown.",
			},
			"checkin_code_file": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The file path for check-in code.",
				ConflictsWith: []string{"checkin_code", "checkout_code", "code_language"},
			},
			"checkout_code_file": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The file path for check-out code.",
				ConflictsWith: []string{"checkin_code", "checkout_code", "code_language"},
			},
			"checkin_code_file_hash": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The file hash for check-in code.",
			},
			"checkout_code_file_hash": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The file hash for check-out code.",
			},
			"checkin_code": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The inline check-in code.",
				ConflictsWith: []string{"checkin_code_file", "checkout_code_file"},
			},
			"checkout_code": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The inline check-out code.",
				ConflictsWith: []string{"checkin_code_file", "checkout_code_file"},
			},
			"code_language": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Text",
				Description:  "The inline code language. Select one of Test, Batch, Node, PoerShell, Python, Shell.",
				ValidateFunc: validation.StringInSlice([]string{"Text", "Batch", "Node", "PowerShell", "Python", "Shell"}, true),
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
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of variables.",
			},
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			checkin_code_file := d.Get("checkin_code_file").(string)
			checkout_code_file := d.Get("checkout_code_file").(string)
			checkin_code := d.Get("checkin_code").(string)
			checkout_code := d.Get("checkout_code").(string)
			show_orig_creds := d.Get("show_orig_creds").(bool)
			response_templates := d.Get("response_templates").(*schema.Set)

			if len(response_templates.List()) == 0 && !show_orig_creds {
				return fmt.Errorf("'show_orig_creds' can be set to false only if response templates are available")
			}

			if (checkin_code_file != "") != (checkout_code_file != "") {
				return fmt.Errorf("'checkin_code_file' and 'checkout_code_file' must be set together or left unset together")
			}
			if (checkin_code != "") != (checkout_code != "") {
				return fmt.Errorf("'checkin_code' and 'checkout_code' must be set together or left unset together")
			}

			// Check and set file hash
			if checkin_code_file != "" && checkout_code_file != "" {
				checkinNewHash, err := utils.HashFileContent(checkin_code_file)
				if err != nil {
					return err
				}

				checkinOldHash := d.Get("checkin_code_file_hash").(string)
				if checkinNewHash != checkinOldHash {
					d.SetNew("checkin_code_file_hash", checkinNewHash)
				}

				checkoutNewHash, err := utils.HashFileContent(checkout_code_file)
				if err != nil {
					return err
				}

				checkoutOldHash := d.Get("checkout_code_file_hash").(string)
				if checkoutNewHash != checkoutOldHash {
					d.SetNew("checkout_code_file_hash", checkoutNewHash)
				}
			}

			return nil
		},
	}
	return rtp
}

func (rtp *ResourceResourceTypePermissions) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permission := &britive.ResourceTypePermission{}
	rtp.helper.mapResourceToModel(d, permission)

	log.Printf("[INFO] Creating resource type permission draft: %#v", permission)

	// Create draft permission
	resp, err := c.CreateResourceTypePermission(*permission)
	if err != nil {
		return diag.FromErr(err)
	}

	// If is_draft is false, finalize the permission
	permission.PermissionID = resp.PermissionID
	log.Printf("[INFO] Finalizing resource type permission: %s", permission.PermissionID)

	//upload files or code
	checkInFilePath := d.Get("checkin_code_file").(string)
	checkOutFilePath := d.Get("checkout_code_file").(string)
	if checkInFilePath != "" && checkOutFilePath != "" {
		err = c.UploadPermissionFiles(permission.PermissionID, checkInFilePath, checkOutFilePath)
		if err != nil {
			return diag.FromErr(err)
		}
		permission.CheckinFileName = filepath.Base(checkInFilePath)
		permission.CheckoutFileName = filepath.Base(checkOutFilePath)
	}
	checkInCode := d.Get("checkin_code").(string)
	checkOutCode := d.Get("checkout_code").(string)
	codeLanguage := d.Get("code_language").(string)
	if checkInCode != "" && checkOutCode != "" {
		err = c.UploadPermissionCodes(permission.PermissionID, checkInCode, checkOutCode, codeLanguage)
		if err != nil {
			return diag.FromErr(err)
		}
		permission.CheckinFileName = permission.PermissionID + "_latest_checkin"
		permission.CheckoutFileName = permission.PermissionID + "_latest_checkout"
		permission.InlineFileExists = true
	}

	_, err = c.UpdateResourceTypePermission(*permission)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rtp.helper.generateUniqueID(resp.PermissionID))
	log.Printf("[INFO] Created resource type permission: %s", resp.PermissionID)
	return rtp.resourceRead(ctx, d, m)
}

func (rtp *ResourceResourceTypePermissions) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

func (rtp *ResourceResourceTypePermissions) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	permissionID, err := rtp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	permission := &britive.ResourceTypePermission{}
	rtp.helper.mapResourceToModel(d, permission)
	permission.PermissionID = permissionID

	//upload files or code
	checkInFilePath := d.Get("checkin_code_file").(string)
	checkOutFilePath := d.Get("checkout_code_file").(string)
	if checkInFilePath != "" && checkOutFilePath != "" {
		err = c.UploadPermissionFiles(permission.PermissionID, checkInFilePath, checkOutFilePath)
		if err != nil {
			return diag.FromErr(err)
		}
		permission.CheckinFileName = filepath.Base(checkInFilePath)
		permission.CheckoutFileName = filepath.Base(checkOutFilePath)
	}

	checkInCode := d.Get("checkin_code").(string)
	checkOutCode := d.Get("checkout_code").(string)
	codeLanguage := d.Get("code_language").(string)
	if checkInCode != "" && checkOutCode != "" {
		err = c.UploadPermissionCodes(permission.PermissionID, checkInCode, checkOutCode, codeLanguage)
		if err != nil {
			return diag.FromErr(err)
		}
		permission.CheckinFileName = "test_123_checkin"
		permission.CheckoutFileName = "test_123_checkout"
		permission.InlineFileExists = true
	}

	log.Printf("[INFO] Updating resource type permission: %s", permissionID)

	_, err = c.UpdateResourceTypePermission(*permission)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated resource type permission: %s", permissionID)
	return rtp.resourceRead(ctx, d, m)
}

func (rtp *ResourceResourceTypePermissions) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics
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
	return diags
}

func (helper *ResourceResourceTypePermissionsHelper) mapResourceToModel(d *schema.ResourceData, permission *britive.ResourceTypePermission) {
	permission.Name = d.Get("name").(string)
	permission.Description = d.Get("description").(string)
	permission.ResourceTypeID = d.Get("resource_type_id").(string)
	permission.IsDraft = d.Get("is_draft").(bool)
	permission.CheckinTimeLimit = d.Get("checkin_time_limit").(int)
	permission.CheckoutTimeLimit = d.Get("checkout_time_limit").(int)
	responseTemplates := d.Get("response_templates").(*schema.Set)
	permission.ResponseTemplates = append(permission.ResponseTemplates, responseTemplates.List()...)
	variables := d.Get("variables").(*schema.Set)
	permission.Variables = append(permission.Variables, variables.List()...)
	permission.ShowOrigCreds = d.Get("show_orig_creds").(bool)
}

func (helper *ResourceResourceTypePermissionsHelper) getAndMapModelToResource(d *schema.ResourceData, permission *britive.ResourceTypePermission) {
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

func (helper *ResourceResourceTypePermissionsHelper) generateUniqueID(permissionID string) string {
	return fmt.Sprintf("resource-manager/permissions/%s", permissionID)
}

func (helper *ResourceResourceTypePermissionsHelper) parseUniqueID(ID string) (string, error) {
	parts := strings.Split(ID, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid resource ID format: %s", ID)
	}
	return parts[2], nil
}

func (rtp *ResourceResourceTypePermissions) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
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
