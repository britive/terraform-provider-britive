package resources

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourcePolicyPriority - Terraform Resource for Policy Priority
type ResourcePolicyPriority struct {
	Resource     *schema.Resource
	helper       *ResourcePolicyPriorityHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// ResourcePolicyPriorityHelper - Helper for policy priority
type ResourcePolicyPriorityHelper struct{}

func NewResourcePolicyPriorityHelper() *ResourcePolicyPriorityHelper {
	return &ResourcePolicyPriorityHelper{}
}

// NewResourcePolicyPriority - Initializes new policy priority resource
func NewResourcePolicyPriority(v *validate.Validation, importHelper *imports.ImportHelper) *ResourcePolicyPriority {
	rpo := &ResourcePolicyPriority{
		helper:       NewResourcePolicyPriorityHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rpo.Resource = &schema.Resource{
		CreateContext: rpo.resourceCreate,
		ReadContext:   rpo.resourceRead,
		UpdateContext: rpo.resourceUpdate,
		DeleteContext: rpo.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rpo.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Profile ID",
			},
			"policy_priority_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable policy ordering",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					if val.(bool) != true {
						errs = append(errs, fmt.Errorf("Invalid Param."))
					}
					return
				},
			},
			"policy_priority": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Policies with id and priority",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Policy Id",
						},
						"priority": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Priority number",
						},
					},
				},
			},
		},
	}
	return rpo
}

func (rpo *ResourcePolicyPriority) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	resourcePolicyPriority := &britive.ProfilePolicyPriority{}

	log.Printf("[INFO] Mapping resource to policy priority model")
	resourcePolicyPriority, err := rpo.helper.mapResourceToModel(c, d, resourcePolicyPriority)
	if err != nil {
		return diag.FromErr(err)
	}

	profileSummary, err := c.GetProfileSummary(resourcePolicyPriority.ProfileID)
	if err != nil {
		return diag.FromErr(err)
	}

	profileSummary.PolicyOrderingEnabled = resourcePolicyPriority.PolicyOrderingEnabled

	log.Printf("[INFO] Enabling policy prioritization")
	profileSummary, err = c.EnableDisablePolicyPrioritization(*profileSummary)
	if err != nil {
		return diag.FromErr(err)
	}

	profileId := resourcePolicyPriority.ProfileID

	if resourcePolicyPriority.PolicyOrderingEnabled {
		log.Printf("[INFO] Prioritizing policies:%v", resourcePolicyPriority.PolicyOrder)
		resourcePolicyPriority, err = c.PrioritizePolicies(*resourcePolicyPriority)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	id := rpo.helper.generateUniqueID(profileId)

	d.SetId(id)

	return rpo.resourceRead(ctx, d, m)
}

func (rpo *ResourcePolicyPriority) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileId := rpo.helper.parseUniqueID(d.Id())

	log.Printf("[INFO] Getting profile policies")
	policies, err := c.GetProfilePolicies(profileId)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Getting Profile")
	profile, err := c.GetProfile(profileId)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Saving state of policy order")
	err = rpo.helper.getAndMapModelToResource(d, policies, profileId, profile.PolicyOrderingEnabled, false)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rpo *ResourcePolicyPriority) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	if d.HasChange("profile_id") || d.HasChange("policy_priority_enabled") || d.HasChange("policy_priority") {
		resourcePolicyPriority := &britive.ProfilePolicyPriority{}

		log.Printf("[INFO] Mapping resource to policy priority model")
		resourcePolicyPriority, err := rpo.helper.mapResourceToModel(c, d, resourcePolicyPriority)
		if err != nil {
			return diag.FromErr(err)
		}

		profileSummary, err := c.GetProfileSummary(resourcePolicyPriority.ProfileID)
		if err != nil {
			return diag.FromErr(err)
		}

		profileSummary.PolicyOrderingEnabled = resourcePolicyPriority.PolicyOrderingEnabled

		log.Printf("[INFO] Enabling policy prioritization")
		profileSummary, err = c.EnableDisablePolicyPrioritization(*profileSummary)
		if err != nil {
			return diag.FromErr(err)
		}

		profileId := resourcePolicyPriority.ProfileID

		if resourcePolicyPriority.PolicyOrderingEnabled {
			log.Printf("[INFO] Prioritizing policies:%v", resourcePolicyPriority.PolicyOrder)
			resourcePolicyPriority, err = c.PrioritizePolicies(*resourcePolicyPriority)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		id := rpo.helper.generateUniqueID(profileId)
		d.SetId(id)
	}

	return rpo.resourceRead(ctx, d, m)

}

func (rpo *ResourcePolicyPriority) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileId := rpo.helper.parseUniqueID(d.Id())

	profileSummary, err := c.GetProfileSummary(profileId)
	if err != nil {
		return diag.FromErr(err)
	}

	profileSummary.PolicyOrderingEnabled = false

	log.Printf("[INFO] Disabling policy prioritization: %s", d.Id())
	_, err = c.EnableDisablePolicyPrioritization(*profileSummary)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func (helper *ResourcePolicyPriorityHelper) getAndMapModelToResource(d *schema.ResourceData, policies []britive.ProfilePolicy, profileId string, policyOrderingEnabled bool, imported bool) error {
	if err := d.Set("profile_id", profileId); err != nil {
		return err
	}

	if err := d.Set("policy_priority_enabled", policyOrderingEnabled); err != nil {
		return err
	}

	order := d.Get("policy_priority").(*schema.Set).List()
	var policyOrder []map[string]interface{}

	if len(order) == 0 && imported {
		for _, policy := range policies {

			pOrder := map[string]interface{}{
				"id":       policy.PolicyID,
				"priority": policy.Order,
			}
			policyOrder = append(policyOrder, pOrder)

		}
		if err := d.Set("policy_priority", policyOrder); err != nil {
			return err
		}
		return nil
	}

	userOrder := make(map[string]int)
	for _, ord := range order {
		mapOrder := ord.(map[string]interface{})
		idArr := strings.Split(mapOrder["id"].(string), "/")
		pId := idArr[len(idArr)-1]
		userOrder[pId] = mapOrder["priority"].(int)
	}

	for _, policy := range policies {
		if _, ok := userOrder[policy.PolicyID]; ok {
			pID := helper.generateUniquePolicyID(profileId, policy.PolicyID)
			pOrder := map[string]interface{}{
				"id":       pID,
				"priority": policy.Order,
			}
			policyOrder = append(policyOrder, pOrder)
		}

	}
	if err := d.Set("policy_priority", policyOrder); err != nil {
		return err
	}

	return nil
}

func (helper *ResourcePolicyPriorityHelper) generateUniquePolicyID(profileID string, policyID string) string {
	return fmt.Sprintf("paps/%s/policies/%s", profileID, policyID)
}

func (helper *ResourcePolicyPriorityHelper) mapResourceToModel(c *britive.Client, d *schema.ResourceData, resourcePolicyPriority *britive.ProfilePolicyPriority) (*britive.ProfilePolicyPriority, error) {
	profileId := d.Get("profile_id").(string)
	policyOrder := d.Get("policy_priority").(*schema.Set).List()
	rawPolicyOrdering, _ := d.GetOk("policy_priority_enabled")
	policyOrderingEnabled := rawPolicyOrdering.(bool)

	userMapPolicyToOrder := make(map[string]int)
	userMapOrderToPolicy := make(map[int]string)
	profilePolicies, err := c.GetProfilePolicies(profileId)
	if err != nil {
		return nil, err
	}
	for _, rawPolicy := range policyOrder {
		policy := rawPolicy.(map[string]interface{})
		policyIdArr := strings.Split(policy["id"].(string), "/")
		policy["id"] = policyIdArr[len(policyIdArr)-1]
		if policy["priority"].(int) < 0 || policy["priority"].(int) >= len(profilePolicies) {
			return nil, fmt.Errorf("invalid priority value: %d. The total number of policies is %d, so the priority must be between 0 and %d, inclusive.", policy["priority"].(int), len(profilePolicies), len(profilePolicies)-1)
		}
		if _, ok := userMapPolicyToOrder[policy["id"].(string)]; ok {
			return nil, fmt.Errorf("duplicate policy detected: %s. Each policy ID must be unique.", policy["id"].(string))
		}
		if _, ok := userMapOrderToPolicy[policy["priority"].(int)]; ok {
			return nil, fmt.Errorf("duplicate priority detected: %d. Each priority value must be unique.", policy["priority"].(int))
		}
		userMapOrderToPolicy[policy["priority"].(int)] = policy["id"].(string)
		userMapPolicyToOrder[policy["id"].(string)] = policy["priority"].(int)
	}

	skipped := 0

	checkPolicy := make(map[string]int)

	for i := 0; i < len(profilePolicies); i++ {
		var tempPolicyOrder britive.PolicyOrder

		if _, ok := userMapOrderToPolicy[i]; ok {
			tempPolicyOrder.Id = userMapOrderToPolicy[i]
			tempPolicyOrder.Order = i
		} else {

			policy := profilePolicies[skipped]
			if _, ok := userMapPolicyToOrder[policy.PolicyID]; ok {
				i--
				skipped++
				continue
			}
			tempPolicyOrder.Id = policy.PolicyID
			tempPolicyOrder.Order = i
			skipped++
		}

		if _, ok := checkPolicy[tempPolicyOrder.Id]; ok {
			return nil, fmt.Errorf("duplicate policy detected: [%s] has already been assigned. Each policy ID must be unique.", tempPolicyOrder.Id)
		}
		checkPolicy[tempPolicyOrder.Id] = tempPolicyOrder.Order

		resourcePolicyPriority.PolicyOrder = append(resourcePolicyPriority.PolicyOrder, tempPolicyOrder)
	}

	log.Printf("%v", resourcePolicyPriority.PolicyOrder)

	resourcePolicyPriority.ProfileID = profileId
	resourcePolicyPriority.Extendable = false
	resourcePolicyPriority.PolicyOrderingEnabled = policyOrderingEnabled

	return resourcePolicyPriority, nil
}

func (helper *ResourcePolicyPriorityHelper) parseUniqueID(id string) string {
	idArr := strings.Split(id, "/")
	profileId := idArr[len(idArr)-3]

	return profileId
}

func (helper *ResourcePolicyPriorityHelper) generateUniqueID(profileId string) string {
	return fmt.Sprintf("paps/%s/policies/priority", profileId)
}

func (rpo *ResourcePolicyPriority) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpo.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/policies/priority", "(?P<profile_id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	profileId := d.Get("profile_id").(string)

	d.SetId(rpo.helper.generateUniqueID(profileId))

	policies, err := c.GetProfilePolicies(profileId)
	if err != nil {
		return nil, err
	}

	profile, err := c.GetProfile(profileId)
	if err != nil {
		return nil, err
	}

	err = rpo.helper.getAndMapModelToResource(d, policies, profileId, profile.PolicyOrderingEnabled, true)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
