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
	_ resource.Resource                = &ResourcePolicy{}
	_ resource.ResourceWithConfigure   = &ResourcePolicy{}
	_ resource.ResourceWithImportState = &ResourcePolicy{}
)

type ResourcePolicy struct {
	client       *britive_client.Client
	helper       *ResourcePolicyHelper
	importHelper *imports.ImportHelper
}

type ResourcePolicyHelper struct{}

func NewResourcePolicy() resource.Resource {
	return &ResourcePolicy{}
}

func NewResourcePolicyHelper() *ResourcePolicyHelper {
	return &ResourcePolicyHelper{}
}

func (rp *ResourcePolicy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_policy"
}

func (rp *ResourcePolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Policy resource")

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

	tflog.Info(ctx, "Provider client configured for Resource Policy")
	rp.helper = NewResourcePolicyHelper()
}

func (rp *ResourcePolicy) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for policy resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The policy name",
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
				Description: "The description of policy",
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
			"access_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Allow"),
				Description: "Type of access for policy",
			},
			"members": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("{}"),
				Description: "Members of policy",
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
				Description: "Condition of policy",
				Validators: []validator.String{
					validate.StringFunc(
						"members",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"permissions": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("[]"),
				Description: "Permissions of the policy",
				Validators: []validator.String{
					validate.StringFunc(
						"permissions",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"roles": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("[]"),
				Description: "Roles of policy",
				Validators: []validator.String{
					validate.StringFunc(
						"roles",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
		},
	}
}

func (rp *ResourcePolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_policy")

	var plan britive_client.PolicyPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during policy creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	policy := britive_client.Policy{}

	err := rp.helper.mapResourceToModel(plan, &policy)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map policy resource to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new policy: %#v", policy))

	po, err := rp.client.CreatePolicy(ctx, policy)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create policy, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new policy: %#v", po))
	plan.ID = types.StringValue(rp.helper.generateUniqueID(po.PolicyID))

	planPtr, err := rp.helper.getAndMapModelToPlan(ctx, plan, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map policy model to plan", map[string]interface{}{
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
		"poicy": planPtr,
	})
}

func (rp *ResourcePolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_policy")

	if rp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.PolicyPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get policy state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rp.helper.getAndMapModelToPlan(ctx, state, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map policy model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read policy:  %#v", planPtr))
}

func (rp *ResourcePolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_policy")

	var plan, state britive_client.PolicyPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	policyID, err := rp.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse policy ID, error:%#v", err))
		return
	}

	var hasChanges bool
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) || !plan.IsActive.Equal(state.IsActive) || !plan.IsDraft.Equal(state.IsDraft) || !plan.IsReadOnly.Equal(state.IsReadOnly) || !plan.AccessType.Equal(state.AccessType) || !plan.Members.Equal(state.Members) || !plan.Condition.Equal(state.Condition) || !plan.Permissions.Equal(state.Permissions) || !plan.Roles.Equal(state.Roles) {
		hasChanges = true

		policy := britive_client.Policy{}

		err := rp.helper.mapResourceToModel(plan, &policy)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to to map policy resource to model, error:%#v", err))
			return
		}

		old_name := state.Name.ValueString()
		oldMem := state.Members.ValueString()
		oldCon := state.Condition.ValueString()
		oldPerm := state.Permissions.ValueString()
		oldRole := state.Roles.ValueString()
		up, err := rp.client.UpdatePolicy(ctx, policy, old_name)
		if err != nil {
			plan.Members = types.StringValue(oldMem)
			plan.Condition = types.StringValue(oldCon)
			plan.Permissions = types.StringValue(oldPerm)
			plan.Roles = types.StringValue(oldRole)
			resp.Diagnostics.AddError("Failed to update policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update policy, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted Updated Policy: %#v", up))
		plan.ID = types.StringValue(rp.helper.generateUniqueID(policyID))
	}
	if hasChanges {
		planPtr, err := rp.helper.getAndMapModelToPlan(ctx, plan, *rp.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map policy model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated policy: %#v", planPtr))
	}
}

func (rp *ResourcePolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_policy")

	var state britive_client.PolicyPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	policyID, err := rp.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse policyID, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting Policy: %s", policyID))
	err = rp.client.DeletePolicy(ctx, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete policy, error:%#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Deleted Policy: %s", policyID))
	resp.State.RemoveResource(ctx)
}

func (rp *ResourcePolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rp.importHelper.ParseImportID([]string{"policies/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to parse importID", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, error:%#v", err))
		return
	}

	policyName := importData.Fields["name"]
	if strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError("Failed to import policy", "Invalid name")
		tflog.Error(ctx, "Failed to import policy, Invalid name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing Policy: %s", policyName))

	policy, err := rp.client.GetPolicyByName(ctx, policyName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import policy", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import policy, error:%#v", err))
		return
	}

	plan := britive_client.PolicyPlan{
		ID: types.StringValue(rp.helper.generateUniqueID(policy.PolicyID)),
	}

	planPtr, err := rp.helper.getAndMapModelToPlan(ctx, plan, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import policy",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import policy model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported policy : %#v", planPtr))
}

func (rph *ResourcePolicyHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.PolicyPlan, c britive_client.Client) (*britive_client.PolicyPlan, error) {
	policyID, err := rph.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading Policy: %s", policyID))

	policyId, err := c.GetPolicy(ctx, policyID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("Policy %s ", policyID)
	}
	if err != nil {
		return nil, err
	}
	policy, err := c.GetPolicyByName(ctx, policyId.Name)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("Policy %s ", policyId.Name)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received Policy: %#v", policy))

	plan.Name = types.StringValue(policy.Name)
	plan.Description = types.StringValue(policy.Description)
	plan.AccessType = types.StringValue(policy.AccessType)
	plan.IsActive = types.BoolValue(policy.IsActive)
	plan.IsDraft = types.BoolValue(policy.IsDraft)
	plan.IsReadOnly = types.BoolValue(policy.IsReadOnly)

	newCon := plan.Condition.ValueString()
	if britive_client.ConditionEqual(policy.Condition, newCon) {
		plan.Condition = types.StringValue(newCon)
	} else {
		plan.Condition = types.StringValue(policy.Condition)
	}

	mem, err := json.Marshal(policy.Members)
	if err != nil {
		return nil, err
	}

	newMem := plan.Members.ValueString()
	if britive_client.MembersEqual(string(mem), newMem) {
		plan.Members = types.StringValue(newMem)
	} else {
		plan.Members = types.StringValue(string(mem))
	}

	perm, err := json.Marshal(policy.Permissions)
	if err != nil {
		return nil, err
	}

	newPerm := plan.Permissions.ValueString()
	if britive_client.ArrayOfMapsEqual(string(perm), newPerm) {
		plan.Permissions = types.StringValue(newPerm)
	} else {
		plan.Permissions = types.StringValue(string(perm))
	}

	role, err := json.Marshal(policy.Roles)
	if err != nil {
		return nil, err
	}

	newRole := plan.Roles.ValueString()
	if britive_client.ArrayOfMapsEqual(string(role), newRole) {
		plan.Roles = types.StringValue(newRole)
	} else {
		plan.Roles = types.StringValue(string(role))
	}

	return &plan, nil

}

func (rph *ResourcePolicyHelper) mapResourceToModel(plan britive_client.PolicyPlan, policy *britive_client.Policy) error {
	policy.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		policy.Description = plan.Description.ValueString()
	}
	if !plan.AccessType.IsNull() && !plan.AccessType.IsUnknown() {
		policy.AccessType = plan.AccessType.ValueString()
	}
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		policy.IsActive = plan.IsActive.ValueBool()
	}
	if !plan.IsDraft.IsNull() && !plan.IsDraft.IsUnknown() {
		policy.IsDraft = plan.IsDraft.ValueBool()
	}
	if !plan.IsReadOnly.IsNull() && !plan.IsReadOnly.IsUnknown() {
		policy.IsReadOnly = plan.IsReadOnly.ValueBool()
	}
	if !plan.Condition.IsNull() && !plan.Condition.IsUnknown() {
		policy.Condition = plan.Condition.ValueString()
	}
	if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
		if err := json.Unmarshal([]byte(plan.Members.ValueString()), &policy.Members); err != nil {
			return err
		}
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		if err := json.Unmarshal([]byte(plan.Permissions.ValueString()), &policy.Permissions); err != nil {
			return err
		}
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		if err := json.Unmarshal([]byte(plan.Roles.ValueString()), &policy.Roles); err != nil {
			return err
		}
	}

	return nil
}

func (rph *ResourcePolicyHelper) generateUniqueID(policyID string) string {
	return fmt.Sprintf("policies/%s", policyID)
}

func (rph *ResourcePolicyHelper) parseUniqueID(ID string) (policyID string, err error) {
	policyParts := strings.Split(ID, "/")
	if len(policyParts) < 2 {
		err = errs.NewInvalidResourceIDError("Policy", ID)
		return
	}
	policyID = policyParts[1]
	return
}
