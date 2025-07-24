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
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourcePolicy - Terraform Resource for Policy
type ResourcePolicy struct {
	Resource     *schema.Resource
	helper       *ResourcePolicyHelper
	importHelper *imports.ImportHelper
}

// NewResourcePolicy - Initialization of new policy resource
func NewResourcePolicy(importHelper *imports.ImportHelper) *ResourcePolicy {
	rp := &ResourcePolicy{
		helper:       NewResourcePolicyHelper(),
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
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The description of the policy",
			},
			"is_active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Is the policy active",
			},
			"is_draft": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Is the policy a draft",
			},
			"is_read_only": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Is the policy read only",
			},
			"access_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Allow",
				Description: "Type of access for the policy",
			},
			"members": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "{}",
				Description:  "Members of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"condition": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				Description:  "Condition of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permissions": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "[]",
				Description:  "Permissions of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"roles": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "[]",
				Description:  "Roles of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
	return rp
}

//region Policy Resource Context Operations

func (rp *ResourcePolicy) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	policy := britive.Policy{}

	err := rp.helper.mapResourceToModel(d, m, &policy, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new policy: %#v", policy)

	po, err := c.CreatePolicy(policy)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new policy: %#v", po)
	d.SetId(rp.helper.generateUniqueID(po.PolicyID))
	rp.resourceRead(ctx, d, m)
	return diags
}

func (rp *ResourcePolicy) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rp.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rp *ResourcePolicy) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	policyID, err := rp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("is_active") || d.HasChange("is_draft") || d.HasChange("is_read_only") || d.HasChange("access_type") || d.HasChange("members") || d.HasChange("condition") || d.HasChange("permissions") || d.HasChange("roles") {
		hasChanges = true

		policy := britive.Policy{}

		err := rp.helper.mapResourceToModel(d, m, &policy, true)
		if err != nil {
			return diag.FromErr(err)
		}

		old_name, _ := d.GetChange("name")
		oldMem, _ := d.GetChange("members")
		oldCon, _ := d.GetChange("condition")
		oldPerm, _ := d.GetChange("permissions")
		oldRole, _ := d.GetChange("roles")
		up, err := c.UpdatePolicy(policy, old_name.(string))
		if err != nil {
			if errState := d.Set("members", oldMem.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			if errState := d.Set("condition", oldCon.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			if errState := d.Set("permissions", oldPerm.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			if errState := d.Set("roles", oldRole.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted Updated Policy: %#v", up)
		d.SetId(rp.helper.generateUniqueID(policyID))
	}
	if hasChanges {
		return rp.resourceRead(ctx, d, m)
	}
	return nil
}

func (rp *ResourcePolicy) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	policyID, err := rp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting Policy: %s", policyID)
	err = c.DeletePolicy(policyID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleted Policy: %s", policyID)
	d.SetId("")

	return diags
}

func (rp *ResourcePolicy) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rp.importHelper.ParseImportID([]string{"policies/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}

	policyName := d.Get("name").(string)
	if strings.TrimSpace(policyName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing Policy: %s", policyName)

	policy, err := c.GetPolicyByName(policyName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("Policy %s", policyName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rp.helper.generateUniqueID(policy.PolicyID))

	err = rp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported Policy: %s", policyName)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceProfilePolicyHelper - Terraform Resource for Policy Helper
type ResourcePolicyHelper struct {
}

// NewResourcePolicyHelper - Initialization of new policy resource helper
func NewResourcePolicyHelper() *ResourcePolicyHelper {
	return &ResourcePolicyHelper{}
}

//region Policy Resource helper functions

func (rph *ResourcePolicyHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, policy *britive.Policy, isUpdate bool) error {

	policy.Name = d.Get("name").(string)
	policy.Description = d.Get("description").(string)
	policy.AccessType = d.Get("access_type").(string)
	policy.IsActive = d.Get("is_active").(bool)
	policy.IsDraft = d.Get("is_draft").(bool)
	policy.IsReadOnly = d.Get("is_read_only").(bool)
	policy.Condition = d.Get("condition").(string)
	json.Unmarshal([]byte(d.Get("members").(string)), &policy.Members)
	json.Unmarshal([]byte(d.Get("permissions").(string)), &policy.Permissions)
	json.Unmarshal([]byte(d.Get("roles").(string)), &policy.Roles)

	return nil
}

func (rph *ResourcePolicyHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	policyID, err := rph.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading Policy: %s", policyID)

	policyId, err := c.GetPolicy(policyID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("Policy %s ", policyID)
	}
	if err != nil {
		return err
	}
	policy, err := c.GetPolicyByName(policyId.Name)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("Policy %s ", policyId.Name)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received Policy: %#v", policy)

	if err := d.Set("name", policy.Name); err != nil {
		return err
	}
	if err := d.Set("description", policy.Description); err != nil {
		return err
	}
	if err := d.Set("access_type", policy.AccessType); err != nil {
		return err
	}
	if err := d.Set("is_active", policy.IsActive); err != nil {
		return err
	}
	if err := d.Set("is_draft", policy.IsDraft); err != nil {
		return err
	}
	if err := d.Set("is_read_only", policy.IsReadOnly); err != nil {
		return err
	}

	newCon := d.Get("condition")
	if britive.ConditionEqual(policy.Condition, newCon.(string)) {
		if err := d.Set("condition", newCon.(string)); err != nil {
			return err
		}
	} else if err := d.Set("condition", policy.Condition); err != nil {
		return err
	}

	mem, err := json.Marshal(policy.Members)
	if err != nil {
		return err
	}

	newMem := d.Get("members")
	if britive.MembersEqual(string(mem), newMem.(string)) {
		if err := d.Set("members", newMem.(string)); err != nil {
			return err
		}
	} else if err := d.Set("members", string(mem)); err != nil {
		return err
	}

	perm, err := json.Marshal(policy.Permissions)
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

	role, err := json.Marshal(policy.Roles)
	if err != nil {
		return err
	}

	newRole := d.Get("roles")
	if britive.ArrayOfMapsEqual(string(role), newRole.(string)) {
		if err := d.Set("roles", newRole.(string)); err != nil {
			return err
		}
	} else if err := d.Set("roles", string(role)); err != nil {
		return err
	}

	return nil
}

func (resourcePolicyHelper *ResourcePolicyHelper) generateUniqueID(policyID string) string {
	return fmt.Sprintf("policies/%s", policyID)
}

func (resourcePolicyHelper *ResourcePolicyHelper) parseUniqueID(ID string) (policyID string, err error) {
	policyParts := strings.Split(ID, "/")
	if len(policyParts) < 2 {
		err = errs.NewInvalidResourceIDError("Policy", ID)
		return
	}
	policyID = policyParts[1]
	return
}

//endregion
