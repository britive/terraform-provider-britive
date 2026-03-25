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
	_ resource.Resource                = &PolicyResource{}
	_ resource.ResourceWithConfigure   = &PolicyResource{}
	_ resource.ResourceWithImportState = &PolicyResource{}
)

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

type PolicyResource struct {
	client *britive.Client
}

type PolicyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	IsDraft     types.Bool   `tfsdk:"is_draft"`
	IsReadOnly  types.Bool   `tfsdk:"is_read_only"`
	AccessType  types.String `tfsdk:"access_type"`
	Members     types.String `tfsdk:"members"`
	Condition   types.String `tfsdk:"condition"`
	Permissions types.String `tfsdk:"permissions"`
	Roles       types.String `tfsdk:"roles"`
}

func (r *PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the policy.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The description of the policy.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Is the policy active.",
			},
			"is_draft": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Is the policy a draft.",
			},
			"is_read_only": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Is the policy read only.",
			},
			"access_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Allow"),
				Description: "Type of access for the policy.",
			},
			"members": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("{}"),
				Description: "Members of the policy (JSON string).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"condition": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Condition of the policy.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"permissions": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("[]"),
				Description: "Permissions of the policy (JSON string).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"roles": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("[]"),
				Description: "Roles of the policy (JSON string).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *PolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := britive.Policy{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		AccessType:  plan.AccessType.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		IsDraft:     plan.IsDraft.ValueBool(),
		IsReadOnly:  plan.IsReadOnly.ValueBool(),
		Condition:   plan.Condition.ValueString(),
	}

	// Unmarshal JSON strings
	if err := json.Unmarshal([]byte(plan.Members.ValueString()), &policy.Members); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Members JSON",
			fmt.Sprintf("Could not parse members JSON: %s", err.Error()),
		)
		return
	}

	if err := json.Unmarshal([]byte(plan.Permissions.ValueString()), &policy.Permissions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Permissions JSON",
			fmt.Sprintf("Could not parse permissions JSON: %s", err.Error()),
		)
		return
	}

	if err := json.Unmarshal([]byte(plan.Roles.ValueString()), &policy.Roles); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Roles JSON",
			fmt.Sprintf("Could not parse roles JSON: %s", err.Error()),
		)
		return
	}

	created, err := r.client.CreatePolicy(policy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Policy",
			fmt.Sprintf("Could not create policy: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generatePolicyID(created.PolicyID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy",
			fmt.Sprintf("Could not read policy after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := parsePolicyID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Policy ID",
			fmt.Sprintf("Could not parse policy ID: %s", err.Error()),
		)
		return
	}

	// Get policy by ID to get the name
	policyInfo, err := r.client.GetPolicy(policyID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy",
			fmt.Sprintf("Could not read policy %s: %s", policyID, err.Error()),
		)
		return
	}

	// Get full policy details by name
	policy, err := r.client.GetPolicyByName(policyInfo.Name)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy",
			fmt.Sprintf("Could not read policy '%s': %s", policyInfo.Name, err.Error()),
		)
		return
	}

	state.Name = types.StringValue(policy.Name)
	state.Description = types.StringValue(policy.Description)
	state.AccessType = types.StringValue(policy.AccessType)
	state.IsActive = types.BoolValue(policy.IsActive)
	state.IsDraft = types.BoolValue(policy.IsDraft)
	state.IsReadOnly = types.BoolValue(policy.IsReadOnly)

	// Handle condition with comparison
	newCondition := state.Condition.ValueString()
	if britive.ConditionEqual(policy.Condition, newCondition) {
		state.Condition = types.StringValue(newCondition)
	} else {
		state.Condition = types.StringValue(policy.Condition)
	}

	// Handle members with comparison
	membersJSON, err := json.Marshal(policy.Members)
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

	// Handle permissions with comparison
	permissionsJSON, err := json.Marshal(policy.Permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Marshaling Permissions",
			fmt.Sprintf("Could not marshal permissions: %s", err.Error()),
		)
		return
	}
	newPermissions := state.Permissions.ValueString()
	if britive.ArrayOfMapsEqual(string(permissionsJSON), newPermissions) {
		state.Permissions = types.StringValue(newPermissions)
	} else {
		state.Permissions = types.StringValue(string(permissionsJSON))
	}

	// Handle roles with comparison
	rolesJSON, err := json.Marshal(policy.Roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Marshaling Roles",
			fmt.Sprintf("Could not marshal roles: %s", err.Error()),
		)
		return
	}
	newRoles := state.Roles.ValueString()
	if britive.ArrayOfMapsEqual(string(rolesJSON), newRoles) {
		state.Roles = types.StringValue(newRoles)
	} else {
		state.Roles = types.StringValue(string(rolesJSON))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PolicyResourceModel
	var state PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := parsePolicyID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Policy ID",
			fmt.Sprintf("Could not parse policy ID: %s", err.Error()),
		)
		return
	}

	policy := britive.Policy{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		AccessType:  plan.AccessType.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		IsDraft:     plan.IsDraft.ValueBool(),
		IsReadOnly:  plan.IsReadOnly.ValueBool(),
		Condition:   plan.Condition.ValueString(),
	}

	// Unmarshal JSON strings
	if err := json.Unmarshal([]byte(plan.Members.ValueString()), &policy.Members); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Members JSON",
			fmt.Sprintf("Could not parse members JSON: %s", err.Error()),
		)
		return
	}

	if err := json.Unmarshal([]byte(plan.Permissions.ValueString()), &policy.Permissions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Permissions JSON",
			fmt.Sprintf("Could not parse permissions JSON: %s", err.Error()),
		)
		return
	}

	if err := json.Unmarshal([]byte(plan.Roles.ValueString()), &policy.Roles); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Roles JSON",
			fmt.Sprintf("Could not parse roles JSON: %s", err.Error()),
		)
		return
	}

	// Get old name for update
	oldName := state.Name.ValueString()

	_, err = r.client.UpdatePolicy(policy, oldName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Policy",
			fmt.Sprintf("Could not update policy: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generatePolicyID(policyID))

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policy",
			fmt.Sprintf("Could not read policy after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := parsePolicyID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Policy ID",
			fmt.Sprintf("Could not parse policy ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeletePolicy(policyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Policy",
			fmt.Sprintf("Could not delete policy %s: %s", policyID, err.Error()),
		)
		return
	}
}

func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. policies/{name}
	// 2. {name}
	idRegexes := []string{
		`^policies/(?P<name>[^/]+)$`,
		`^(?P<name>[^/]+)$`,
	}

	var policyName string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "name" && i < len(matches) {
					policyName = matches[i]
					break
				}
			}
			if policyName != "" {
				break
			}
		}
	}

	if policyName == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'policies/{name}' or '{name}'", req.ID),
		)
		return
	}

	if strings.TrimSpace(policyName) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Policy name cannot be empty or whitespace.",
		)
		return
	}

	// Get policy by name
	policy, err := r.client.GetPolicyByName(policyName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Policy Not Found",
			fmt.Sprintf("Policy '%s' not found.", policyName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Policy",
			fmt.Sprintf("Could not import policy '%s': %s", policyName, err.Error()),
		)
		return
	}

	// Marshal JSON fields
	membersJSON, _ := json.Marshal(policy.Members)
	permissionsJSON, _ := json.Marshal(policy.Permissions)
	rolesJSON, _ := json.Marshal(policy.Roles)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generatePolicyID(policy.PolicyID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), policy.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), policy.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_active"), policy.IsActive)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_draft"), policy.IsDraft)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_read_only"), policy.IsReadOnly)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_type"), policy.AccessType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("members"), string(membersJSON))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("condition"), policy.Condition)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permissions"), string(permissionsJSON))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("roles"), string(rolesJSON))...)
}

// populateStateFromAPI fetches policy data from API and populates the state model
func (r *PolicyResource) populateStateFromAPI(ctx context.Context, state *PolicyResourceModel) error {
	policyID, err := parsePolicyID(state.ID.ValueString())
	if err != nil {
		return err
	}

	// Get policy by ID to get the name
	policyInfo, err := r.client.GetPolicy(policyID)
	if err != nil {
		return err
	}

	// Get full policy details by name
	policy, err := r.client.GetPolicyByName(policyInfo.Name)
	if err != nil {
		return err
	}

	state.Name = types.StringValue(policy.Name)
	state.Description = types.StringValue(policy.Description)
	state.AccessType = types.StringValue(policy.AccessType)
	state.IsActive = types.BoolValue(policy.IsActive)
	state.IsDraft = types.BoolValue(policy.IsDraft)
	state.IsReadOnly = types.BoolValue(policy.IsReadOnly)

	// Handle condition with comparison
	newCondition := state.Condition.ValueString()
	if britive.ConditionEqual(policy.Condition, newCondition) {
		state.Condition = types.StringValue(newCondition)
	} else {
		state.Condition = types.StringValue(policy.Condition)
	}

	// Handle members with comparison
	membersJSON, err := json.Marshal(policy.Members)
	if err != nil {
		return err
	}
	newMembers := state.Members.ValueString()
	if britive.MembersEqual(string(membersJSON), newMembers) {
		state.Members = types.StringValue(newMembers)
	} else {
		state.Members = types.StringValue(string(membersJSON))
	}

	// Handle permissions with comparison
	permissionsJSON, err := json.Marshal(policy.Permissions)
	if err != nil {
		return err
	}
	newPermissions := state.Permissions.ValueString()
	if britive.ArrayOfMapsEqual(string(permissionsJSON), newPermissions) {
		state.Permissions = types.StringValue(newPermissions)
	} else {
		state.Permissions = types.StringValue(string(permissionsJSON))
	}

	// Handle roles with comparison
	rolesJSON, err := json.Marshal(policy.Roles)
	if err != nil {
		return err
	}
	newRoles := state.Roles.ValueString()
	if britive.ArrayOfMapsEqual(string(rolesJSON), newRoles) {
		state.Roles = types.StringValue(newRoles)
	} else {
		state.Roles = types.StringValue(string(rolesJSON))
	}

	return nil
}

// Helper functions
func generatePolicyID(policyID string) string {
	return fmt.Sprintf("policies/%s", policyID)
}

func parsePolicyID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid policy ID format: %s", id)
	}
	return parts[1], nil
}
