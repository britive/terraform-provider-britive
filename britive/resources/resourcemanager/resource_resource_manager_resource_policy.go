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

type ResourceResourcePolicy struct {
	Resource     *schema.Resource
	validation   *validate.Validation
	importHelper *imports.ImportHelper
	helper       *ResourceResourcePolicyHelper
}

type ResourceResourcePolicyHelper struct {
}

func NewResourceResourcePolicyHelper() *ResourceResourcePolicyHelper {
	return &ResourceResourcePolicyHelper{}
}

func NewResourceResourcePolicy(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResourcePolicy {
	rrp := &ResourceResourcePolicy{
		validation:   v,
		importHelper: importHelper,
		helper:       NewResourceResourcePolicyHelper(),
	}
	rrp.Resource = &schema.Resource{
		CreateContext: rrp.resourceCreate,
		ReadContext:   rrp.resourceRead,
		UpdateContext: rrp.resourceUpdate,
		DeleteContext: rrp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rrp.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
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
				Default:     "resourcemanager",
				Description: "The consumer service",
			},
			"access_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Allow",
				Description: "Type of access for the policy",
			},
			"access_level": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Level of access for the policy",
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
	return rrp
}

func (rrp *ResourceResourcePolicy) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	resourcePolicy := &britive.ResourceManagerResourcePolicy{}
	err := rrp.helper.mapResourceToModel(d, resourcePolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new resource manager resource policy: %#v", resourcePolicy)

	resourcePolicy, err = c.CreateUpdateResourceManagerResourcePolicy(*resourcePolicy, "", false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new resource policy: %#v", resourcePolicy)
	d.SetId(rrp.helper.generateUniqueID(resourcePolicy.PolicyID))
	return rrp.resourceRead(ctx, d, m)
}

func (rrp *ResourceResourcePolicy) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	policyID := rrp.helper.parseUniqueID(d.Id())

	policyName := d.Get("policy_name").(string)

	log.Printf("[INFO] Reading resource manager resource policy: %s", policyID)

	resourceManagerProfilePolicy, err := c.GetResourceManagerResourcePolicy(policyName)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received resource manager resource policy: %#v", resourceManagerProfilePolicy)

	err = rrp.helper.getAndMapModelToResource(d, resourceManagerProfilePolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rrp *ResourceResourcePolicy) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	if d.HasChange("profile_id") || d.HasChange("policy_name") || d.HasChange("description") || d.HasChange("is_active") || d.HasChange("is_draft") || d.HasChange("is_read_only") || d.HasChange("consumer") || d.HasChange("access_type") || d.HasChange("access_level") || d.HasChange("members") || d.HasChange("condition") || d.HasChange("resource_labels") {
		policyID := rrp.helper.parseUniqueID(d.Id())

		resourcepolicy := &britive.ResourceManagerResourcePolicy{}

		err := rrp.helper.mapResourceToModel(d, resourcepolicy)
		if err != nil {
			return diag.FromErr(err)
		}

		resourcepolicy.PolicyID = policyID

		old_name, _ := d.GetChange("policy_name")
		oldMem, _ := d.GetChange("members")
		oldCon, _ := d.GetChange("condition")
		upp, err := c.CreateUpdateResourceManagerResourcePolicy(*resourcepolicy, old_name.(string), true)
		if err != nil {
			if errState := d.Set("members", oldMem.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			if errState := d.Set("condition", oldCon.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted Updated resource manager resource policy: %#v", upp)
	}

	return rrp.resourceRead(ctx, d, m)
}

func (rrp *ResourceResourcePolicy) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	policyID := rrp.helper.parseUniqueID(d.Id())

	log.Printf("[INFO] Deleting resource manager resource policy, %s", policyID)

	err := c.DeleteResourceManagerResourcePolicy(policyID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	log.Printf("[INFO] Deleted resource manager resource policy, %s", policyID)
	return diags
}

func (rrp *ResourceResourcePolicy) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	if err := rrp.importHelper.ParseImportID([]string{"resource-manager/policies/(?P<policy_name>[^/]+)", "(?P<policy_name>[^/]+)"}, d); err != nil {
		return nil, err
	}

	policyName := d.Get("policy_name").(string)
	if strings.TrimSpace(policyName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("policy_name")
	}

	log.Printf("[INFO] Importing resource manager resource policy: %s", policyName)

	resourcePolicy, err := c.GetResourceManagerResourcePolicy(policyName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("policy %s", policyName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rrp.helper.generateUniqueID(resourcePolicy.PolicyID))
	log.Printf("[INFO] Imported resource manager resource policy: %s", policyName)
	return []*schema.ResourceData{d}, nil
}

func (helper *ResourceResourcePolicyHelper) mapResourceToModel(d *schema.ResourceData, resourcePolicy *britive.ResourceManagerResourcePolicy) error {
	resourcePolicy.Name = d.Get("policy_name").(string)
	if val, ok := d.GetOk("access_type"); ok {
		resourcePolicy.AccessType = val.(string)
	}
	if val, ok := d.GetOk("access_level"); ok {
		resourcePolicy.AccessLevel = val.(string)
	}
	if val, ok := d.GetOk("description"); ok {
		resourcePolicy.Description = val.(string)
	}
	if val, ok := d.GetOk("is_active"); ok {
		resourcePolicy.IsActive = val.(bool)
	}
	if val, ok := d.GetOk("is_draft"); ok {
		resourcePolicy.IsDraft = val.(bool)
	}
	if val, ok := d.GetOk("is_read_only"); ok {
		resourcePolicy.IsReadOnly = val.(bool)
	}
	if val, ok := d.GetOk("consumer"); ok {
		resourcePolicy.Consumer = val.(string)
	}
	err := json.Unmarshal([]byte(d.Get("members").(string)), &resourcePolicy.Members)
	if err != nil {
		return err
	}
	if val, ok := d.GetOk("condition"); ok {
		resourcePolicy.Condition = val.(string)
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
	resourcePolicy.ResourceLabels = resourceLabelsMap

	return nil
}

func (helper *ResourceResourcePolicyHelper) getAndMapModelToResource(d *schema.ResourceData, resourcePolicy *britive.ResourceManagerResourcePolicy) error {
	if err := d.Set("policy_name", resourcePolicy.Name); err != nil {
		return err
	}
	if err := d.Set("description", resourcePolicy.Description); err != nil {
		return err
	}
	if err := d.Set("consumer", resourcePolicy.Consumer); err != nil {
		return err
	}
	if err := d.Set("access_type", resourcePolicy.AccessType); err != nil {
		return err
	}
	if err := d.Set("access_level", resourcePolicy.AccessLevel); err != nil {
		return err
	}
	if err := d.Set("is_active", resourcePolicy.IsActive); err != nil {
		return err
	}
	if err := d.Set("is_draft", resourcePolicy.IsDraft); err != nil {
		return err
	}
	if err := d.Set("is_read_only", resourcePolicy.IsReadOnly); err != nil {
		return err
	}

	normalizedCondition := ""
	if resourcePolicy.Condition != "" {
		var condMap interface{}
		if err := json.Unmarshal([]byte(resourcePolicy.Condition), &condMap); err != nil {
			return err
		}
		apiCon, err := json.Marshal(condMap)
		if err != nil {
			return err
		}
		normalizedCondition = string(apiCon)
	}

	newCon := d.Get("condition")
	if britive.ConditionEqual(normalizedCondition, newCon.(string)) {
		if err := d.Set("condition", newCon.(string)); err != nil {
			return err
		}
	} else if err := d.Set("condition", normalizedCondition); err != nil {
		return err
	}

	mem, err := json.Marshal(resourcePolicy.Members)
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

	var resourceLabelsList []map[string]interface{}
	for name, values := range resourcePolicy.ResourceLabels {
		resourceLabelMap := map[string]interface{}{
			"label_key": name,
			"values":    values,
		}
		resourceLabelsList = append(resourceLabelsList, resourceLabelMap)
	}

	return nil
}

func (helper *ResourceResourcePolicyHelper) generateUniqueID(policyID string) string {
	return fmt.Sprintf("resource-manager/policies/%s", policyID)
}

func (helper *ResourceResourcePolicyHelper) parseUniqueID(id string) string {
	idArr := strings.Split(id, "/")
	return idArr[len(idArr)-1]
}
