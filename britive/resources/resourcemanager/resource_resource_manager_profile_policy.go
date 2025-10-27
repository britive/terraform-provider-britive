package resourcemanager

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

type ResourceResourceManagerProfilePolicy struct {
	Resource     *schema.Resource
	helper       *ResourceResourceManagerProfilePolicyHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerProfilePolicyHelper struct{}

func NewResourceResourceManagerProfilePolicyHelper() *ResourceResourceManagerProfilePolicyHelper {
	return &ResourceResourceManagerProfilePolicyHelper{}
}

func NewResourceResourceManagerProfilePolicy(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResourceManagerProfilePolicy {
	rrmpp := &ResourceResourceManagerProfilePolicy{
		validation:   v,
		importHelper: importHelper,
		helper:       NewResourceResourceManagerProfilePolicyHelper(),
	}
	rrmpp.Resource = &schema.Resource{
		CreateContext: rrmpp.resourceCreate,
		UpdateContext: rrmpp.resourceUpdate,
		ReadContext:   rrmpp.resourceRead,
		DeleteContext: rrmpp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rrmpp.resourceStateImporter,
		},
		CustomizeDiff: rrmpp.validation.ValidateImmutableFields([]string{
			"profile_id",
		}),
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
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
				Default:     "",
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
				Default:     "resourceprofile",
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
			"resource_labels": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Resource labels for policy",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label_key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of resource label",
						},
						"values": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "List of values of resource label",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
	return rrmpp
}

func (rrmpp *ResourceResourceManagerProfilePolicy) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	resourceManagerProfilePolicy := &britive.ResourceManagerProfilePolicy{}

	err := rrmpp.helper.mapResourceToModel(d, resourceManagerProfilePolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new resource manager profile policy: %#v", resourceManagerProfilePolicy)

	resourceManagerProfilePolicy, err = c.CreateUpdateResourceManagerProfilePolicy(*resourceManagerProfilePolicy, "", false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile policy: %#v", resourceManagerProfilePolicy)
	d.SetId(rrmpp.helper.generateUniqueID(resourceManagerProfilePolicy.ProfileID, resourceManagerProfilePolicy.PolicyID))
	return rrmpp.resourceRead(ctx, d, m)
}

func (rrmpp *ResourceResourceManagerProfilePolicy) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, policyID := rrmpp.helper.parseUniqueID(d.Id())

	log.Printf("[INFO] Reading resource manager profile policy: %s/%s", profileID, policyID)

	policyName := d.Get("policy_name").(string)

	resourceManagerProfilePolicy, err := c.GetResourceManagerProfilePolicy(profileID, policyName)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received resource manager profile policy: %#v", resourceManagerProfilePolicy)

	resourceManagerProfilePolicy.ProfileID = d.Get("profile_id").(string)
	err = rrmpp.helper.getAndMapModelToResource(d, resourceManagerProfilePolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rrmpp *ResourceResourceManagerProfilePolicy) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	if d.HasChange("profile_id") || d.HasChange("policy_name") || d.HasChange("description") || d.HasChange("is_active") || d.HasChange("is_draft") || d.HasChange("is_read_only") || d.HasChange("consumer") || d.HasChange("access_type") || d.HasChange("members") || d.HasChange("condition") || d.HasChange("resource_labels") {
		profileID, policyID := rrmpp.helper.parseUniqueID(d.Id())

		resourceManagerProfilePolicy := &britive.ResourceManagerProfilePolicy{}

		err := rrmpp.helper.mapResourceToModel(d, resourceManagerProfilePolicy)
		if err != nil {
			return diag.FromErr(err)
		}

		resourceManagerProfilePolicy.PolicyID = policyID
		resourceManagerProfilePolicy.ProfileID = profileID

		old_name, _ := d.GetChange("policy_name")
		oldMem, _ := d.GetChange("members")
		oldCon, _ := d.GetChange("condition")
		upp, err := c.CreateUpdateResourceManagerProfilePolicy(*resourceManagerProfilePolicy, old_name.(string), true)
		if err != nil {
			if errState := d.Set("members", oldMem.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			if errState := d.Set("condition", oldCon.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted Updated resource manager profile policy: %#v", upp)
	}

	return rrmpp.resourceRead(ctx, d, m)
}

func (rrmpp *ResourceResourceManagerProfilePolicy) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	profileID, policyID := rrmpp.helper.parseUniqueID(d.Id())

	log.Printf("[INFO] Deleting resource manager profile policy, %s/%s", profileID, policyID)

	err := c.DeleteResourceManagerProfilePolicy(profileID, policyID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	log.Printf("[INFO] Deleted resource manager profile policy, %s/%s", profileID, policyID)
	return diags
}

func (rrmpp *ResourceResourceManagerProfilePolicy) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rrmpp.importHelper.ParseImportID([]string{"resource-manager/profiles/(?P<profile_id>[^/]+)/policies/(?P<policy_name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<policy_name>[^/]+)"}, d); err != nil {
		return nil, err
	}

	profileID := d.Get("profile_id").(string)
	policyName := d.Get("policy_name").(string)
	if strings.TrimSpace(profileID) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_id")
	}
	if strings.TrimSpace(policyName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("policy_name")
	}

	log.Printf("[INFO] Importing resource manager profile policy: %s/%s", profileID, policyName)

	policy, err := c.GetResourceManagerProfilePolicy(profileID, policyName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("policy %s for profile %s", policyName, profileID)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rrmpp.helper.generateUniqueID(profileID, policy.PolicyID))
	log.Printf("[INFO] Imported resource manager profile policy: %s/%s", profileID, policyName)
	return []*schema.ResourceData{d}, nil
}

func (helper *ResourceResourceManagerProfilePolicyHelper) mapResourceToModel(d *schema.ResourceData, resourceManagerProfilePolicy *britive.ResourceManagerProfilePolicy) error {
	profIdArr := strings.Split(d.Get("profile_id").(string), "/")
	resourceManagerProfilePolicy.ProfileID = profIdArr[len(profIdArr)-1]
	resourceManagerProfilePolicy.Name = d.Get("policy_name").(string)
	if val, ok := d.GetOk("access_type"); ok {
		resourceManagerProfilePolicy.AccessType = val.(string)
	}
	if val, ok := d.GetOk("description"); ok {
		resourceManagerProfilePolicy.Description = val.(string)
	}
	if val, ok := d.GetOk("is_active"); ok {
		resourceManagerProfilePolicy.IsActive = val.(bool)
	}
	if val, ok := d.GetOk("is_draft"); ok {
		resourceManagerProfilePolicy.IsDraft = val.(bool)
	}
	if val, ok := d.GetOk("is_read_only"); ok {
		resourceManagerProfilePolicy.IsReadOnly = val.(bool)
	}
	if val, ok := d.GetOk("consumer"); ok {
		resourceManagerProfilePolicy.Consumer = val.(string)
	}
	err := json.Unmarshal([]byte(d.Get("members").(string)), &resourceManagerProfilePolicy.Members)
	if err != nil {
		return err
	}
	if val, ok := d.GetOk("condition"); ok {
		resourceManagerProfilePolicy.Condition = val.(string)
	}
	var resourceLabels []interface{}
	if val, ok := d.GetOk("resource_labels"); ok {
		resourceLabels = val.(*schema.Set).List()
	}
	resourceLabelsMap := make(map[string][]string)
	for _, label := range resourceLabels {
		labelMap := label.(map[string]interface{})
		labelName := labelMap["label_key"].(string)
		userLabelValuesList := labelMap["values"].(*schema.Set).List()
		labelValues := make([]string, len(userLabelValuesList))
		for i, value := range userLabelValuesList {
			labelValues[i] = value.(string)
		}
		resourceLabelsMap[labelName] = labelValues
	}
	resourceManagerProfilePolicy.ResourceLabels = resourceLabelsMap

	return nil
}

func (helper *ResourceResourceManagerProfilePolicyHelper) getAndMapModelToResource(d *schema.ResourceData, resourceManagerProfilePolicy *britive.ResourceManagerProfilePolicy) error {
	if err := d.Set("profile_id", resourceManagerProfilePolicy.ProfileID); err != nil {
		return err
	}
	if err := d.Set("policy_name", resourceManagerProfilePolicy.Name); err != nil {
		return err
	}
	if err := d.Set("description", resourceManagerProfilePolicy.Description); err != nil {
		return err
	}
	if err := d.Set("consumer", resourceManagerProfilePolicy.Consumer); err != nil {
		return err
	}
	if err := d.Set("access_type", resourceManagerProfilePolicy.AccessType); err != nil {
		return err
	}
	if err := d.Set("is_active", resourceManagerProfilePolicy.IsActive); err != nil {
		return err
	}
	if err := d.Set("is_draft", resourceManagerProfilePolicy.IsDraft); err != nil {
		return err
	}
	if err := d.Set("is_read_only", resourceManagerProfilePolicy.IsReadOnly); err != nil {
		return err
	}

	log.Printf("=========== okay profile_id : %s", resourceManagerProfilePolicy.ProfileID)

	apiCon := ""
	if resourceManagerProfilePolicy.Condition != "" {
		var condMap interface{}
		if err := json.Unmarshal([]byte(resourceManagerProfilePolicy.Condition), &condMap); err != nil {
			log.Printf("====== unmarshal error : %s", resourceManagerProfilePolicy.Condition)
			return err
		}
		normalizedCondition, err := json.Marshal(condMap)
		if err != nil {
			log.Printf("======= marshal error : %v", condMap)
			return err
		}
		apiCon = string(normalizedCondition)
	}

	newCon := d.Get("condition")
	if britive.ConditionEqual(apiCon, newCon.(string)) {
		if err := d.Set("condition", newCon.(string)); err != nil {
			return err
		}
	} else if err := d.Set("condition", apiCon); err != nil {
		return err
	}

	mem, err := json.Marshal(resourceManagerProfilePolicy.Members)
	if err != nil {
		return err
	}

	log.Printf("======== okay condition")

	newMem := d.Get("members")
	if britive.MembersEqual(string(mem), newMem.(string)) {
		if err := d.Set("members", newMem.(string)); err != nil {
			return err
		}
		log.Printf("========= okay members : %s", newMem.(string))
	} else if err := d.Set("members", string(mem)); err != nil {
		return err
	}
	log.Printf("========= okay members : %s", string(mem))

	var resourceLabelsList []map[string]interface{}
	for name, values := range resourceManagerProfilePolicy.ResourceLabels {
		resourceLabelMap := map[string]interface{}{
			"label_key": name,
			"values":    values,
		}
		resourceLabelsList = append(resourceLabelsList, resourceLabelMap)
	}

	log.Printf("==== all okay")

	return nil
}

func (helper *ResourceResourceManagerProfilePolicyHelper) generateUniqueID(profileID, policyID string) string {
	return fmt.Sprintf("resource-manager/profiles/%s/policies/%s", profileID, policyID)
}

func (helper *ResourceResourceManagerProfilePolicyHelper) parseUniqueID(id string) (string, string) {
	idArr := strings.Split(id, "/")
	length := len(idArr)
	return idArr[length-3], idArr[length-1]
}
