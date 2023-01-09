package britive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceProfilePolicy - Terraform Resource for Profile Policy
type ResourceProfilePolicy struct {
	Resource     *schema.Resource
	helper       *ResourceProfilePolicyHelper
	importHelper *ImportHelper
}

// NewResourceProfilePolicy - Initialization of new profile policy resource
func NewResourceProfilePolicy(importHelper *ImportHelper) *ResourceProfilePolicy {
	rpp := &ResourceProfilePolicy{
		helper:       NewResourceProfilePolicyHelper(),
		importHelper: importHelper,
	}
	rpp.Resource = &schema.Resource{
		CreateContext: rpp.resourceCreate,
		ReadContext:   rpp.resourceRead,
		UpdateContext: rpp.resourceUpdate,
		DeleteContext: rpp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rpp.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"policy_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The policy associated with the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the profile policy",
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
			"consumer": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "papservice",
				Description: "The consumer service",
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
				Description:  "Members of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return britive.MembersEqual(old, new)
				},
			},
			"condition": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Condition of the policy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return britive.ConditionEqual(old, new)
				},
			},
		},
	}
	return rpp
}

//region Profile Policy Resource Context Operations

func (rpp *ResourceProfilePolicy) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	profilePolicy := britive.ProfilePolicy{}

	err := rpp.helper.mapResourceToModel(d, m, &profilePolicy, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new profile policy: %#v", profilePolicy)

	pp, err := c.CreateProfilePolicy(profilePolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile policy: %#v", pp)
	d.SetId(rpp.helper.generateUniqueID(profilePolicy.ProfileID, pp.PolicyID))
	rpp.resourceRead(ctx, d, m)
	return diags
}

func (rpp *ResourceProfilePolicy) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rpp.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rpp *ResourceProfilePolicy) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var hasChanges bool
	if d.HasChange("profile_id") || d.HasChange("policy_name") || d.HasChange("description") || d.HasChange("is_active") || d.HasChange("is_draft") || d.HasChange("is_read_only") || d.HasChange("consumer") || d.HasChange("access_type") || d.HasChange("members") || d.HasChange("condition") {
		hasChanges = true
		profileID, policyID, err := rpp.helper.parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		profilePolicy := britive.ProfilePolicy{}

		err = rpp.helper.mapResourceToModel(d, m, &profilePolicy, true)
		if err != nil {
			return diag.FromErr(err)
		}

		profilePolicy.PolicyID = policyID
		profilePolicy.ProfileID = profileID

		upp, err := c.UpdateProfilePolicy(profilePolicy)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted Updated profile policy: %#v", upp)
		d.SetId(rpp.helper.generateUniqueID(profilePolicy.ProfileID, profilePolicy.PolicyID))
	}
	if hasChanges {
		return rpp.resourceRead(ctx, d, m)
	}
	return nil
}

func (rpp *ResourceProfilePolicy) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, policyID, err := rpp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting profile policy: %s/%s", profileID, policyID)
	err = c.DeleteProfilePolicy(profileID, policyID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleted profile policy: %s/%s", profileID, policyID)
	d.SetId("")

	return diags
}

func (rpp *ResourceProfilePolicy) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpp.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/policies/(?P<policy_name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<policy_name>[^/]+)"}, d); err != nil {
		return nil, err
	}

	profileID := d.Get("profile_id").(string)
	policyName := d.Get("policy_name").(string)
	if strings.TrimSpace(profileID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("profile_id")
	}
	if strings.TrimSpace(policyName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("policy_name")
	}

	log.Printf("[INFO] Importing profile policy: %s/%s", profileID, policyName)

	policy, err := c.GetProfilePolicyByName(profileID, policyName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("policy %s for profile %s", policyName, profileID)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rpp.helper.generateUniqueID(profileID, policy.PolicyID))

	err = rpp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported profile policy: %s/%s", profileID, policyName)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceProfilePolicyHelper - Terraform Resource for Profile Policy Helper
type ResourceProfilePolicyHelper struct {
}

// NewResourceProfilePolicyHelper - Initialization of new profile policy resource helper
func NewResourceProfilePolicyHelper() *ResourceProfilePolicyHelper {
	return &ResourceProfilePolicyHelper{}
}

//region ProfilePolicy Resource helper functions

func (rpph *ResourceProfilePolicyHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, profilePolicy *britive.ProfilePolicy, isUpdate bool) error {

	profilePolicy.ProfileID = d.Get("profile_id").(string)
	profilePolicy.Name = d.Get("policy_name").(string)
	profilePolicy.Description = d.Get("description").(string)
	profilePolicy.Consumer = d.Get("consumer").(string)
	profilePolicy.AccessType = d.Get("access_type").(string)
	profilePolicy.IsActive = d.Get("is_active").(bool)
	profilePolicy.IsDraft = d.Get("is_draft").(bool)
	profilePolicy.IsReadOnly = d.Get("is_read_only").(bool)
	profilePolicy.Condition = d.Get("condition").(string)
	json.Unmarshal([]byte(d.Get("members").(string)), &profilePolicy.Members)

	return nil
}

func (rpph *ResourceProfilePolicyHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	profileID, policyID, err := rpph.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading profile policy: %s/%s", profileID, policyID)

	profilePolicyId, err := c.GetProfilePolicy(profileID, policyID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("policy %s in profile %s", policyID, profileID)
	}
	if err != nil {
		return err
	}
	profilePolicy, err := c.GetProfilePolicyByName(profileID, profilePolicyId.Name)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("policy %s in profile %s", profilePolicy.Name, profileID)
	}
	if err != nil {
		return err
	}

	profilePolicy.ProfileID = profileID

	log.Printf("[INFO] Received profile policy: %#v", profilePolicy)

	if err := d.Set("profile_id", profilePolicy.ProfileID); err != nil {
		return err
	}
	if err := d.Set("policy_name", profilePolicy.Name); err != nil {
		return err
	}
	if err := d.Set("description", profilePolicy.Description); err != nil {
		return err
	}
	if err := d.Set("consumer", profilePolicy.Consumer); err != nil {
		return err
	}
	if err := d.Set("access_type", profilePolicy.AccessType); err != nil {
		return err
	}
	if err := d.Set("is_active", profilePolicy.IsActive); err != nil {
		return err
	}
	if err := d.Set("is_draft", profilePolicy.IsDraft); err != nil {
		return err
	}
	if err := d.Set("is_read_only", profilePolicy.IsReadOnly); err != nil {
		return err
	}
	if err := d.Set("condition", profilePolicy.Condition); err != nil {
		return err
	}
	mem, err := json.Marshal(profilePolicy.Members)
	if err != nil {
		return err
	}
	if err := d.Set("members", string(mem)); err != nil {
		return err
	}
	return nil
}

func (resourceProfilePolicyHelper *ResourceProfilePolicyHelper) generateUniqueID(profileID string, policyID string) string {
	return fmt.Sprintf("paps/%s/policies/%s", profileID, policyID)
}

func (resourceProfilePolicyHelper *ResourceProfilePolicyHelper) parseUniqueID(ID string) (profileID string, policyID string, err error) {
	profilePolicyParts := strings.Split(ID, "/")
	if len(profilePolicyParts) < 4 {
		err = NewInvalidResourceIDError("profile policy", ID)
		return
	}

	profileID = profilePolicyParts[1]
	policyID = profilePolicyParts[3]
	return
}

//endregion
