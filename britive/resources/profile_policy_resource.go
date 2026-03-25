package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ProfilePolicyResource{}
	_ resource.ResourceWithConfigure   = &ProfilePolicyResource{}
	_ resource.ResourceWithImportState = &ProfilePolicyResource{}
)

func NewProfilePolicyResource() resource.Resource {
	return &ProfilePolicyResource{}
}

type ProfilePolicyResource struct {
	client *britive.Client
}

type ProfilePolicyResourceModel struct {
	ID           types.String                   `tfsdk:"id"`
	ProfileID    types.String                   `tfsdk:"profile_id"`
	PolicyName   types.String                   `tfsdk:"policy_name"`
	Description  types.String                   `tfsdk:"description"`
	IsActive     types.Bool                     `tfsdk:"is_active"`
	IsDraft      types.Bool                     `tfsdk:"is_draft"`
	IsReadOnly   types.Bool                     `tfsdk:"is_read_only"`
	Consumer     types.String                   `tfsdk:"consumer"`
	AccessType   types.String                   `tfsdk:"access_type"`
	Members      types.String                   `tfsdk:"members"`
	Condition    types.String                   `tfsdk:"condition"`
	Associations []ProfilePolicyAssociationModel `tfsdk:"associations"`
}

type ProfilePolicyAssociationModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

func (r *ProfilePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile_policy"
}

func (r *ProfilePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive profile policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the profile policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_name": schema.StringAttribute{
				Required:    true,
				Description: "The policy associated with the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The description of the profile policy.",
				Default:     stringdefault.StaticString(""),
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Is the policy active.",
				Default:     booldefault.StaticBool(true),
			},
			"is_draft": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Is the policy a draft.",
				Default:     booldefault.StaticBool(false),
			},
			"is_read_only": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Is the policy read only.",
				Default:     booldefault.StaticBool(false),
			},
			"consumer": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The consumer service.",
				Default:     stringdefault.StaticString("papservice"),
			},
			"access_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Type of access for the policy.",
				Default:     stringdefault.StaticString("Allow"),
			},
			"members": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Members of the policy (JSON string).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Default: stringdefault.StaticString("{}"),
			},
			"condition": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Condition of the policy.",
				Default:     stringdefault.StaticString(""),
			},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				Description: "The list of associations for the Britive profile policy.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of association, should be one of [Environment, EnvironmentGroup].",
							Validators: []validator.String{
								stringvalidator.OneOf("Environment", "EnvironmentGroup"),
							},
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The association value.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ProfilePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ProfilePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfilePolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profilePolicy := britive.ProfilePolicy{
		ProfileID:   plan.ProfileID.ValueString(),
		Name:        plan.PolicyName.ValueString(),
		Description: plan.Description.ValueString(),
		Consumer:    plan.Consumer.ValueString(),
		AccessType:  plan.AccessType.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		IsDraft:     plan.IsDraft.ValueBool(),
		IsReadOnly:  plan.IsReadOnly.ValueBool(),
		Condition:   plan.Condition.ValueString(),
	}

	// Unmarshal members
	if err := json.Unmarshal([]byte(plan.Members.ValueString()), &profilePolicy.Members); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Members JSON",
			fmt.Sprintf("Could not parse members JSON: %s", err.Error()),
		)
		return
	}

	// Get associations
	associations, err := r.getProfilePolicyAssociations(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resolving Associations",
			fmt.Sprintf("Could not resolve associations: %s", err.Error()),
		)
		return
	}
	profilePolicy.Associations = associations

	created, err := r.client.CreateProfilePolicy(profilePolicy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Profile Policy",
			fmt.Sprintf("Could not create profile policy: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateProfilePolicyID(profilePolicy.ProfileID, created.PolicyID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policy",
			fmt.Sprintf("Could not read profile policy after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfilePolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, policyID, err := parseProfilePolicyID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Policy ID",
			fmt.Sprintf("Could not parse profile policy ID: %s", err.Error()),
		)
		return
	}

	// Get policy by ID to get the name
	policyInfo, err := r.client.GetProfilePolicy(profileID, policyID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policy",
			fmt.Sprintf("Could not read profile policy %s/%s: %s", profileID, policyID, err.Error()),
		)
		return
	}

	// Get full policy details by name
	profilePolicy, err := r.client.GetProfilePolicyByName(profileID, policyInfo.Name)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policy",
			fmt.Sprintf("Could not read profile policy '%s' in profile '%s': %s", policyInfo.Name, profileID, err.Error()),
		)
		return
	}

	profilePolicy.ProfileID = profileID

	state.ProfileID = types.StringValue(profilePolicy.ProfileID)
	state.PolicyName = types.StringValue(profilePolicy.Name)
	state.Description = types.StringValue(profilePolicy.Description)
	state.Consumer = types.StringValue(profilePolicy.Consumer)
	state.AccessType = types.StringValue(profilePolicy.AccessType)
	state.IsActive = types.BoolValue(profilePolicy.IsActive)
	state.IsDraft = types.BoolValue(profilePolicy.IsDraft)
	state.IsReadOnly = types.BoolValue(profilePolicy.IsReadOnly)

	// Handle condition with comparison
	newCondition := state.Condition.ValueString()
	if britive.ConditionEqual(profilePolicy.Condition, newCondition) {
		state.Condition = types.StringValue(newCondition)
	} else {
		state.Condition = types.StringValue(profilePolicy.Condition)
	}

	// Handle members with comparison
	membersJSON, err := json.Marshal(profilePolicy.Members)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Marshaling Members",
			fmt.Sprintf("Could not marshal members: %s", err.Error()),
		)
		return
	}
	newMembers := state.Members.ValueString()
	if britive.MembersEqual(string(membersJSON), newMembers) {
		state.Members = types.StringValue(newMembers)
	} else {
		state.Members = types.StringValue(string(membersJSON))
	}

	// Map associations
	associations, err := r.mapProfilePolicyAssociationsModelToResource(ctx, &state, profilePolicy.Associations)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Associations",
			fmt.Sprintf("Could not map associations: %s", err.Error()),
		)
		return
	}
	state.Associations = associations

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfilePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfilePolicyResourceModel
	var state ProfilePolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, policyID, err := parseProfilePolicyID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Policy ID",
			fmt.Sprintf("Could not parse profile policy ID: %s", err.Error()),
		)
		return
	}

	profilePolicy := britive.ProfilePolicy{
		PolicyID:    policyID,
		ProfileID:   profileID,
		Name:        plan.PolicyName.ValueString(),
		Description: plan.Description.ValueString(),
		Consumer:    plan.Consumer.ValueString(),
		AccessType:  plan.AccessType.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		IsDraft:     plan.IsDraft.ValueBool(),
		IsReadOnly:  plan.IsReadOnly.ValueBool(),
		Condition:   plan.Condition.ValueString(),
	}

	// Unmarshal members
	if err := json.Unmarshal([]byte(plan.Members.ValueString()), &profilePolicy.Members); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Members JSON",
			fmt.Sprintf("Could not parse members JSON: %s", err.Error()),
		)
		return
	}

	// Get associations
	associations, err := r.getProfilePolicyAssociations(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resolving Associations",
			fmt.Sprintf("Could not resolve associations: %s", err.Error()),
		)
		return
	}
	profilePolicy.Associations = associations

	// Get old name for update
	oldName := state.PolicyName.ValueString()

	_, err = r.client.UpdateProfilePolicy(profilePolicy, oldName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Profile Policy",
			fmt.Sprintf("Could not update profile policy: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateProfilePolicyID(profileID, policyID))

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policy",
			fmt.Sprintf("Could not read profile policy after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfilePolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, policyID, err := parseProfilePolicyID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Policy ID",
			fmt.Sprintf("Could not parse profile policy ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeleteProfilePolicy(profileID, policyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Profile Policy",
			fmt.Sprintf("Could not delete profile policy %s/%s: %s", profileID, policyID, err.Error()),
		)
		return
	}
}

func (r *ProfilePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. paps/{profile_id}/policies/{policy_name}
	// 2. {profile_id}/{policy_name}
	idRegexes := []string{
		`^paps/(?P<profile_id>[^/]+)/policies/(?P<policy_name>.+)$`,
		`^(?P<profile_id>[^/]+)/(?P<policy_name>.+)$`,
	}

	var profileID, policyName string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if i == 0 {
					continue
				}
				switch matchName {
				case "profile_id":
					profileID = matches[i]
				case "policy_name":
					policyName = matches[i]
				}
			}
			break
		}
	}

	if profileID == "" || policyName == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats", req.ID),
		)
		return
	}

	if strings.TrimSpace(profileID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "profile_id cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "policy_name cannot be empty or whitespace")
		return
	}

	// Get profile policy by name
	policy, err := r.client.GetProfilePolicyByName(profileID, policyName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Profile Policy Not Found",
			fmt.Sprintf("Policy '%s' not found in profile '%s'.", policyName, profileID),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Policy",
			fmt.Sprintf("Could not import profile policy: %s", err.Error()),
		)
		return
	}

	policy.ProfileID = profileID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateProfilePolicyID(profileID, policy.PolicyID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_name"), policy.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), policy.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("consumer"), policy.Consumer)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_type"), policy.AccessType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_active"), policy.IsActive)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_draft"), policy.IsDraft)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_read_only"), policy.IsReadOnly)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("condition"), policy.Condition)...)

	// Marshal members
	membersJSON, _ := json.Marshal(policy.Members)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("members"), string(membersJSON))...)

	// Create a temporary state model to map associations
	var state ProfilePolicyResourceModel
	state.ID = types.StringValue(generateProfilePolicyID(profileID, policy.PolicyID))
	state.ProfileID = types.StringValue(profileID)
	state.Associations = nil

	// Map associations
	associations, err := r.mapProfilePolicyAssociationsModelToResource(ctx, &state, policy.Associations)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Associations",
			fmt.Sprintf("Could not map associations during import: %s", err.Error()),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("associations"), associations)...)
}

// getProfilePolicyAssociations resolves association names/IDs to actual IDs
func (r *ProfilePolicyResource) getProfilePolicyAssociations(ctx context.Context, plan *ProfilePolicyResourceModel) ([]britive.ProfilePolicyAssociation, error) {
	associationScopes := make([]britive.ProfilePolicyAssociation, 0)

	if len(plan.Associations) == 0 {
		return associationScopes, nil
	}

	profileID := plan.ProfileID.ValueString()

	appID, err := r.client.RetrieveAppIdGivenProfileId(profileID)
	if err != nil {
		return nil, err
	}

	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(appID)
	if err != nil {
		return nil, err
	}
	if appRootEnvironmentGroup == nil {
		return associationScopes, nil
	}

	applicationType, err := r.client.GetApplicationType(appID)
	if err != nil {
		return nil, err
	}
	appType := applicationType.ApplicationType

	unmatchedAssociations := make([]ProfilePolicyAssociationModel, 0)

	for _, assoc := range plan.Associations {
		associationType := assoc.Type.ValueString()
		associationValue := assoc.Value.ValueString()

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
				associationScopes = append(associationScopes, britive.ProfilePolicyAssociation{
					Type:  associationType,
					Value: aeg.ID,
				})
				break
			} else if associationType == "Environment" && appType == "AWS Standalone" {
				newAssociationValue := r.client.GetEnvId(appID, associationValue)
				if aeg.ID == newAssociationValue {
					isAssociationExists = true
					associationScopes = append(associationScopes, britive.ProfilePolicyAssociation{
						Type:  associationType,
						Value: aeg.ID,
					})
					break
				}
			}
		}

		if !isAssociationExists {
			unmatchedAssociations = append(unmatchedAssociations, assoc)
		}
	}

	if len(unmatchedAssociations) > 0 {
		return nil, fmt.Errorf("associations not found: %v", unmatchedAssociations)
	}

	return associationScopes, nil
}

// mapProfilePolicyAssociationsModelToResource maps API associations back to resource model
func (r *ProfilePolicyResource) mapProfilePolicyAssociationsModelToResource(ctx context.Context, state *ProfilePolicyResourceModel, associations []britive.ProfilePolicyAssociation) ([]ProfilePolicyAssociationModel, error) {
	if len(associations) == 0 {
		return nil, nil
	}

	profileID := state.ProfileID.ValueString()

	appID, err := r.client.RetrieveAppIdGivenProfileId(profileID)
	if err != nil {
		return nil, err
	}

	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(appID)
	if err != nil {
		return nil, err
	}
	if appRootEnvironmentGroup == nil {
		return nil, nil
	}

	applicationType, err := r.client.GetApplicationType(appID)
	if err != nil {
		return nil, err
	}
	appType := applicationType.ApplicationType

	// Get input associations from state (already a slice)
	inputAssociations := state.Associations

	profilePolicyAssociations := make([]ProfilePolicyAssociationModel, 0)

	for _, association := range associations {
		var rootAssociations []britive.Association
		if association.Type == "EnvironmentGroup" {
			rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
		} else {
			rootAssociations = appRootEnvironmentGroup.Environments
		}

		var matchedAssoc *britive.Association
		for _, aeg := range rootAssociations {
			if aeg.ID == association.Value {
				matchedAssoc = &aeg
				break
			}
		}

		if matchedAssoc == nil {
			return nil, fmt.Errorf("association %s not found", association.Value)
		}

		associationValue := matchedAssoc.Name

		// Try to preserve user's original input
		for _, inputAssoc := range inputAssociations {
			iat := inputAssoc.Type.ValueString()
			iav := inputAssoc.Value.ValueString()

			if association.Type == "EnvironmentGroup" && (appType == "AWS" || appType == "AWS Standalone") && strings.EqualFold("root", matchedAssoc.Name) && strings.EqualFold("root", iav) {
				associationValue = iav
			}

			if association.Type == iat && matchedAssoc.ID == iav {
				associationValue = matchedAssoc.ID
				break
			} else if association.Type == "Environment" && appType == "AWS Standalone" {
				envID := r.client.GetEnvId(appID, iav)
				if association.Type == iat && matchedAssoc.ID == envID {
					associationValue = iav
					break
				}
			}
		}

		profilePolicyAssociations = append(profilePolicyAssociations, ProfilePolicyAssociationModel{
			Type:  types.StringValue(association.Type),
			Value: types.StringValue(associationValue),
		})
	}

	return profilePolicyAssociations, nil
}

// populateStateFromAPI fetches profile policy data from API and populates the state model
func (r *ProfilePolicyResource) populateStateFromAPI(ctx context.Context, state *ProfilePolicyResourceModel) error {
	profileID, policyID, err := parseProfilePolicyID(state.ID.ValueString())
	if err != nil {
		return err
	}

	// Get policy by ID to get the name
	policyInfo, err := r.client.GetProfilePolicy(profileID, policyID)
	if err != nil {
		return err
	}

	// Get full policy details by name
	profilePolicy, err := r.client.GetProfilePolicyByName(profileID, policyInfo.Name)
	if err != nil {
		return err
	}

	profilePolicy.ProfileID = profileID

	state.ProfileID = types.StringValue(profilePolicy.ProfileID)
	state.PolicyName = types.StringValue(profilePolicy.Name)
	state.Description = types.StringValue(profilePolicy.Description)
	state.Consumer = types.StringValue(profilePolicy.Consumer)
	state.AccessType = types.StringValue(profilePolicy.AccessType)
	state.IsActive = types.BoolValue(profilePolicy.IsActive)
	state.IsDraft = types.BoolValue(profilePolicy.IsDraft)
	state.IsReadOnly = types.BoolValue(profilePolicy.IsReadOnly)

	// Handle condition with comparison
	newCondition := state.Condition.ValueString()
	if britive.ConditionEqual(profilePolicy.Condition, newCondition) {
		state.Condition = types.StringValue(newCondition)
	} else {
		state.Condition = types.StringValue(profilePolicy.Condition)
	}

	// Handle members with comparison
	membersJSON, err := json.Marshal(profilePolicy.Members)
	if err != nil {
		return err
	}
	newMembers := state.Members.ValueString()
	if britive.MembersEqual(string(membersJSON), newMembers) {
		state.Members = types.StringValue(newMembers)
	} else {
		state.Members = types.StringValue(string(membersJSON))
	}

	// Map associations
	associations, err := r.mapProfilePolicyAssociationsModelToResource(ctx, state, profilePolicy.Associations)
	if err != nil {
		return err
	}
	state.Associations = associations

	return nil
}

// Helper functions
func generateProfilePolicyID(profileID, policyID string) string {
	return fmt.Sprintf("paps/%s/policies/%s", profileID, policyID)
}

func parseProfilePolicyID(id string) (profileID, policyID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 4 {
		err = fmt.Errorf("invalid profile policy ID format: %s", id)
		return
	}
	profileID = parts[1]
	policyID = parts[3]
	return
}
