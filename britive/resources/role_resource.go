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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &RoleResource{}
	_ resource.ResourceWithConfigure   = &RoleResource{}
	_ resource.ResourceWithImportState = &RoleResource{}
)

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

type RoleResource struct {
	client *britive.Client
}

type RoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.String `tfsdk:"permissions"`
}

func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of Britive role.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive role.",
			},
			"permissions": schema.StringAttribute{
				Required:    true,
				Description: "Permissions of the role (JSON string).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := britive.Role{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Unmarshal JSON string
	if err := json.Unmarshal([]byte(plan.Permissions.ValueString()), &role.Permissions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Permissions JSON",
			fmt.Sprintf("Could not parse permissions JSON: %s", err.Error()),
		)
		return
	}

	created, err := r.client.AddRole(role)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Role",
			fmt.Sprintf("Could not create role: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateRoleID(created.RoleID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role",
			fmt.Sprintf("Could not read role after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := parseRoleID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Role ID",
			fmt.Sprintf("Could not parse role ID: %s", err.Error()),
		)
		return
	}

	// Get role by ID to get the name
	roleInfo, err := r.client.GetRole(roleID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role",
			fmt.Sprintf("Could not read role %s: %s", roleID, err.Error()),
		)
		return
	}

	// Get full role details by name
	role, err := r.client.GetRoleByName(roleInfo.Name)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role",
			fmt.Sprintf("Could not read role '%s': %s", roleInfo.Name, err.Error()),
		)
		return
	}

	state.Name = types.StringValue(role.Name)
	state.Description = types.StringValue(role.Description)

	// Handle permissions with comparison
	permissionsJSON, err := json.Marshal(role.Permissions)
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleResourceModel
	var state RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := parseRoleID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Role ID",
			fmt.Sprintf("Could not parse role ID: %s", err.Error()),
		)
		return
	}

	role := britive.Role{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Unmarshal JSON string
	if err := json.Unmarshal([]byte(plan.Permissions.ValueString()), &role.Permissions); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Permissions JSON",
			fmt.Sprintf("Could not parse permissions JSON: %s", err.Error()),
		)
		return
	}

	// Get old name for update
	oldName := state.Name.ValueString()

	_, err = r.client.UpdateRole(role, oldName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Role",
			fmt.Sprintf("Could not update role: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateRoleID(roleID))

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role",
			fmt.Sprintf("Could not read role after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := parseRoleID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Role ID",
			fmt.Sprintf("Could not parse role ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeleteRole(roleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Role",
			fmt.Sprintf("Could not delete role %s: %s", roleID, err.Error()),
		)
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. roles/{name}
	// 2. {name}
	idRegexes := []string{
		`^roles/(?P<name>[^/]+)$`,
		`^(?P<name>[^/]+)$`,
	}

	var roleName string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "name" && i < len(matches) {
					roleName = matches[i]
					break
				}
			}
			if roleName != "" {
				break
			}
		}
	}

	if roleName == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'roles/{name}' or '{name}'", req.ID),
		)
		return
	}

	if strings.TrimSpace(roleName) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Role name cannot be empty or whitespace.",
		)
		return
	}

	// Get role by name
	role, err := r.client.GetRoleByName(roleName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Role Not Found",
			fmt.Sprintf("Role '%s' not found.", roleName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Role",
			fmt.Sprintf("Could not import role '%s': %s", roleName, err.Error()),
		)
		return
	}

	// Marshal JSON field
	permissionsJSON, _ := json.Marshal(role.Permissions)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateRoleID(role.RoleID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), role.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), role.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permissions"), string(permissionsJSON))...)
}

// populateStateFromAPI fetches role data from API and populates the state model
func (r *RoleResource) populateStateFromAPI(ctx context.Context, state *RoleResourceModel) error {
	roleID, err := parseRoleID(state.ID.ValueString())
	if err != nil {
		return err
	}

	// Get role by ID to get the name
	roleInfo, err := r.client.GetRole(roleID)
	if err != nil {
		return err
	}

	// Get full role details by name
	role, err := r.client.GetRoleByName(roleInfo.Name)
	if err != nil {
		return err
	}

	state.Name = types.StringValue(role.Name)
	state.Description = types.StringValue(role.Description)

	// Handle permissions with comparison
	permissionsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return err
	}
	newPermissions := state.Permissions.ValueString()
	if britive.ArrayOfMapsEqual(string(permissionsJSON), newPermissions) {
		state.Permissions = types.StringValue(newPermissions)
	} else {
		state.Permissions = types.StringValue(string(permissionsJSON))
	}

	return nil
}

// Helper functions
func generateRoleID(roleID string) string {
	return fmt.Sprintf("roles/%s", roleID)
}

func parseRoleID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid role ID format: %s", id)
	}
	return parts[1], nil
}
