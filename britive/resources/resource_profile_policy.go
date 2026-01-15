package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceProfilePolicy{}
	_ resource.ResourceWithConfigure   = &ResourceProfilePolicy{}
	_ resource.ResourceWithImportState = &ResourceProfilePolicy{}
)

type ResourceProfilePolicy struct {
	client       *britive_client.Client
	helper       *ResourceProfilePolicyHelper
	importHelper *imports.ImportHelper
}

type ResourceProfilePolicyHelper struct{}

func NewResourceProfilePolicy() resource.Resource {
	return &ResourceProfilePolicy{}
}

func NewResourceProfilePolicyHelper() *ResourceProfilePolicyHelper {
	return &ResourceProfilePolicyHelper{}
}

func (rpp *ResourceProfilePolicy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_profile_policy"
}

func (rpp *ResourceProfilePolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Profile Policy resource")

	if req.ProviderData == nil {
		return
	}

	rpp.client = req.ProviderData.(*britive_client.Client)
	if rpp.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Profile Policy")
	rpp.helper = NewResourceProfilePolicyHelper()
}

func (rpp *ResourceProfilePolicy) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for profile policy resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"profileId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"policy_name": schema.StringAttribute{
				Required:    true,
				Description: "The policy name associated with profile",
				Validators: []validator.String{
					validate.StringFunc(
						"applicationId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The description of the profile policy",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Is policy active",
			},
			"is_draft": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Is policy a draft",
			},
			"is_read_only": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Is the policy read only",
			},
			"consumer": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("papservice"),
				Description: "The consumer service",
			},
			"access_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Allow"),
				Description: "Type of access for profile policy",
			},
			"members": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("{}"),
				Description: "Members of profile policy",
				Validators: []validator.String{
					validate.StringFunc(
						"members",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"condition": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Condition of profile policy",
				Validators: []validator.String{
					validate.StringFunc(
						"members",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				Description: "The list associations for profile policy",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of association, should be one of [Environment, EnvironmentGroup]",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"Environment",
									"EnvironmentGroup",
								),
							},
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The association value",
							Validators: []validator.String{
								validate.StringFunc(
									"assocationValue",
									validate.StringIsNotWhiteSpace(),
								),
							},
						},
					},
				},
			},
		},
	}
}

func (rpp *ResourceProfilePolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_profile_policy")

	var plan britive_client.ProfilePolicyPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during profile_policy creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profilePolicy := britive_client.ProfilePolicy{}

	err := rpp.helper.mapResourceToModel(ctx, plan, &profilePolicy, rpp.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, error: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating profile policy: %#v", profilePolicy))

	pp, err := rpp.client.CreateProfilePolicy(ctx, profilePolicy)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create profile policy, error: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted profile policy: %#v", pp))
	plan.ID = types.StringValue(rpp.helper.generateUniqueID(profilePolicy.ProfileID, pp.PolicyID))

	planPtr, err := rpp.helper.getAndMapModelToPlan(ctx, plan, rpp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile_policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile_policy model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after create", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Create completed and state set", map[string]interface{}{
		"profile_poicy": planPtr,
	})
}

func (rpp *ResourceProfilePolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_profile_policy")

	if rpp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ProfilePolicyPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get profile policy state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rpp.helper.getAndMapModelToPlan(ctx, state, rpp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile policy model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after create", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Read profile policy:  %#v", planPtr))
}

func (rpp *ResourceProfilePolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_profile_policy")

	var plan, state britive_client.ProfilePolicyPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	var hasChanges bool
	if plan.ProfileID.Equal(state.ProfileID) || plan.PolicyName.Equal(state.PolicyName) || plan.Description.Equal(state.Description) || plan.IsActive.Equal(state.IsActive) || plan.IsDraft.Equal(state.IsDraft) || plan.IsReadOnly.Equal(state.IsReadOnly) || plan.Consumer.Equal(state.Consumer) || plan.AccessType.Equal(state.AccessType) || plan.Members.Equal(state.Members) || plan.Condition.Equal(state.Condition) || plan.Associations.Equal(state.Associations) {
		hasChanges = true
		profileID, policyID, err := rpp.helper.parseUniqueID(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse policyID", "Invalid policyID")
			tflog.Error(ctx, fmt.Sprintf("Failed to parse policyID, error:%#v", err))
			return
		}

		profilePolicy := britive_client.ProfilePolicy{}

		err = rpp.helper.mapResourceToModel(ctx, plan, &profilePolicy, rpp.client)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update profile policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, error:%#v", err))
			return
		}

		profilePolicy.PolicyID = policyID
		profilePolicy.ProfileID = profileID

		old_name := state.PolicyName.ValueString()
		oldMem := state.Members.ValueString()
		oldCon := state.Condition.ValueString()
		upp, err := rpp.client.UpdateProfilePolicy(ctx, profilePolicy, old_name)
		if err != nil {
			plan.Members = types.StringValue(oldMem)
			plan.Condition = types.StringValue(oldCon)
			resp.Diagnostics.AddError("Failed to update profile policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update profile policy. error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted Updated profile policy: %#v", upp))
		plan.ID = types.StringValue(rpp.helper.generateUniqueID(profilePolicy.ProfileID, profilePolicy.PolicyID))
	}
	if hasChanges {
		planPtr, err := rpp.helper.getAndMapModelToPlan(ctx, plan, rpp.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map profile policy model to plan", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "Failed to set state after update", map[string]interface{}{
				"diagnostics": resp.Diagnostics,
			})
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Updated profile policy: %#v", planPtr))
	}
}

func (rpp *ResourceProfilePolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_profile_policy")

	var state britive_client.ProfilePolicyPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID, policyID, err := rpp.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning("Failed to delete profile policy", "Failed to parse policyID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse policyID, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting profile policy: %s/%s", profileID, policyID))
	err = rpp.client.DeleteProfilePolicy(ctx, profileID, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete profile policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete profile policy, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleted profile policy %s/%s", profileID, policyID))
	resp.State.RemoveResource(ctx)

}

func (rpp *ResourceProfilePolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rpp.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/policies/(?P<policy_name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<policy_name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import profile policy", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	profileID := importData.Fields["profile_id"]
	policyName := importData.Fields["policy_name"]
	if strings.TrimSpace(profileID) == "" {
		resp.Diagnostics.AddError("Failed to import profile policy", "Invalid profileID")
		tflog.Error(ctx, "Failed to import profile policy, Invalid profileID")
		return
	}
	if strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError("Failed to import profile policy", "Invalid policyName")
		tflog.Error(ctx, "Failed to import profile policy, Invalid policyName")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing profile policy: %s/%s", profileID, policyName))

	policy, err := rpp.client.GetProfilePolicyByName(ctx, profileID, policyName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import profile policy, error:%#v", err))
		return
	}

	plan := &britive_client.ProfilePolicyPlan{
		ID: types.StringValue(rpp.helper.generateUniqueID(profileID, policy.PolicyID)),
	}

	planPtr, err := rpp.helper.getAndMapModelToPlan(ctx, *plan, rpp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import profile policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import profile policy model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after import", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Imported profile policy : %#v", planPtr))
}

func (rpph *ResourceProfilePolicyHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ProfilePolicyPlan, c *britive_client.Client) (*britive_client.ProfilePolicyPlan, error) {
	profileID, policyID, err := rpph.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading profile policy: %s/%s", profileID, policyID))

	profilePolicyId, err := c.GetProfilePolicy(ctx, profileID, policyID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("policy %s in profile %s", policyID, profileID)
	}
	if err != nil {
		return nil, err
	}
	profilePolicy, err := c.GetProfilePolicyByName(ctx, profileID, profilePolicyId.Name)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("policy %s in profile %s", profilePolicy.Name, profileID)
	}
	if err != nil {
		return nil, err
	}

	profilePolicy.ProfileID = profileID

	tflog.Info(ctx, fmt.Sprintf("Received profile policy: %#v", profilePolicy))

	plan.ProfileID = types.StringValue(profilePolicy.ProfileID)
	plan.PolicyName = types.StringValue(profilePolicy.Name)
	plan.Description = types.StringValue(profilePolicy.Description)
	plan.Consumer = types.StringValue(profilePolicy.Consumer)
	plan.AccessType = types.StringValue(profilePolicy.AccessType)
	plan.IsActive = types.BoolValue(profilePolicy.IsActive)
	plan.IsDraft = types.BoolValue(profilePolicy.IsDraft)
	plan.IsReadOnly = types.BoolValue(profilePolicy.IsReadOnly)

	newCon := plan.Condition.ValueString()
	if britive_client.ConditionEqual(profilePolicy.Condition, newCon) {
		plan.Condition = types.StringValue(newCon)
	} else {
		plan.Condition = types.StringValue(profilePolicy.Condition)
	}

	mem, err := json.Marshal(profilePolicy.Members)
	if err != nil {
		return nil, err
	}

	newMem := plan.Members.ValueString()
	if britive_client.MembersEqual(string(mem), newMem) {
		plan.Members = types.StringValue(newMem)
	} else {
		plan.Members = types.StringValue(string(mem))
	}

	associations, err := rpph.mapProfilePolicyAssociationsModelToResource(ctx, plan, profilePolicy.ProfileID, profilePolicy.Associations, c)
	if err != nil {
		return nil, err
	}
	associationSet, err := rpph.mapAssociationsToSet(associations)
	if err != nil {
		return nil, err
	}
	plan.Associations = associationSet

	return &plan, nil
}

func (rpph *ResourceProfilePolicyHelper) mapResourceToModel(ctx context.Context, plan britive_client.ProfilePolicyPlan, profilePolicy *britive_client.ProfilePolicy, c *britive_client.Client) error {
	profilePolicy.ProfileID = plan.ProfileID.ValueString()
	profilePolicy.Name = plan.PolicyName.ValueString()
	profilePolicy.Description = plan.Description.ValueString()
	profilePolicy.Consumer = plan.Consumer.ValueString()
	profilePolicy.AccessType = plan.AccessType.ValueString()
	profilePolicy.IsActive = plan.IsActive.ValueBool()
	profilePolicy.IsDraft = plan.IsDraft.ValueBool()
	profilePolicy.IsReadOnly = plan.IsReadOnly.ValueBool()
	profilePolicy.Condition = plan.Condition.ValueString()
	json.Unmarshal([]byte(plan.Members.ValueString()), &profilePolicy.Members)

	associations, err := rpph.getProfilePolicyAssociations(ctx, plan, profilePolicy.ProfileID, c)
	if err != nil {
		return err
	}
	profilePolicy.Associations = associations

	return nil
}

func (rpph *ResourceProfilePolicyHelper) getProfilePolicyAssociations(ctx context.Context, plan britive_client.ProfilePolicyPlan, profileID string, c *britive_client.Client) ([]britive_client.ProfilePolicyAssociation, error) {
	associationScopes := make([]britive_client.ProfilePolicyAssociation, 0)
	as, err := rpph.mapSetToAssociations(plan.Associations)
	if err != nil {
		return nil, err
	}

	if len(as) == 0 {
		return associationScopes, nil
	}

	appId, err := c.RetrieveAppIdGivenProfileId(ctx, profileID)
	if err != nil {
		return associationScopes, err
	}

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(ctx, appId)
	if err != nil {
		return associationScopes, err
	}
	if appRootEnvironmentGroup == nil {
		return associationScopes, nil
	}
	applicationType, err := c.GetApplicationType(ctx, appId)
	if err != nil {
		return associationScopes, err
	}
	appType := applicationType.ApplicationType
	unmatchedAssociations := make([]britive_client.PolicyAssociationPlan, 0)
	for _, a := range as {
		associationType := a.Type.ValueString()
		associationValue := a.Value.ValueString()
		var rootAssociations []britive_client.Association
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
				newAssociationValue := c.GetEnvId(ctx, appId, associationValue)
				if aeg.ID == newAssociationValue {
					isAssociationExists = true
					associationScopes = rpph.appendProfilePolicyAssociations(associationScopes, associationType, aeg.ID)
					break
				}
			}
		}
		if !isAssociationExists {
			unmatchedAssociations = append(unmatchedAssociations, a)
		}

	}
	if len(unmatchedAssociations) > 0 {
		return nil, errs.NewNotFoundErrorf("associations %v", unmatchedAssociations)
	}
	return associationScopes, nil
}

func (rpph *ResourceProfilePolicyHelper) appendProfilePolicyAssociations(associations []britive_client.ProfilePolicyAssociation, associationType string, associationID string) []britive_client.ProfilePolicyAssociation {
	associations = append(associations, britive_client.ProfilePolicyAssociation{
		Type:  associationType,
		Value: associationID,
	})
	return associations
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

func (rpph *ResourceProfilePolicyHelper) mapProfilePolicyAssociationsModelToResource(ctx context.Context, plan britive_client.ProfilePolicyPlan, profileID string, associations []britive_client.ProfilePolicyAssociation, c *britive_client.Client) ([]britive_client.PolicyAssociationPlan, error) {
	profilePolicyAssociations := make([]britive_client.PolicyAssociationPlan, 0)
	inputAssociations, err := rpph.mapSetToAssociations(plan.Associations)
	if err != nil {
		return nil, err
	}
	if inputAssociations == nil {
		return profilePolicyAssociations, nil
	}

	appId, err := c.RetrieveAppIdGivenProfileId(ctx, profileID)
	if err != nil {
		return profilePolicyAssociations, err
	}

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(ctx, appId)
	if err != nil {
		return profilePolicyAssociations, err
	}
	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return profilePolicyAssociations, nil
	}
	applicationType, err := c.GetApplicationType(ctx, appId)
	if err != nil {
		return profilePolicyAssociations, err
	}
	appType := applicationType.ApplicationType
	for _, association := range associations {
		var rootAssociations []britive_client.Association
		if association.Type == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}
		var a *britive_client.Association
		for _, aeg := range rootAssociations {
			if aeg.ID == association.Value {
				a = &aeg
				break
			}
		}
		if a == nil {
			return profilePolicyAssociations, errs.NewNotFoundErrorf("association %s", association.Value)
		}
		var profilePolicyAssociation britive_client.PolicyAssociationPlan
		associationValue := a.Name
		for _, inputAssociation := range inputAssociations {
			iat := inputAssociation.Type.ValueString()
			iav := inputAssociation.Value.ValueString()
			if association.Type == "EnvironmentGroup" && (appType == "AWS" || appType == "AWS Standalone") && strings.EqualFold("root", a.Name) && strings.EqualFold("root", iav) {
				associationValue = iav
			}
			if association.Type == iat && a.ID == iav {
				associationValue = a.ID
				break
			} else if association.Type == "Environment" && appType == "AWS Standalone" {
				envId := c.GetEnvId(ctx, appId, iav)
				if association.Type == iat && a.ID == envId {
					associationValue = iav
					break
				}
			}
		}
		profilePolicyAssociation.Type = types.StringValue(association.Type)
		profilePolicyAssociation.Value = types.StringValue(associationValue)
		profilePolicyAssociations = append(profilePolicyAssociations, profilePolicyAssociation)

	}
	return profilePolicyAssociations, nil

}

func (rpph *ResourceProfilePolicyHelper) mapAssociationsToSet(associationPlan []britive_client.PolicyAssociationPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(associationPlan))

	for _, p := range associationPlan {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"type":  types.StringType,
				"value": types.StringType,
			},
			map[string]attr.Value{
				"type":  p.Type,
				"value": p.Value,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for profile policy associations: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":  types.StringType,
				"value": types.StringType,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create profile policy associations set: %v", diags)
	}
	return set, nil
}

func (rpph *ResourceProfilePolicyHelper) mapSetToAssociations(associationeSet types.Set) ([]britive_client.PolicyAssociationPlan, error) {
	var result []britive_client.PolicyAssociationPlan
	objs := associationeSet.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected Object, got %T", e)
		}

		var p britive_client.PolicyAssociationPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert object to PolicyAssociations: %v", diags)
		}
		result = append(result, p)
	}
	return result, nil
}
