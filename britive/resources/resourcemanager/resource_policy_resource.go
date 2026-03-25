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

type ResourcePolicyResource struct {
	client *britive.Client
}

type ResourcePolicyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	PolicyName     types.String `tfsdk:"policy_name"`
	Description    types.String `tfsdk:"description"`
	IsActive       types.Bool   `tfsdk:"is_active"`
	IsDraft        types.Bool   `tfsdk:"is_draft"`
	IsReadOnly     types.Bool   `tfsdk:"is_read_only"`
	Consumer       types.String `tfsdk:"consumer"`
	AccessType     types.String `tfsdk:"access_type"`
	AccessLevel    types.String `tfsdk:"access_level"`
	Members        types.String `tfsdk:"members"`
	Condition      types.String `tfsdk:"condition"`
	ResourceLabels []ResourceLabelModel `tfsdk:"resource_labels"`
}

type ResourceLabelModel struct {
	LabelKey types.String `tfsdk:"label_key"`
	Values   types.Set    `tfsdk:"values"`
}

func NewResourcePolicyResource() resource.Resource {
	return &ResourcePolicyResource{}
}

func (r *ResourcePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_policy"
}

func (r *ResourcePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager resource policy",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				Default:     stringdefault.StaticString("resourcemanager"),
				Description: "The consumer service",
			},
			"access_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Allow"),
				Description: "Type of access for the policy",
			},
			"access_level": schema.StringAttribute{
				Optional:    true,
				Description: "Level of access for the policy",
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

func (r *ResourcePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourcePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourcePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourcePolicy, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}

	log.Printf("[INFO] Creating new resource manager resource policy: %#v", resourcePolicy)

	created, err := r.client.CreateUpdateResourceManagerResourcePolicy(resourcePolicy, "", false)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Manager Resource Policy", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new resource policy: %#v", created)

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/policies/%s", created.PolicyID))

	// Read back to get computed values
	readResp, err := r.client.GetResourceManagerResourcePolicy(created.Name)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Resource Policy", err.Error())
		return
	}

	r.mapModelToResource(ctx, readResp, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourcePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourcePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyName := state.PolicyName.ValueString()

	log.Printf("[INFO] Reading resource manager resource policy: %s", policyName)

	resourceManagerResourcePolicy, err := r.client.GetResourceManagerResourcePolicy(policyName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Resource Policy", err.Error())
		return
	}

	log.Printf("[INFO] Received resource manager resource policy: %#v", resourceManagerResourcePolicy)

	r.mapModelToResource(ctx, resourceManagerResourcePolicy, &state, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourcePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourcePolicyResourceModel
	var state ResourcePolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID := r.parseUniqueID(state.ID.ValueString())

	resourcePolicy, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}
	resourcePolicy.PolicyID = policyID

	oldPolicyName := state.PolicyName.ValueString()

	log.Printf("[INFO] Updating resource manager resource policy: %s", policyID)

	_, err = r.client.CreateUpdateResourceManagerResourcePolicy(resourcePolicy, oldPolicyName, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource Manager Resource Policy", err.Error())
		return
	}

	log.Printf("[INFO] Submitted Updated resource manager resource policy: %s", policyID)

	// Read back to get updated values
	readResp, err := r.client.GetResourceManagerResourcePolicy(plan.PolicyName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Resource Policy", err.Error())
		return
	}

	r.mapModelToResource(ctx, readResp, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourcePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourcePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID := r.parseUniqueID(state.ID.ValueString())

	log.Printf("[INFO] Deleting resource manager resource policy: %s", policyID)

	err := r.client.DeleteResourceManagerResourcePolicy(policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Manager Resource Policy", err.Error())
		return
	}

	log.Printf("[INFO] Deleted resource manager resource policy: %s", policyID)
}

func (r *ResourcePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var policyName string

	// Support two formats: "resource-manager/policies/{policy_name}" or "{policy_name}"
	if strings.HasPrefix(importID, "resource-manager/policies/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/policies/{policy_name}' or '{policy_name}', got: %s", importID))
			return
		}
		policyName = parts[2]
	} else {
		policyName = importID
	}

	if strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Policy name cannot be empty")
		return
	}

	log.Printf("[INFO] Importing resource manager resource policy: %s", policyName)

	resourcePolicy, err := r.client.GetResourceManagerResourcePolicy(policyName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Manager Resource Policy Not Found", fmt.Sprintf("Resource policy %s not found", policyName))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Manager Resource Policy", err.Error())
		return
	}

	var state ResourcePolicyResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/policies/%s", resourcePolicy.PolicyID))

	r.mapModelToResource(ctx, resourcePolicy, &state, &resp.Diagnostics)

	log.Printf("[INFO] Imported resource manager resource policy: %s", policyName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ResourcePolicyResource) mapResourceToModel(ctx context.Context, plan *ResourcePolicyResourceModel) (britive.ResourceManagerResourcePolicy, error) {
	policy := britive.ResourceManagerResourcePolicy{
		Name:        plan.PolicyName.ValueString(),
		Description: plan.Description.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		IsDraft:     plan.IsDraft.ValueBool(),
		IsReadOnly:  plan.IsReadOnly.ValueBool(),
		Consumer:    plan.Consumer.ValueString(),
		AccessType:  plan.AccessType.ValueString(),
		AccessLevel: plan.AccessLevel.ValueString(),
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

	// Parse resource labels
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

func (r *ResourcePolicyResource) mapModelToResource(ctx context.Context, policy *britive.ResourceManagerResourcePolicy, state *ResourcePolicyResourceModel, diags *diag.Diagnostics) {
	state.PolicyName = types.StringValue(policy.Name)
	state.Description = types.StringValue(policy.Description)
	state.Consumer = types.StringValue(policy.Consumer)
	state.AccessType = types.StringValue(policy.AccessType)
	state.AccessLevel = types.StringValue(policy.AccessLevel)
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

	// Use user's condition if it matches normalized API condition, otherwise use API's
	if !state.Condition.IsNull() && britive.ConditionEqual(normalizedCondition, state.Condition.ValueString()) {
		// Keep user's format
	} else {
		state.Condition = types.StringValue(normalizedCondition)
	}

	// Handle members with JSON marshaling
	if policy.Members != nil {
		mem, err := json.Marshal(policy.Members)
		if err == nil {
			// Use user's members if it matches normalized API members, otherwise use API's
			if !state.Members.IsNull() && britive.MembersEqual(string(mem), state.Members.ValueString()) {
				// Keep user's format
			} else {
				state.Members = types.StringValue(string(mem))
			}
		}
	}

	// Map resource labels
	var resourceLabelsList []ResourceLabelModel
	for labelKey, values := range policy.ResourceLabels {
		valuesSet, diagsSet := types.SetValueFrom(ctx, types.StringType, values)
		if diagsSet.HasError() {
			diags.Append(diagsSet...)
			continue
		}
		resourceLabelsList = append(resourceLabelsList, ResourceLabelModel{
			LabelKey: types.StringValue(labelKey),
			Values:   valuesSet,
		})
	}
	state.ResourceLabels = resourceLabelsList
}

func (r *ResourcePolicyResource) parseUniqueID(id string) string {
	idArr := strings.Split(id, "/")
	return idArr[len(idArr)-1]
}
