package resourcemanager

import (
	"context"
	"encoding/json"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.ResourceWithConfigure   = &ResourceResourceManagerResourcePolicy{}
	_ resource.Resource                = &ResourceResourceManagerResourcePolicy{}
	_ resource.ResourceWithImportState = &ResourceResourceManagerResourcePolicy{}
)

type ResourceResourceManagerResourcePolicy struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResourcePolicyHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResourcePolicyHelper struct{}

func NewResourceResourceManagerResourcePolicy() resource.Resource {
	return &ResourceResourceManagerResourcePolicy{}
}

func NewResourceResourceManagerResourcePolicyHelper() *ResourceResourceManagerResourcePolicyHelper {
	return &ResourceResourceManagerResourcePolicyHelper{}
}

func (rp *ResourceResourceManagerResourcePolicy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_resource_policy"
}

func (rp *ResourceResourceManagerResourcePolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Resource Manager Resource Policy resource")

	if req.ProviderData == nil {
		return
	}

	rp.client = req.ProviderData.(*britive_client.Client)
	if rp.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured")
	rp.helper = NewResourceResourceManagerResourcePolicyHelper()
}

func (rp *ResourceResourceManagerResourcePolicy) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Resource Policy resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_name": schema.StringAttribute{
				Required:    true,
				Description: "The policy associated with the profile",
				Validators: []validator.String{
					validate.StringFunc(
						"policy_name",
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
				Description: "Members of the policy",
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
				Description: "Condition of the policy",
				Validators: []validator.String{
					validate.StringFunc(
						"condition",
						validate.StringIsNotWhiteSpace(),
					),
				},
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
							Description: "List of values of resource label",
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (rp *ResourceResourceManagerResourcePolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Called create for britive_resource_manager_resource_policy")

	var plan britive_client.ResourceManagerResourcePolicyPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during resource policy creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	resourcePolicy := &britive_client.ResourceManagerResourcePolicy{}
	err := rp.helper.mapResourceToModel(ctx, plan, resourcePolicy)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resourcePolicy to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new resource manager resource policy: %#v", resourcePolicy))

	resourcePolicy, err = rp.client.CreateUpdateResourceManagerResourcePolicy(ctx, *resourcePolicy, "", false)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource policy, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new resource policy: %#v", resourcePolicy))
	plan.ID = types.StringValue(rp.helper.generateUniqueID(resourcePolicy.PolicyID))

	planPtr, err := rp.helper.getAndMapModelToPlan(ctx, plan, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource policy model to plan", map[string]interface{}{
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
		"resource_policy": planPtr,
	})
}

func (rp *ResourceResourceManagerResourcePolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_resource_policy")

	if rp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResourcePolicyPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get resource policy state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := rp.helper.getAndMapModelToPlan(ctx, state, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map resource policy model to plan failed in Read", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	diags = resp.State.Set(ctx, newPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state in Read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Read completed for britive_resource_manager_resource_policy")
}

func (rp *ResourceResourceManagerResourcePolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_resource_manager_resource_policy")

	var plan, state britive_client.ResourceManagerResourcePolicyPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	isUpdated := false
	if !plan.PolicyName.Equal(state.PolicyName) || !plan.Description.Equal(state.Description) || !plan.IsActive.Equal(state.IsActive) || !plan.IsDraft.Equal(state.IsDraft) || !plan.IsReadOnly.Equal(state.IsReadOnly) || !plan.AccessType.Equal(state.AccessType) || !plan.AccessLevel.Equal(state.AccessLevel) || !plan.Consumer.Equal(state.Consumer) || !plan.Members.Equal(state.Members) || !plan.Condition.Equal(state.Condition) || !plan.ResourceLabels.Equal(state.ResourceLabels) {
		isUpdated = true
		policyID := rp.helper.parseUniqueID(plan.ID.ValueString())

		resourcepolicy := &britive_client.ResourceManagerResourcePolicy{}

		err := rp.helper.mapResourceToModel(ctx, plan, resourcepolicy)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map resource policy to model, error:%#v", err))
			return
		}

		resourcepolicy.PolicyID = policyID

		old_name := state.PolicyName.ValueString()
		oldMem := state.Members.ValueString()
		oldCon := state.Condition.ValueString()
		upp, err := rp.client.CreateUpdateResourceManagerResourcePolicy(ctx, *resourcepolicy, old_name, true)
		if err != nil {
			plan.Members = types.StringValue(oldMem)
			plan.Condition = types.StringValue(oldCon)
			resp.Diagnostics.AddError("Failed to update resource policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource policy, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted Updated resource manager resource policy: %#v", upp))
	}
	if isUpdated {
		planPtr, err := rp.helper.getAndMapModelToPlan(ctx, plan, *rp.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get resource policy",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map resource policy model to plan", map[string]interface{}{
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
		tflog.Info(ctx, "Update completed and state set")
	}
}

func (rp *ResourceResourceManagerResourcePolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_resource_policy")

	var state britive_client.ResourceManagerResourcePolicyPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	policyID := rp.helper.parseUniqueID(state.ID.ValueString())

	tflog.Info(ctx, fmt.Sprintf("Deleting resource manager resource policy, %s", policyID))

	err := rp.client.DeleteResourceManagerResourcePolicy(ctx, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete resource policy, error:%#v", err))
		return
	}

	resp.State.RemoveResource(ctx)

	tflog.Info(ctx, fmt.Sprintf("Deleted resource manager resource policy, %s", policyID))
}

func (rp *ResourceResourceManagerResourcePolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rp.importHelper.ParseImportID([]string{"resource-manager/policies/(?P<policy_name>[^/]+)", "(?P<policy_name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import resource policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, error:%#v", err))
		return
	}

	policyName := importData.Fields["policy_name"]
	if strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError("Failed to import resource policy", "Invalid policy_name")
		tflog.Error(ctx, "Failed to import resource policy, Invalid policy_name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing resource manager resource policy: %s", policyName))

	resourcePolicy, err := rp.client.GetResourceManagerResourcePolicy(ctx, policyName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import resource policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import resource policy, error:%#v", err))
		return
	}

	plan := britive_client.ResourceManagerResourcePolicyPlan{
		ID: types.StringValue(rp.helper.generateUniqueID(resourcePolicy.PolicyID)),
		ResourceLabels: types.SetNull(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"label_key": types.StringType,
					"values": types.SetType{
						ElemType: types.StringType,
					},
				},
			},
		),
	}

	planPtr, err := rp.helper.getAndMapModelToPlan(ctx, plan, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource policy model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported resource policy: %#v", planPtr))
}

func (rph *ResourceResourceManagerResourcePolicyHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourcePolicyPlan, c britive_client.Client) (*britive_client.ResourceManagerResourcePolicyPlan, error) {
	policyID := rph.parseUniqueID(plan.ID.ValueString())

	policyName := plan.PolicyName.ValueString()

	tflog.Info(ctx, fmt.Sprintf("Reading resource manager resource policy: %s", policyID))

	resourcePolicy, err := c.GetResourceManagerResourcePolicy(ctx, policyName)
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received resource manager resource policy: %#v", resourcePolicy))

	plan.PolicyName = types.StringValue(resourcePolicy.Name)
	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && resourcePolicy.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(resourcePolicy.Description)
	}

	if (plan.Consumer.IsNull() || plan.Consumer.IsUnknown()) && resourcePolicy.Consumer == "" {
		plan.Consumer = types.StringNull()
	} else {
		plan.Consumer = types.StringValue(resourcePolicy.Consumer)
	}

	if (plan.AccessType.IsNull() || plan.AccessType.IsUnknown()) && resourcePolicy.AccessType == "" {
		plan.AccessType = types.StringNull()
	} else {
		plan.AccessType = types.StringValue(resourcePolicy.AccessType)
	}

	if (plan.AccessLevel.IsNull() || plan.AccessLevel.IsUnknown()) && resourcePolicy.AccessLevel == "" {
		plan.AccessLevel = types.StringNull()
	} else {
		plan.AccessLevel = types.StringValue(resourcePolicy.AccessLevel)
	}

	if (plan.IsActive.IsNull() || plan.IsActive.IsUnknown()) && resourcePolicy.IsActive == false {
		plan.IsActive = types.BoolNull()
	} else {
		plan.IsActive = types.BoolValue(resourcePolicy.IsActive)
	}

	if (plan.IsDraft.IsNull() || plan.IsDraft.IsUnknown()) && resourcePolicy.IsDraft == false {
		plan.IsDraft = types.BoolNull()
	} else {
		plan.IsDraft = types.BoolValue(resourcePolicy.IsDraft)
	}

	if (plan.IsReadOnly.IsNull() || plan.IsReadOnly.IsUnknown()) && resourcePolicy.IsReadOnly == false {
		plan.IsReadOnly = types.BoolNull()
	} else {
		plan.IsReadOnly = types.BoolValue(resourcePolicy.IsReadOnly)
	}

	if (plan.Condition.IsNull() || plan.Condition.IsUnknown()) && resourcePolicy.Condition == "" {
		plan.Condition = types.StringNull()
	} else {
		normalizedCondition := ""
		if resourcePolicy.Condition != "" {
			var condMap interface{}
			if err := json.Unmarshal([]byte(resourcePolicy.Condition), &condMap); err != nil {
				return nil, err
			}
			apiCon, err := json.Marshal(condMap)
			if err != nil {
				return nil, err
			}
			normalizedCondition = string(apiCon)
		}

		newCon := plan.Condition.ValueString()
		if britive_client.ConditionEqual(normalizedCondition, newCon) {
			plan.Condition = types.StringValue(newCon)
		} else {
			plan.Condition = types.StringValue(normalizedCondition)
		}
	}

	if (plan.Members.IsNull() || plan.Members.IsUnknown()) && (resourcePolicy.Members == "" || resourcePolicy.Members == "{}") {
		plan.Members = types.StringNull()
	} else {
		mem, err := json.Marshal(resourcePolicy.Members)
		if err != nil {
			return nil, err
		}

		newMem := plan.Members.ValueString()
		if britive_client.MembersEqual(string(mem), newMem) {
			plan.Members = types.StringValue(newMem)
		} else {
			plan.Members = types.StringValue(string(mem))
		}
	}

	if (plan.ResourceLabels.IsNull() || plan.ResourceLabels.IsUnknown()) && len(resourcePolicy.ResourceLabels) == 0 {
		plan.ResourceLabels = types.SetNull(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"label_key": types.StringType,
					"values": types.SetType{
						ElemType: types.StringType,
					},
				},
			},
		)
	} else {
		var resourceLabelsList []britive_client.ResourceLabelsForPolicy
		for name, values := range resourcePolicy.ResourceLabels {
			resourceLabel := britive_client.ResourceLabelsForPolicy{
				LabelKey: types.StringValue(name),
			}
			resValues, err := rph.mapValuesListToSet(ctx, values)
			if err != nil {
				return nil, err
			}
			resourceLabel.Values = resValues
			resourceLabelsList = append(resourceLabelsList, resourceLabel)
		}

		plan.ResourceLabels, err = rph.mapResourceLabelsListToSet(ctx, resourceLabelsList)
		if err != nil {
			return nil, err
		}
	}

	return &plan, err
}

func (rph *ResourceResourceManagerResourcePolicyHelper) mapResourceToModel(ctx context.Context, plan britive_client.ResourceManagerResourcePolicyPlan, resourcePolicy *britive_client.ResourceManagerResourcePolicy) error {
	resourcePolicy.Name = plan.PolicyName.ValueString()
	resourcePolicy.AccessType = plan.AccessType.ValueString()
	if !plan.AccessLevel.IsNull() && !plan.AccessLevel.IsUnknown() {
		resourcePolicy.AccessLevel = plan.AccessLevel.ValueString()
	}
	resourcePolicy.Description = plan.Description.ValueString()
	resourcePolicy.IsActive = plan.IsActive.ValueBool()
	resourcePolicy.IsDraft = plan.IsDraft.ValueBool()
	resourcePolicy.IsReadOnly = plan.IsReadOnly.ValueBool()
	resourcePolicy.Consumer = plan.Consumer.ValueString()

	err := json.Unmarshal([]byte(plan.Members.ValueString()), &resourcePolicy.Members)
	if err != nil {
		return err
	}
	resourcePolicy.Condition = plan.Condition.ValueString()

	if !plan.ResourceLabels.IsNull() && !plan.ResourceLabels.IsUnknown() {
		resourceLabels, err := rph.mapResourceLablesToList(ctx, plan.ResourceLabels)
		if err != nil {
			return err
		}
		resourceLabelsMap := make(map[string][]string)
		for _, label := range resourceLabels {
			labelName := label.LabelKey.ValueString()
			labelValues, err := rph.mapValuesToList(ctx, label.Values)
			if err != nil {
				return err
			}
			resourceLabelsMap[labelName] = labelValues
		}
		resourcePolicy.ResourceLabels = resourceLabelsMap
	}

	return nil
}

func (rph *ResourceResourceManagerResourcePolicyHelper) generateUniqueID(policyID string) string {
	return fmt.Sprintf("resource-manager/policies/%s", policyID)
}

func (rph *ResourceResourceManagerResourcePolicyHelper) parseUniqueID(id string) string {
	idArr := strings.Split(id, "/")
	return idArr[len(idArr)-1]
}

func (rph *ResourceResourceManagerResourcePolicyHelper) mapResourceLablesToList(ctx context.Context, set types.Set) ([]britive_client.ResourceLabelsForPolicy, error) {
	var resourceLabels []britive_client.ResourceLabelsForPolicy
	diags := set.ElementsAs(
		ctx,
		&resourceLabels,
		true,
	)
	if diags.HasError() {
		return nil, fmt.Errorf("Failed to map resource_labels as list")
	}

	return resourceLabels, nil
}

func (rph *ResourceResourceManagerResourcePolicyHelper) mapResourceLabelsListToSet(ctx context.Context, list []britive_client.ResourceLabelsForPolicy) (types.Set, error) {
	set, diags := types.SetValueFrom(
		ctx,
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"label_key": types.StringType,
				"values": types.SetType{
					ElemType: types.StringType,
				},
			},
		},
		list,
	)

	if diags.HasError() {
		return types.Set{}, fmt.Errorf("Failed to map resource_label list as Set")
	}

	return set, nil
}

func (rph *ResourceResourceManagerResourcePolicyHelper) mapValuesToList(ctx context.Context, set types.Set) ([]string, error) {
	var list []string
	diags := set.ElementsAs(ctx, &list, true)
	if diags.HasError() {
		return nil, fmt.Errorf("Failed to map values as list")
	}
	return list, nil
}

func (rph *ResourceResourceManagerResourcePolicyHelper) mapValuesListToSet(ctx context.Context, list []string) (types.Set, error) {
	set, diags := types.SetValueFrom(ctx, types.StringType, list)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("Failed to map values list as set")
	}

	return set, nil
}
