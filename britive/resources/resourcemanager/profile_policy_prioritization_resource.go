package resourcemanager

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
	_ resource.Resource                   = &RMProfilePolicyPrioritizationResource{}
	_ resource.ResourceWithConfigure      = &RMProfilePolicyPrioritizationResource{}
	_ resource.ResourceWithImportState    = &RMProfilePolicyPrioritizationResource{}
	_ resource.ResourceWithValidateConfig = &RMProfilePolicyPrioritizationResource{}
)

func NewRMProfilePolicyPrioritizationResource() resource.Resource {
	return &RMProfilePolicyPrioritizationResource{}
}

type RMProfilePolicyPrioritizationResource struct {
	client *britive.Client
}

type RMProfilePolicyPrioritizationModel struct {
	ID                    types.String            `tfsdk:"id"`
	ProfileID             types.String            `tfsdk:"profile_id"`
	PolicyPriorityEnabled types.Bool              `tfsdk:"policy_priority_enabled"`
	PolicyPriority        []RMPolicyPriorityModel `tfsdk:"policy_priority"`
}

type RMPolicyPriorityModel struct {
	ID       types.String `tfsdk:"id"`
	Priority types.Int64  `tfsdk:"priority"`
}

func (r *RMProfilePolicyPrioritizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_profile_policy_prioritization"
}

func (r *RMProfilePolicyPrioritizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages policy prioritization for a Britive resource manager profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the resource manager profile policy prioritization.",
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
							Description: "Policy ID.",
						},
						"priority": schema.Int64Attribute{
							Required:    true,
							Description: "Priority number (0-based).",
						},
					},
				},
			},
		},
	}
}

func (r *RMProfilePolicyPrioritizationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data RMProfilePolicyPrioritizationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.PolicyPriorityEnabled.IsNull() && !data.PolicyPriorityEnabled.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("policy_priority_enabled"),
			"Invalid Configuration",
			"policy_priority_enabled must be true. Set to true or omit this field to use the default.",
		)
	}
}

func (r *RMProfilePolicyPrioritizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *RMProfilePolicyPrioritizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RMProfilePolicyPrioritizationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := parseRMProfileID(plan.ProfileID.ValueString())
	policyOrderingEnabled := plan.PolicyPriorityEnabled.ValueBool()

	if err := r.client.EnableDisableResourceManagerPolicyPrioritization(profileID, policyOrderingEnabled); err != nil {
		resp.Diagnostics.AddError("Error Enabling Policy Prioritization", err.Error())
		return
	}

	if policyOrderingEnabled {
		priority, err := r.buildPriorityModel(ctx, &plan, profileID)
		if err != nil {
			resp.Diagnostics.AddError("Error Building Policy Priority", err.Error())
			return
		}
		if _, err := r.client.ResourceManagerPrioritizeProfilePolicies(*priority); err != nil {
			resp.Diagnostics.AddError("Error Prioritizing Policies", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(generateRMPolicyPrioritizationID(profileID))

	if err := r.populateState(ctx, &plan, false); err != nil {
		resp.Diagnostics.AddError("Error Reading Policy Prioritization", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RMProfilePolicyPrioritizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RMProfilePolicyPrioritizationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.populateState(ctx, &state, false); err != nil {
		resp.Diagnostics.AddError("Error Reading Policy Prioritization", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RMProfilePolicyPrioritizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RMProfilePolicyPrioritizationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := parseRMProfileID(plan.ProfileID.ValueString())
	policyOrderingEnabled := plan.PolicyPriorityEnabled.ValueBool()

	if err := r.client.EnableDisableResourceManagerPolicyPrioritization(profileID, policyOrderingEnabled); err != nil {
		resp.Diagnostics.AddError("Error Updating Policy Prioritization", err.Error())
		return
	}

	if policyOrderingEnabled {
		priority, err := r.buildPriorityModel(ctx, &plan, profileID)
		if err != nil {
			resp.Diagnostics.AddError("Error Building Policy Priority", err.Error())
			return
		}
		if _, err := r.client.ResourceManagerPrioritizeProfilePolicies(*priority); err != nil {
			resp.Diagnostics.AddError("Error Prioritizing Policies", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(generateRMPolicyPrioritizationID(profileID))

	if err := r.populateState(ctx, &plan, false); err != nil {
		resp.Diagnostics.AddError("Error Reading Policy Prioritization", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RMProfilePolicyPrioritizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RMProfilePolicyPrioritizationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := parseRMPolicyPrioritizationID(state.ID.ValueString())

	if err := r.client.EnableDisableResourceManagerPolicyPrioritization(profileID, false); err != nil {
		resp.Diagnostics.AddError("Error Disabling Policy Prioritization", err.Error())
	}
}

func (r *RMProfilePolicyPrioritizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idRegexes := []string{
		`^resource-manager/(?P<profile_id>[^/]+)/policies/priority$`,
		`^(?P<profile_id>[^/]+)$`,
	}

	var profileID string
	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, name := range re.SubexpNames() {
				if name == "profile_id" && i < len(matches) {
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
			fmt.Sprintf("Import ID %q must match 'resource-manager/{profile_id}/policies/priority' or '{profile_id}'", req.ID),
		)
		return
	}

	profile, err := r.client.GetResourceManagerProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Profile", err.Error())
		return
	}
	if !profile.PolicyOrderingEnabled {
		resp.Diagnostics.AddError("Cannot Import", "Policy ordering is disabled for this profile.")
		return
	}

	var state RMProfilePolicyPrioritizationModel
	state.ID = types.StringValue(generateRMPolicyPrioritizationID(profileID))
	state.ProfileID = types.StringValue(profileID)
	state.PolicyPriorityEnabled = types.BoolValue(true)

	if err := r.populateState(ctx, &state, true); err != nil {
		resp.Diagnostics.AddError("Error Reading Policy Prioritization", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildPriorityModel constructs the API model applying the fill-in ordering logic:
// user-specified policies get their declared priorities; remaining policies fill gaps in order.
func (r *RMProfilePolicyPrioritizationResource) buildPriorityModel(_ context.Context, plan *RMProfilePolicyPrioritizationModel, profileID string) (*britive.ProfilePolicyPriority, error) {
	profilePolicies, err := r.client.GetResourceManagerProfilePolicies(profileID)
	if err != nil {
		return nil, err
	}

	userMapPolicyToOrder := make(map[string]int)
	userMapOrderToPolicy := make(map[int]string)

	for _, p := range plan.PolicyPriority {
		idArr := strings.Split(p.ID.ValueString(), "/")
		policyID := idArr[len(idArr)-1]
		priority := int(p.Priority.ValueInt64())

		if priority < 0 || priority >= len(profilePolicies) {
			return nil, fmt.Errorf("invalid priority %d (total policies: %d, must be 0–%d)", priority, len(profilePolicies), len(profilePolicies)-1)
		}
		if _, dup := userMapPolicyToOrder[policyID]; dup {
			return nil, fmt.Errorf("duplicate policy ID: %s", policyID)
		}
		if _, dup := userMapOrderToPolicy[priority]; dup {
			return nil, fmt.Errorf("duplicate priority: %d", priority)
		}
		userMapOrderToPolicy[priority] = policyID
		userMapPolicyToOrder[policyID] = priority
	}

	result := &britive.ProfilePolicyPriority{
		ProfileID:             profileID,
		PolicyOrderingEnabled: plan.PolicyPriorityEnabled.ValueBool(),
	}

	skipped := 0
	checkPolicy := make(map[string]int)
	for i := 0; i < len(profilePolicies); i++ {
		var entry britive.PolicyOrder
		if policyID, ok := userMapOrderToPolicy[i]; ok {
			entry.Id = policyID
			entry.Order = i
		} else {
			policy := profilePolicies[skipped]
			if _, ok := userMapPolicyToOrder[policy.PolicyID]; ok {
				i--
				skipped++
				continue
			}
			entry.Id = policy.PolicyID
			entry.Order = i
			skipped++
		}
		if _, dup := checkPolicy[entry.Id]; dup {
			return nil, fmt.Errorf("duplicate policy in final order: %s", entry.Id)
		}
		checkPolicy[entry.Id] = entry.Order
		result.PolicyOrder = append(result.PolicyOrder, entry)
	}

	return result, nil
}

// populateState fetches current API state and writes it into the model.
func (r *RMProfilePolicyPrioritizationResource) populateState(_ context.Context, state *RMProfilePolicyPrioritizationModel, imported bool) error {
	profileID := parseRMPolicyPrioritizationID(state.ID.ValueString())

	policies, err := r.client.GetResourceManagerProfilePolicies(profileID)
	if err != nil {
		return err
	}

	profile, err := r.client.GetResourceManagerProfile(profileID)
	if err != nil {
		return err
	}

	state.PolicyPriorityEnabled = types.BoolValue(profile.PolicyOrderingEnabled)

	if imported || len(state.PolicyPriority) == 0 {
		var result []RMPolicyPriorityModel
		for _, p := range policies {
			result = append(result, RMPolicyPriorityModel{
				ID:       types.StringValue(p.PolicyID),
				Priority: types.Int64Value(int64(p.Order)),
			})
		}
		state.PolicyPriority = result
		return nil
	}

	// Preserve user's policy ID format (full path vs bare ID)
	userOrder := make(map[string]string)
	for _, existing := range state.PolicyPriority {
		idArr := strings.Split(existing.ID.ValueString(), "/")
		policyID := idArr[len(idArr)-1]
		userOrder[policyID] = existing.ID.ValueString()
	}

	var result []RMPolicyPriorityModel
	for _, p := range policies {
		if userID, ok := userOrder[p.PolicyID]; ok {
			result = append(result, RMPolicyPriorityModel{
				ID:       types.StringValue(userID),
				Priority: types.Int64Value(int64(p.Order)),
			})
		}
	}
	state.PolicyPriority = result
	return nil
}

func generateRMPolicyPrioritizationID(profileID string) string {
	return fmt.Sprintf("resource-manager/%s/policies/priority", profileID)
}

// parseRMPolicyPrioritizationID extracts the profile ID from "resource-manager/{id}/policies/priority".
func parseRMPolicyPrioritizationID(id string) string {
	idArr := strings.Split(id, "/")
	if len(idArr) >= 4 {
		return idArr[len(idArr)-3]
	}
	return id
}

// parseRMProfileID extracts the bare profile ID from a full resource path or bare ID.
func parseRMProfileID(id string) string {
	idArr := strings.Split(id, "/")
	return idArr[len(idArr)-1]
}
