package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ProfilePolicyPrioritizationResource{}
	_ resource.ResourceWithConfigure      = &ProfilePolicyPrioritizationResource{}
	_ resource.ResourceWithImportState    = &ProfilePolicyPrioritizationResource{}
	_ resource.ResourceWithValidateConfig = &ProfilePolicyPrioritizationResource{}
)

func NewProfilePolicyPrioritizationResource() resource.Resource {
	return &ProfilePolicyPrioritizationResource{}
}

type ProfilePolicyPrioritizationResource struct {
	client *britive.Client
}

type ProfilePolicyPrioritizationResourceModel struct {
	ID                    types.String          `tfsdk:"id"`
	ProfileID             types.String          `tfsdk:"profile_id"`
	PolicyPriorityEnabled types.Bool            `tfsdk:"policy_priority_enabled"`
	PolicyPriority        []PolicyPriorityModel `tfsdk:"policy_priority"`
}

type PolicyPriorityModel struct {
	ID       types.String `tfsdk:"id"`
	Priority types.Int64  `tfsdk:"priority"`
}

func (r *ProfilePolicyPrioritizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile_policy_prioritization"
}

func (r *ProfilePolicyPrioritizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages policy prioritization for a Britive profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the profile policy prioritization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "Profile ID.",
			},
			"policy_priority_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable policy ordering (must be true).",
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"policy_priority": schema.SetNestedBlock{
				Description: "Policies with id and priority.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "Policy Id.",
						},
						"priority": schema.Int64Attribute{
							Required:    true,
							Description: "Priority number.",
						},
					},
				},
			},
		},
	}
}

func (r *ProfilePolicyPrioritizationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ProfilePolicyPrioritizationResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate policy_priority_enabled must be true
	if !data.PolicyPriorityEnabled.IsNull() && !data.PolicyPriorityEnabled.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("policy_priority_enabled"),
			"Invalid Configuration",
			"policy_priority_enabled must be true. Set to true or omit this field to use the default.",
		)
	}
}

func (r *ProfilePolicyPrioritizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProfilePolicyPrioritizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfilePolicyPrioritizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := plan.ProfileID.ValueString()
	policyOrderingEnabled := plan.PolicyPriorityEnabled.ValueBool()

	// Build policy priority model
	resourcePolicyPriority, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Policy Priority",
			fmt.Sprintf("Could not build policy priority: %s", err.Error()),
		)
		return
	}

	// Enable/disable policy prioritization
	profileSummary, err := r.client.GetProfileSummary(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Profile Summary",
			fmt.Sprintf("Could not get profile summary: %s", err.Error()),
		)
		return
	}

	profileSummary.PolicyOrderingEnabled = policyOrderingEnabled

	profileSummary, err = r.client.EnableDisablePolicyPrioritization(*profileSummary)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Enabling Policy Prioritization",
			fmt.Sprintf("Could not enable policy prioritization: %s", err.Error()),
		)
		return
	}

	// Prioritize policies if enabled
	if policyOrderingEnabled {
		_, err = r.client.PrioritizePolicies(*resourcePolicyPriority)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Prioritizing Policies",
				fmt.Sprintf("Could not prioritize policies: %s", err.Error()),
			)
			return
		}
	}

	plan.ID = types.StringValue(generateProfilePolicyPrioritizationID(profileID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan, false); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policy Prioritization",
			fmt.Sprintf("Could not read profile policy prioritization after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePolicyPrioritizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfilePolicyPrioritizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := parseProfilePolicyPrioritizationID(state.ID.ValueString())

	// Get profile policies
	policies, err := r.client.GetProfilePolicies(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policies",
			fmt.Sprintf("Could not read profile policies: %s", err.Error()),
		)
		return
	}

	// Get profile
	profile, err := r.client.GetProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile",
			fmt.Sprintf("Could not read profile: %s", err.Error()),
		)
		return
	}

	state.ProfileID = types.StringValue(profileID)
	state.PolicyPriorityEnabled = types.BoolValue(profile.PolicyOrderingEnabled)

	// Map policies to state
	if err := r.mapPoliciesToState(ctx, &state, policies, false); err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Policies",
			fmt.Sprintf("Could not map policies to state: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfilePolicyPrioritizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfilePolicyPrioritizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := plan.ProfileID.ValueString()
	policyOrderingEnabled := plan.PolicyPriorityEnabled.ValueBool()

	// Build policy priority model
	resourcePolicyPriority, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Policy Priority",
			fmt.Sprintf("Could not build policy priority: %s", err.Error()),
		)
		return
	}

	// Enable/disable policy prioritization
	profileSummary, err := r.client.GetProfileSummary(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Profile Summary",
			fmt.Sprintf("Could not get profile summary: %s", err.Error()),
		)
		return
	}

	profileSummary.PolicyOrderingEnabled = policyOrderingEnabled

	profileSummary, err = r.client.EnableDisablePolicyPrioritization(*profileSummary)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Policy Prioritization",
			fmt.Sprintf("Could not update policy prioritization: %s", err.Error()),
		)
		return
	}

	// Prioritize policies if enabled
	if policyOrderingEnabled {
		_, err = r.client.PrioritizePolicies(*resourcePolicyPriority)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Prioritizing Policies",
				fmt.Sprintf("Could not prioritize policies: %s", err.Error()),
			)
			return
		}
	}

	plan.ID = types.StringValue(generateProfilePolicyPrioritizationID(profileID))

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan, false); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Policy Prioritization",
			fmt.Sprintf("Could not read profile policy prioritization after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePolicyPrioritizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfilePolicyPrioritizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := parseProfilePolicyPrioritizationID(state.ID.ValueString())

	// Disable policy prioritization
	profileSummary, err := r.client.GetProfileSummary(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Profile Summary",
			fmt.Sprintf("Could not get profile summary: %s", err.Error()),
		)
		return
	}

	profileSummary.PolicyOrderingEnabled = false

	_, err = r.client.EnableDisablePolicyPrioritization(*profileSummary)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Disabling Policy Prioritization",
			fmt.Sprintf("Could not disable policy prioritization: %s", err.Error()),
		)
		return
	}
}

func (r *ProfilePolicyPrioritizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. paps/{profile_id}/policies/priority
	// 2. {profile_id}
	idRegexes := []string{
		`^paps/(?P<profile_id>[^/]+)/policies/priority$`,
		`^(?P<profile_id>[^/]+)$`,
	}

	var profileID string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "profile_id" && i < len(matches) {
					profileID = matches[i]
					break
				}
			}
			if profileID != "" {
				break
			}
		}
	}

	if profileID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'paps/{profile_id}/policies/priority' or '{profile_id}'", req.ID),
		)
		return
	}

	// Get profile policies
	policies, err := r.client.GetProfilePolicies(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Policy Prioritization",
			fmt.Sprintf("Could not get profile policies: %s", err.Error()),
		)
		return
	}

	// Get profile
	profile, err := r.client.GetProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Policy Prioritization",
			fmt.Sprintf("Could not get profile: %s", err.Error()),
		)
		return
	}

	// Build state
	var state ProfilePolicyPrioritizationResourceModel
	state.ID = types.StringValue(generateProfilePolicyPrioritizationID(profileID))
	state.ProfileID = types.StringValue(profileID)
	state.PolicyPriorityEnabled = types.BoolValue(profile.PolicyOrderingEnabled)
	state.PolicyPriority = nil

	// Map policies (import=true to get all policies)
	if err := r.mapPoliciesToState(ctx, &state, policies, true); err != nil {
		resp.Diagnostics.AddError(
			"Error Mapping Policies",
			fmt.Sprintf("Could not map policies during import: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// mapResourceToModel builds the API model from the resource model with complex ordering logic
func (r *ProfilePolicyPrioritizationResource) mapResourceToModel(ctx context.Context, plan *ProfilePolicyPrioritizationResourceModel) (*britive.ProfilePolicyPriority, error) {
	profileID := plan.ProfileID.ValueString()
	policyOrderingEnabled := plan.PolicyPriorityEnabled.ValueBool()

	// Get all profile policies
	profilePolicies, err := r.client.GetProfilePolicies(profileID)
	if err != nil {
		return nil, err
	}

	// Use user-specified policy priorities directly (slice, no ElementsAs needed)
	userPolicyPriorities := plan.PolicyPriority

	// Create maps for validation and lookup
	userMapPolicyToOrder := make(map[string]int)
	userMapOrderToPolicy := make(map[int]string)

	for _, userPolicy := range userPolicyPriorities {
		policyIDArr := strings.Split(userPolicy.ID.ValueString(), "/")
		policyID := policyIDArr[len(policyIDArr)-1]
		priority := int(userPolicy.Priority.ValueInt64())

		// Validate priority range
		if priority < 0 || priority >= len(profilePolicies) {
			return nil, fmt.Errorf("invalid priority value: %d. The total number of policies is %d, so the priority must be between 0 and %d, inclusive", priority, len(profilePolicies), len(profilePolicies)-1)
		}

		// Check for duplicate policies
		if _, ok := userMapPolicyToOrder[policyID]; ok {
			return nil, fmt.Errorf("duplicate policy detected: %s. Each policy ID must be unique", policyID)
		}

		// Check for duplicate priorities
		if _, ok := userMapOrderToPolicy[priority]; ok {
			return nil, fmt.Errorf("duplicate priority detected: %d. Each priority value must be unique", priority)
		}

		userMapOrderToPolicy[priority] = policyID
		userMapPolicyToOrder[policyID] = priority
	}

	// Build complete policy order (including unspecified policies)
	resourcePolicyPriority := &britive.ProfilePolicyPriority{
		ProfileID:             profileID,
		PolicyOrderingEnabled: policyOrderingEnabled,
	}

	skipped := 0
	checkPolicy := make(map[string]int)

	for i := 0; i < len(profilePolicies); i++ {
		var tempPolicyOrder britive.PolicyOrder

		if policyID, ok := userMapOrderToPolicy[i]; ok {
			// User specified this priority
			tempPolicyOrder.Id = policyID
			tempPolicyOrder.Order = i
		} else {
			// Fill in with unspecified policy
			policy := profilePolicies[skipped]
			if _, ok := userMapPolicyToOrder[policy.PolicyID]; ok {
				// This policy was assigned a different priority, skip it
				i--
				skipped++
				continue
			}
			tempPolicyOrder.Id = policy.PolicyID
			tempPolicyOrder.Order = i
			skipped++
		}

		// Check for duplicates
		if _, ok := checkPolicy[tempPolicyOrder.Id]; ok {
			return nil, fmt.Errorf("duplicate policy detected: [%s] has already been assigned. Each policy ID must be unique", tempPolicyOrder.Id)
		}
		checkPolicy[tempPolicyOrder.Id] = tempPolicyOrder.Order

		resourcePolicyPriority.PolicyOrder = append(resourcePolicyPriority.PolicyOrder, tempPolicyOrder)
	}

	return resourcePolicyPriority, nil
}

// mapPoliciesToState maps API policies back to state
func (r *ProfilePolicyPrioritizationResource) mapPoliciesToState(ctx context.Context, state *ProfilePolicyPrioritizationResourceModel, policies []britive.ProfilePolicy, imported bool) error {
	var userPolicyPriorities []PolicyPriorityModel

	// During import or when no existing priorities, return all policies
	if imported || len(state.PolicyPriority) == 0 {
		for _, policy := range policies {
			userPolicyPriorities = append(userPolicyPriorities, PolicyPriorityModel{
				ID:       types.StringValue(policy.PolicyID),
				Priority: types.Int64Value(int64(policy.Order)),
			})
		}
	} else {
		// During normal operation, preserve user's policy ID format
		existingPriorities := state.PolicyPriority

		// Build map of policy ID to user's original ID format
		userOrder := make(map[string]string)
		for _, existing := range existingPriorities {
			idArr := strings.Split(existing.ID.ValueString(), "/")
			policyID := idArr[len(idArr)-1]
			userOrder[policyID] = existing.ID.ValueString()
		}

		// Map policies back, preserving user's format
		for _, policy := range policies {
			if userID, ok := userOrder[policy.PolicyID]; ok {
				userPolicyPriorities = append(userPolicyPriorities, PolicyPriorityModel{
					ID:       types.StringValue(userID),
					Priority: types.Int64Value(int64(policy.Order)),
				})
			}
		}
	}

	state.PolicyPriority = userPolicyPriorities
	return nil
}

// populateStateFromAPI fetches data from API and populates state
func (r *ProfilePolicyPrioritizationResource) populateStateFromAPI(ctx context.Context, state *ProfilePolicyPrioritizationResourceModel, imported bool) error {
	profileID := parseProfilePolicyPrioritizationID(state.ID.ValueString())

	// Get profile policies
	policies, err := r.client.GetProfilePolicies(profileID)
	if err != nil {
		return err
	}

	// Get profile
	profile, err := r.client.GetProfile(profileID)
	if err != nil {
		return err
	}

	state.PolicyPriorityEnabled = types.BoolValue(profile.PolicyOrderingEnabled)

	return r.mapPoliciesToState(ctx, state, policies, imported)
}

// Helper functions
func generateProfilePolicyPrioritizationID(profileID string) string {
	return fmt.Sprintf("paps/%s/policies/priority", profileID)
}

func parseProfilePolicyPrioritizationID(id string) string {
	idArr := strings.Split(id, "/")
	return idArr[len(idArr)-3]
}
