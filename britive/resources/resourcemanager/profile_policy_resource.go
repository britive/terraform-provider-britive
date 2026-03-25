package resourcemanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProfilePolicyResource struct {
	client *britive.Client
}

type ProfilePolicyResourceModel struct {
	ID             types.String               `tfsdk:"id"`
	ProfileID      types.String               `tfsdk:"profile_id"`
	PolicyName     types.String               `tfsdk:"policy_name"`
	Description    types.String               `tfsdk:"description"`
	IsActive       types.Bool                 `tfsdk:"is_active"`
	IsDraft        types.Bool                 `tfsdk:"is_draft"`
	IsReadOnly     types.Bool                 `tfsdk:"is_read_only"`
	Consumer       types.String               `tfsdk:"consumer"`
	AccessType     types.String               `tfsdk:"access_type"`
	Members        types.String               `tfsdk:"members"`
	Condition      types.String               `tfsdk:"condition"`
	ResourceLabels []ProfilePolicyLabelModel  `tfsdk:"resource_labels"`
}

func NewProfilePolicyResource() resource.Resource {
	return &ProfilePolicyResource{}
}

func (r *ProfilePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_profile_policy"
}

func (r *ProfilePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager profile policy",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
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
			},
			"policy_name": schema.StringAttribute{
				Required:    true,
				Description: "The policy associated with the profile",
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
				Description: "Is the policy active",
			},
			"is_draft": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Is the policy a draft",
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
				Default:     stringdefault.StaticString("resourceprofile"),
				Description: "The consumer service",
			},
			"access_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Allow"),
				Description: "Type of access for the policy",
			},
			"members": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("{}"),
				Description: "Members of the policy (JSON string)",
			},
			"condition": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Condition of the policy (JSON string)",
			},
		},
		Blocks: map[string]schema.Block{
			"resource_labels": schema.SetNestedBlock{
				Description: "Resource labels for policy",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"label_key": schema.StringAttribute{
							Required:    true,
							Description: "Name of resource label",
						},
						"values": schema.SetAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "List of values of resource label",
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
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
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

	resourceManagerProfilePolicy, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}

	log.Printf("[INFO] Creating new resource manager profile policy: %#v", resourceManagerProfilePolicy)

	created, err := r.client.CreateUpdateResourceManagerProfilePolicy(resourceManagerProfilePolicy, "", false)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Manager Profile Policy", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new profile policy: %#v", created)

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/profiles/%s/policies/%s", created.ProfileID, created.PolicyID))

	// Read back to get computed values
	readResp, err := r.client.GetResourceManagerProfilePolicy(created.ProfileID, created.Name)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile Policy", err.Error())
		return
	}
	readResp.ProfileID = plan.ProfileID.ValueString()

	r.mapModelToResource(ctx, readResp, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfilePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, _ := r.parseUniqueID(state.ID.ValueString())
	policyName := state.PolicyName.ValueString()

	log.Printf("[INFO] Reading resource manager profile policy: %s/%s", profileID, policyName)

	resourceManagerProfilePolicy, err := r.client.GetResourceManagerProfilePolicy(profileID, policyName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile Policy", err.Error())
		return
	}

	log.Printf("[INFO] Received resource manager profile policy: %#v", resourceManagerProfilePolicy)

	resourceManagerProfilePolicy.ProfileID = state.ProfileID.ValueString()

	r.mapModelToResource(ctx, resourceManagerProfilePolicy, &state, &resp.Diagnostics)

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

	profileID, policyID := r.parseUniqueID(state.ID.ValueString())

	resourceManagerProfilePolicy, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}
	resourceManagerProfilePolicy.PolicyID = policyID
	resourceManagerProfilePolicy.ProfileID = profileID

	oldPolicyName := state.PolicyName.ValueString()

	log.Printf("[INFO] Updating resource manager profile policy: %s/%s", profileID, policyID)

	_, err = r.client.CreateUpdateResourceManagerProfilePolicy(resourceManagerProfilePolicy, oldPolicyName, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource Manager Profile Policy", err.Error())
		return
	}

	log.Printf("[INFO] Submitted Updated resource manager profile policy: %s/%s", profileID, policyID)

	// Read back to get updated values
	readResp, err := r.client.GetResourceManagerProfilePolicy(profileID, plan.PolicyName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile Policy", err.Error())
		return
	}
	readResp.ProfileID = plan.ProfileID.ValueString()

	r.mapModelToResource(ctx, readResp, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfilePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, policyID := r.parseUniqueID(state.ID.ValueString())

	log.Printf("[INFO] Deleting resource manager profile policy: %s/%s", profileID, policyID)

	err := r.client.DeleteResourceManagerProfilePolicy(profileID, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Manager Profile Policy", err.Error())
		return
	}

	log.Printf("[INFO] Deleted resource manager profile policy: %s/%s", profileID, policyID)
}

func (r *ProfilePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var profileID, policyName string

	// Support two formats: "resource-manager/profiles/{profile_id}/policies/{policy_name}" or "{profile_id}/{policy_name}"
	if strings.HasPrefix(importID, "resource-manager/profiles/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 5 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/profiles/{profile_id}/policies/{policy_name}' or '{profile_id}/{policy_name}', got: %s", importID))
			return
		}
		profileID = parts[2]
		policyName = parts[4]
	} else {
		parts := strings.Split(importID, "/")
		if len(parts) != 2 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/profiles/{profile_id}/policies/{policy_name}' or '{profile_id}/{policy_name}', got: %s", importID))
			return
		}
		profileID = parts[0]
		policyName = parts[1]
	}

	if strings.TrimSpace(profileID) == "" || strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Profile ID and Policy name cannot be empty")
		return
	}

	log.Printf("[INFO] Importing resource manager profile policy: %s/%s", profileID, policyName)

	policy, err := r.client.GetResourceManagerProfilePolicy(profileID, policyName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Manager Profile Policy Not Found", fmt.Sprintf("Policy %s for profile %s not found", policyName, profileID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Manager Profile Policy", err.Error())
		return
	}

	var state ProfilePolicyResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/profiles/%s/policies/%s", profileID, policy.PolicyID))
	state.ProfileID = types.StringValue(profileID)
	policy.ProfileID = profileID

	r.mapModelToResource(ctx, policy, &state, &resp.Diagnostics)

	log.Printf("[INFO] Imported resource manager profile policy: %s/%s", profileID, policyName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ProfilePolicyResource) mapResourceToModel(ctx context.Context, plan *ProfilePolicyResourceModel) (britive.ResourceManagerProfilePolicy, error) {
	// Extract actual profile ID from potential composite path
	rawProfileID := plan.ProfileID.ValueString()
	profIDArr := strings.Split(rawProfileID, "/")
	profileID := profIDArr[len(profIDArr)-1]

	policy := britive.ResourceManagerProfilePolicy{
		ProfileID:   profileID,
		Name:        plan.PolicyName.ValueString(),
		Description: plan.Description.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		IsDraft:     plan.IsDraft.ValueBool(),
		IsReadOnly:  plan.IsReadOnly.ValueBool(),
		Consumer:    plan.Consumer.ValueString(),
		AccessType:  plan.AccessType.ValueString(),
		Condition:   plan.Condition.ValueString(),
	}

	// Parse members JSON
	if !plan.Members.IsNull() && plan.Members.ValueString() != "" {
		var members interface{}
		err := json.Unmarshal([]byte(plan.Members.ValueString()), &members)
		if err != nil {
			return policy, fmt.Errorf("error parsing members JSON: %v", err)
		}
		policy.Members = members
	}

	// Parse resource labels directly (slice, no ElementsAs needed)
	if len(plan.ResourceLabels) > 0 {
		policy.ResourceLabels = make(map[string][]string)
		for _, label := range plan.ResourceLabels {
			var values []string
			diagsVals := label.Values.ElementsAs(ctx, &values, false)
			if diagsVals.HasError() {
				return policy, fmt.Errorf("error parsing resource label values")
			}
			policy.ResourceLabels[label.LabelKey.ValueString()] = values
		}
	}

	return policy, nil
}

func (r *ProfilePolicyResource) mapModelToResource(ctx context.Context, policy *britive.ResourceManagerProfilePolicy, state *ProfilePolicyResourceModel, diags *diag.Diagnostics) {
	state.ProfileID = types.StringValue(policy.ProfileID)
	state.PolicyName = types.StringValue(policy.Name)
	state.Description = types.StringValue(policy.Description)
	state.Consumer = types.StringValue(policy.Consumer)
	state.AccessType = types.StringValue(policy.AccessType)
	state.IsActive = types.BoolValue(policy.IsActive)
	state.IsDraft = types.BoolValue(policy.IsDraft)
	state.IsReadOnly = types.BoolValue(policy.IsReadOnly)

	// Handle condition with JSON normalization
	normalizedCondition := ""
	if policy.Condition != "" {
		var condMap interface{}
		if err := json.Unmarshal([]byte(policy.Condition), &condMap); err == nil {
			apiCon, err := json.Marshal(condMap)
			if err == nil {
				normalizedCondition = string(apiCon)
			}
		}
	}

	if !state.Condition.IsNull() && britive.ConditionEqual(normalizedCondition, state.Condition.ValueString()) {
		// Keep user's format
	} else {
		state.Condition = types.StringValue(normalizedCondition)
	}

	// Handle members with JSON marshaling
	if policy.Members != nil {
		mem, err := json.Marshal(policy.Members)
		if err == nil {
			if !state.Members.IsNull() && britive.MembersEqual(string(mem), state.Members.ValueString()) {
				// Keep user's format
			} else {
				state.Members = types.StringValue(string(mem))
			}
		}
	}

	// Map resource labels directly as a slice (SetNestedBlock)
	var resourceLabelsList []ProfilePolicyLabelModel
	for labelKey, values := range policy.ResourceLabels {
		valuesSet, diagsSet := types.SetValueFrom(ctx, types.StringType, values)
		if diagsSet.HasError() {
			diags.Append(diagsSet...)
			continue
		}
		resourceLabelsList = append(resourceLabelsList, ProfilePolicyLabelModel{
			LabelKey: types.StringValue(labelKey),
			Values:   valuesSet,
		})
	}
	state.ResourceLabels = resourceLabelsList
}

func (r *ProfilePolicyResource) parseUniqueID(id string) (string, string) {
	idArr := strings.Split(id, "/")
	length := len(idArr)
	return idArr[length-3], idArr[length-1]
}

// ProfilePolicyLabelModel - separate type to avoid conflicts with ResourceLabelModel
type ProfilePolicyLabelModel struct {
	LabelKey types.String `tfsdk:"label_key"`
	Values   types.Set    `tfsdk:"values"`
}
