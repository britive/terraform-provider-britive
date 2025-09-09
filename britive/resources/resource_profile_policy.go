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

// ResourceProfilePolicy - Terraform Resource for Profile Policy
type ResourceProfilePolicy struct {
	Resource     *schema.Resource
	helper       *ResourceProfilePolicyHelper
	importHelper *imports.ImportHelper
}

// NewResourceProfilePolicy - Initialization of new profile policy resource
func NewResourceProfilePolicy(importHelper *imports.ImportHelper) *ResourceProfilePolicy {
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
			"associations": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The list of associations for the Britive profile policy",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The type of association, should be one of [Environment, EnvironmentGroup]",
							ValidateFunc: validation.StringInSlice([]string{"Environment", "EnvironmentGroup"}, false),
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The association value",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
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
	if d.HasChange("profile_id") || d.HasChange("policy_name") || d.HasChange("description") || d.HasChange("is_active") || d.HasChange("is_draft") || d.HasChange("is_read_only") || d.HasChange("consumer") || d.HasChange("access_type") || d.HasChange("members") || d.HasChange("condition") || d.HasChange("associations") {
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

		old_name, _ := d.GetChange("policy_name")
		oldMem, _ := d.GetChange("members")
		oldCon, _ := d.GetChange("condition")
		upp, err := c.UpdateProfilePolicy(profilePolicy, old_name.(string))
		if err != nil {
			if errState := d.Set("members", oldMem.(string)); errState != nil {
				return diag.FromErr(errState)
			}
			if errState := d.Set("condition", oldCon.(string)); errState != nil {
				return diag.FromErr(errState)
			}
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
		return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_id")
	}
	if strings.TrimSpace(policyName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("policy_name")
	}

	log.Printf("[INFO] Importing profile policy: %s/%s", profileID, policyName)

	policy, err := c.GetProfilePolicyByName(profileID, policyName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("policy %s for profile %s", policyName, profileID)
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

	associations, err := rpph.getProfilePolicyAssociations(profilePolicy.ProfileID, d, m)
	if err != nil {
		return err
	}
	profilePolicy.Associations = associations

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
		return errs.NewNotFoundErrorf("policy %s in profile %s", policyID, profileID)
	}
	if err != nil {
		return err
	}
	profilePolicy, err := c.GetProfilePolicyByName(profileID, profilePolicyId.Name)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("policy %s in profile %s", profilePolicy.Name, profileID)
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

	newCon := d.Get("condition")
	if britive.ConditionEqual(profilePolicy.Condition, newCon.(string)) {
		if err := d.Set("condition", newCon.(string)); err != nil {
			return err
		}
	} else if err := d.Set("condition", profilePolicy.Condition); err != nil {
		return err
	}

	mem, err := json.Marshal(profilePolicy.Members)
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

	associations, err := rpph.mapProfilePolicyAssociationsModelToResource(profilePolicy.ProfileID, profilePolicy.Associations, d, m)
	if err != nil {
		return err
	}
	if err := d.Set("associations", associations); err != nil {
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
		err = errs.NewInvalidResourceIDError("profile policy", ID)
		return
	}

	profileID = profilePolicyParts[1]
	policyID = profilePolicyParts[3]
	return
}

//endregion

func (rpph *ResourceProfilePolicyHelper) appendProfilePolicyAssociations(associations []britive.ProfilePolicyAssociation, associationType string, associationID string) []britive.ProfilePolicyAssociation {
	associations = append(associations, britive.ProfilePolicyAssociation{
		Type:  associationType,
		Value: associationID,
	})
	return associations
}

func (rpph *ResourceProfilePolicyHelper) getProfilePolicyAssociations(profileID string, d *schema.ResourceData, m interface{}) ([]britive.ProfilePolicyAssociation, error) {
	c := m.(*britive.Client)
	associationScopes := make([]britive.ProfilePolicyAssociation, 0)
	as := d.Get("associations").(*schema.Set)
	if as == nil {
		return associationScopes, nil
	}

	appId, err := c.RetrieveAppIdGivenProfileId(profileID)
	if err != nil {
		return associationScopes, err
	}

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(appId)
	if err != nil {
		return associationScopes, err
	}
	if appRootEnvironmentGroup == nil {
		return associationScopes, nil
	}
	applicationType, err := c.GetApplicationType(appId)
	if err != nil {
		return associationScopes, err
	}
	appType := applicationType.ApplicationType
	unmatchedAssociations := make([]interface{}, 0)
	for _, a := range as.List() {
		s := a.(map[string]interface{})
		associationType := s["type"].(string)
		associationValue := s["value"].(string)
		var rootAssociations []britive.Association
		isAssociationExists := false
		if associationType == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
			if appType == "AWS" && strings.EqualFold("root", associationValue) {
				associationValue = "Root"
			} else if appType == "AWS Standalone" && strings.EqualFold("root", associationValue) {
				associationValue = "root"
			}
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}
		for _, aeg := range rootAssociations {
			if aeg.Name == associationValue || aeg.ID == associationValue {
				isAssociationExists = true
				associationScopes = rpph.appendProfilePolicyAssociations(associationScopes, associationType, aeg.ID)
				break
			} else if associationType == "Environment" && appType == "AWS Standalone" {
				newAssociationValue := c.GetEnvId(appId, associationValue)
				if aeg.ID == newAssociationValue {
					isAssociationExists = true
					associationScopes = rpph.appendProfilePolicyAssociations(associationScopes, associationType, aeg.ID)
					break
				}
			}
		}
		if !isAssociationExists {
			unmatchedAssociations = append(unmatchedAssociations, s)
		}

	}
	if len(unmatchedAssociations) > 0 {
		return nil, errs.NewNotFoundErrorf("associations %v", unmatchedAssociations)
	}
	return associationScopes, nil
}

func (rpph *ResourceProfilePolicyHelper) mapProfilePolicyAssociationsModelToResource(profileID string, associations []britive.ProfilePolicyAssociation, d *schema.ResourceData, m interface{}) ([]interface{}, error) {
	c := m.(*britive.Client)
	profilePolicyAssociations := make([]interface{}, 0)
	inputAssociations := d.Get("associations").(*schema.Set)
	if inputAssociations == nil {
		return profilePolicyAssociations, nil
	}

	appId, err := c.RetrieveAppIdGivenProfileId(profileID)
	if err != nil {
		return profilePolicyAssociations, err
	}

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(appId)
	if err != nil {
		return profilePolicyAssociations, err
	}
	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return profilePolicyAssociations, nil
	}
	applicationType, err := c.GetApplicationType(appId)
	if err != nil {
		return profilePolicyAssociations, err
	}
	appType := applicationType.ApplicationType
	for _, association := range associations {
		var rootAssociations []britive.Association
		if association.Type == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}
		var a *britive.Association
		for _, aeg := range rootAssociations {
			if aeg.ID == association.Value {
				a = &aeg
				break
			}
		}
		if a == nil {
			return profilePolicyAssociations, errs.NewNotFoundErrorf("association %s", association.Value)
		}
		profilePolicyAssociation := make(map[string]interface{})
		associationValue := a.Name
		for _, inputAssociation := range inputAssociations.List() {
			ia := inputAssociation.(map[string]interface{})
			iat := ia["type"].(string)
			iav := ia["value"].(string)
			if association.Type == "EnvironmentGroup" && (appType == "AWS" || appType == "AWS Standalone") && strings.EqualFold("root", a.Name) && strings.EqualFold("root", iav) {
				associationValue = iav
			}
			if association.Type == iat && a.ID == iav {
				associationValue = a.ID
				break
			} else if association.Type == "Environment" && appType == "AWS Standalone" {
				envId := c.GetEnvId(appId, iav)
				if association.Type == iat && a.ID == envId {
					associationValue = iav
					break
				}
			}
		}
		profilePolicyAssociation["type"] = association.Type
		profilePolicyAssociation["value"] = associationValue
		profilePolicyAssociations = append(profilePolicyAssociations, profilePolicyAssociation)

	}
	return profilePolicyAssociations, nil

}
