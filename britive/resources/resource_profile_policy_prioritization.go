package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceProfilePolicyPrioritization{}
	_ resource.ResourceWithConfigure   = &ResourceProfilePolicyPrioritization{}
	_ resource.ResourceWithImportState = &ResourceProfilePolicyPrioritization{}
)

type ResourceProfilePolicyPrioritization struct {
	client       *britive_client.Client
	helper       *ResourceProfilePolicyPrioritizationHelper
	importHelper *imports.ImportHelper
}

type ResourceProfilePolicyPrioritizationHelper struct{}

func NewResourceProfilePolicyPrioritization() resource.Resource {
	return &ResourceProfilePolicyPrioritization{}
}

func NewResourceProfilePolicyPrioritizationHelper() *ResourceProfilePolicyPrioritizationHelper {
	return &ResourceProfilePolicyPrioritizationHelper{}
}

func (rppp *ResourceProfilePolicyPrioritization) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_profile_policy_prioritization"
}

func (rppp *ResourceProfilePolicyPrioritization) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Profile Policy Prioritization resource")

	if req.ProviderData == nil {
		return
	}

	rppp.client = req.ProviderData.(*britive_client.Client)
	if rppp.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Profile Policy Prioritization")
	rppp.helper = NewResourceProfilePolicyPrioritizationHelper()
}

func (rppp *ResourceProfilePolicyPrioritization) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for profile policy prioritization resource",
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
				Description: "Profile ID",
			},
			"policy_priority_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Enable policy ordering",
				Validators: []validator.Bool{
					validate.BoolFunc(
						validate.IsPolicyPriorityEnabled(),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"policy_priority": schema.SetNestedBlock{
				Description: "Policies with id and priority",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "policy name",
						},
						"priority": schema.Int64Attribute{
							Required:    true,
							Description: "priority number",
						},
					},
				},
			},
		},
	}
}

func (rppp *ResourceProfilePolicyPrioritization) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_profile_policy_prioritization")

	var plan britive_client.ProfilePolicyPrioritizationPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during profile_policy_prioritization creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	resourcePolicyPriority := &britive_client.ProfilePolicyPriority{}

	tflog.Info(ctx, "Mapping resource to policy priority model")
	resourcePolicyPriority, err := rppp.helper.mapResourceToModel(ctx, plan, resourcePolicyPriority, *rppp.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create policy_priority", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, error:%#v", err))
		return
	}

	profileSummary, err := rppp.client.GetProfileSummary(ctx, resourcePolicyPriority.ProfileID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch profile summary", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile summary, error:%#v", err))
		return
	}

	profileSummary.PolicyOrderingEnabled = resourcePolicyPriority.PolicyOrderingEnabled

	tflog.Info(ctx, "Enabling policy prioritization")
	profileSummary, err = rppp.client.EnableDisablePolicyPrioritization(ctx, *profileSummary)
	if err != nil {
		resp.Diagnostics.AddError("Failed to enable policy prioritization", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to enable policy prioritization, error:%#v", err))
		return
	}

	profileId := resourcePolicyPriority.ProfileID

	if resourcePolicyPriority.PolicyOrderingEnabled {
		tflog.Info(ctx, fmt.Sprintf("Prioritizing policies:%v", resourcePolicyPriority.PolicyOrder))
		resourcePolicyPriority, err = rppp.client.PrioritizePolicies(ctx, *resourcePolicyPriority)
		if err != nil {
			resp.Diagnostics.AddError("Failed to prioritize policies", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to prioritize policies, error:%#v", err))
			return
		}
	}

	id := rppp.helper.generateUniqueID(profileId)

	plan.ID = types.StringValue(id)

	planPtr, err := rppp.helper.getAndMapModelToPlan(ctx, plan, *rppp.client, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get policy_order",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map policy_prioritization model to plan", map[string]interface{}{
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
		"policy_order": planPtr,
	})
}

func (rppp *ResourceProfilePolicyPrioritization) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_profile_policy_prioritization")

	if rppp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ProfilePolicyPrioritizationPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get policy priority state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rppp.helper.getAndMapModelToPlan(ctx, state, *rppp.client, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get policy priorities",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map policy prioritization model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read policy priorities:  %#v", planPtr))
}

func (rppp *ResourceProfilePolicyPrioritization) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_profile_policy_prioritization")

	var plan, state britive_client.ProfilePolicyPrioritizationPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	var hasChanges bool
	if !plan.ProfileID.Equal(state.ProfileID) || !plan.PolicyPriorityEnabled.Equal(state.PolicyPriorityEnabled) || !plan.PolicyPriority.Equal(state.PolicyPriority) {
		hasChanges = true
		resourcePolicyPriority := &britive_client.ProfilePolicyPriority{}

		tflog.Info(ctx, "Mapping resource to policy priority model")
		resourcePolicyPriority, err := rppp.helper.mapResourceToModel(ctx, plan, resourcePolicyPriority, *rppp.client)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update policy prioritization", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map policy priority resource to model, error:%#v", err))
			return
		}

		profileSummary, err := rppp.client.GetProfileSummary(ctx, resourcePolicyPriority.ProfileID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update policy prioritization", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile summary, error:%#v", err))
			return
		}

		profileSummary.PolicyOrderingEnabled = resourcePolicyPriority.PolicyOrderingEnabled

		tflog.Info(ctx, "Enabling policy prioritization")
		profileSummary, err = rppp.client.EnableDisablePolicyPrioritization(ctx, *profileSummary)
		if err != nil {
			resp.Diagnostics.AddError("Failed to enable policy prioritization", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to enable policy prioritization, error:%#v", err))
			return
		}

		profileId := resourcePolicyPriority.ProfileID

		if resourcePolicyPriority.PolicyOrderingEnabled {
			tflog.Info(ctx, fmt.Sprintf("Prioritizing policies:%v", resourcePolicyPriority.PolicyOrder))
			resourcePolicyPriority, err = rppp.client.PrioritizePolicies(ctx, *resourcePolicyPriority)
			if err != nil {
				resp.Diagnostics.AddError("Failed to update policy prioritization", err.Error())
				tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile summary, error:%#v", err))
				return
			}
		}

		id := rppp.helper.generateUniqueID(profileId)
		plan.ID = types.StringValue(id)
	}
	if hasChanges {
		planPtr, err := rppp.helper.getAndMapModelToPlan(ctx, plan, *rppp.client, false)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map policy priority model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated policy priorities: %#v", planPtr))
	}
}

func (rppp *ResourceProfilePolicyPrioritization) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_profile_policy_prioritization")

	var state britive_client.ProfilePolicyPrioritizationPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileId := rppp.helper.parseUniqueID(state.ID.ValueString())

	profileSummary, err := rppp.client.GetProfileSummary(ctx, profileId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete policy prioritization", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to get profile summary, error:%#v", err))
		return
	}

	profileSummary.PolicyOrderingEnabled = false

	tflog.Info(ctx, fmt.Sprintf("Disabling policy prioritization: %s", state.ID.ValueString()))
	_, err = rppp.client.EnableDisablePolicyPrioritization(ctx, *profileSummary)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete policy priorities", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete policy priorities, err:%#v", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (rppp *ResourceProfilePolicyPrioritization) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rppp.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/policies/priority", "(?P<profile_id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import policy priorities", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, error:%#v", err))
		return
	}

	profileId := importData.Fields["profile_id"]

	plan := &britive_client.ProfilePolicyPrioritizationPlan{
		ID: types.StringValue(rppp.helper.generateUniqueID(profileId)),
	}

	planPtr, err := rppp.helper.getAndMapModelToPlan(ctx, *plan, *rppp.client, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import policy priorities",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import policy prioritization model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported policy priorities : %#v", planPtr))
}

func (rppph *ResourceProfilePolicyPrioritizationHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ProfilePolicyPrioritizationPlan, c britive_client.Client, imported bool) (*britive_client.ProfilePolicyPrioritizationPlan, error) {
	profileId := rppph.parseUniqueID(plan.ID.ValueString())

	tflog.Info(ctx, fmt.Sprintf("Getting profile policy, profileID:%s", profileId))
	policies, err := c.GetProfilePolicies(ctx, profileId)
	if err != nil {
		return nil, err
	}

	profile, err := c.GetProfile(ctx, profileId)
	if err != nil {
		return nil, err
	}

	plan.ProfileID = types.StringValue(profileId)
	plan.PolicyPriorityEnabled = types.BoolValue(profile.PolicyOrderingEnabled)

	order, err := rppph.mapSetToPolicyOrder(plan.PolicyPriority)
	if err != nil {
		return nil, err
	}
	var policyOrder []britive_client.PolicyPriorityPlan

	if len(order) == 0 && imported {
		for _, policy := range policies {
			pOrder := britive_client.PolicyPriorityPlan{
				ID:       types.StringValue(policy.PolicyID),
				Priority: types.Int64Value(int64(policy.Order)),
			}
			policyOrder = append(policyOrder, pOrder)
		}
		policyOrderSet, err := rppph.mapPolicyPriorityToSet(policyOrder)
		if err != nil {
			return nil, err
		}
		plan.PolicyPriority = policyOrderSet
		return &plan, nil
	}

	userOrder := make(map[string]string)
	for _, ord := range order {
		idArr := strings.Split(ord.ID.ValueString(), "/")
		pId := idArr[len(idArr)-1]
		userOrder[pId] = ord.ID.ValueString()
	}

	for _, policy := range policies {
		if _, ok := userOrder[policy.PolicyID]; ok {
			pOrder := britive_client.PolicyPriorityPlan{
				ID:       types.StringValue(userOrder[policy.PolicyID]),
				Priority: types.Int64Value(int64(policy.Order)),
			}
			policyOrder = append(policyOrder, pOrder)
		}

	}
	policyOrderSet, err := rppph.mapPolicyPriorityToSet(policyOrder)
	if err != nil {
		return nil, err
	}
	plan.PolicyPriority = policyOrderSet

	return &plan, nil
}

func (rppph *ResourceProfilePolicyPrioritizationHelper) mapResourceToModel(ctx context.Context, plan britive_client.ProfilePolicyPrioritizationPlan, resourcePolicyPriority *britive_client.ProfilePolicyPriority, c britive_client.Client) (*britive_client.ProfilePolicyPriority, error) {
	profileId := plan.ProfileID.ValueString()
	policyOrder, err := rppph.mapSetToPolicyOrder(plan.PolicyPriority)
	if err != nil {
		return nil, err
	}
	policyOrderingEnabled := plan.PolicyPriorityEnabled.ValueBool()

	userMapPolicyToOrder := make(map[string]int)
	userMapOrderToPolicy := make(map[int]string)
	profilePolicies, err := c.GetProfilePolicies(ctx, profileId)
	if err != nil {
		return nil, err
	}
	for _, policy := range policyOrder {
		policyIdArr := strings.Split(policy.ID.ValueString(), "/")
		policy.ID = types.StringValue(policyIdArr[len(policyIdArr)-1])
		if policy.Priority.ValueInt64() < 0 || int(policy.Priority.ValueInt64()) >= len(profilePolicies) {
			return nil, fmt.Errorf("invalid priority value: %d. The total number of policies is %d, so the priority must be between 0 and %d, inclusive.", policy.Priority.ValueInt64(), len(profilePolicies), len(profilePolicies)-1)
		}
		if _, ok := userMapPolicyToOrder[policy.ID.ValueString()]; ok {
			return nil, fmt.Errorf("duplicate policy detected: %s. Each policy ID must be unique.", policy.ID.ValueString())
		}
		if _, ok := userMapOrderToPolicy[int(policy.Priority.ValueInt64())]; ok {
			return nil, fmt.Errorf("duplicate priority detected: %d. Each priority value must be unique.", policy.Priority.ValueInt64())
		}
		userMapOrderToPolicy[int(policy.Priority.ValueInt64())] = policy.ID.ValueString()
		userMapPolicyToOrder[policy.ID.ValueString()] = int(policy.Priority.ValueInt64())
	}

	skipped := 0

	checkPolicy := make(map[string]int)

	for i := 0; i < len(profilePolicies); i++ {
		var tempPolicyOrder britive_client.PolicyOrder

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

	resourcePolicyPriority.ProfileID = profileId
	resourcePolicyPriority.Extendable = false
	resourcePolicyPriority.PolicyOrderingEnabled = policyOrderingEnabled

	return resourcePolicyPriority, nil
}

func (rppph *ResourceProfilePolicyPrioritizationHelper) parseUniqueID(id string) string {
	idArr := strings.Split(id, "/")
	profileId := idArr[len(idArr)-3]

	return profileId
}

func (rppph *ResourceProfilePolicyPrioritizationHelper) generateUniqueID(profileId string) string {
	return fmt.Sprintf("paps/%s/policies/priority", profileId)
}

func (rppph *ResourceProfilePolicyPrioritizationHelper) mapSetToPolicyOrder(policyOrderSet types.Set) ([]britive_client.PolicyPriorityPlan, error) {
	var result []britive_client.PolicyPriorityPlan
	objs := policyOrderSet.Elements()
	for _, e := range objs {
		obj, ok := e.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected Object, got %T", e)
		}
		var p britive_client.PolicyPriorityPlan
		diags := obj.As(context.Background(), &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert object to PolicyProperty resource: %v", diags)
		}
		result = append(result, p)
	}
	return result, nil
}

func (rppph *ResourceProfilePolicyPrioritizationHelper) mapPolicyPriorityToSet(policyPriority []britive_client.PolicyPriorityPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(policyPriority))

	for _, p := range policyPriority {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"id":       types.StringType,
				"priority": types.Int64Type,
			},
			map[string]attr.Value{
				"id":       p.ID,
				"priority": p.Priority,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for policy priority: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":       types.StringType,
				"priority": types.Int64Type,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create sensitive properties set: %v", diags)
	}

	return set, nil
}
