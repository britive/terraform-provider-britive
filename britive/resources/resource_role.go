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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceRole{}
	_ resource.ResourceWithConfigure   = &ResourceRole{}
	_ resource.ResourceWithImportState = &ResourceRole{}
)

type ResourceRole struct {
	client       *britive_client.Client
	helper       *ResourceRoleHelper
	importHelper *imports.ImportHelper
}

type ResourceRoleHelper struct{}

func NewResourceRole() resource.Resource {
	return &ResourceRole{}
}

func NewResourceRoleHelper() *ResourceRoleHelper {
	return &ResourceRoleHelper{}
}

func (rr *ResourceRole) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_role"
}

func (rr *ResourceRole) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Role resource")

	if req.ProviderData == nil {
		return
	}

	rr.client = req.ProviderData.(*britive_client.Client)
	if rr.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for ResourceTag")
	rr.helper = NewResourceRoleHelper()
}

func (rr *ResourceRole) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Role resource",
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
				Description: "The name of Britive Role",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive Role",
			},
			"permissions": schema.StringAttribute{
				Required:    true,
				Description: "permissions of role",
				Validators: []validator.String{
					validate.StringFunc(
						"Permissions",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
		},
	}
}

func (rr *ResourceRole) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_role")

	var plan britive_client.RolePlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during role creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})

		return
	}

	role := britive_client.Role{}

	err := rr.helper.mapResourceToModel(plan, &role)
	if err != nil {
		resp.Diagnostics.AddError("Faied to create role", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map role resource to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Adding new role: %#v", role))

	ro, err := rr.client.AddRole(ctx, role)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create role", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Dailed to create role, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new role: %#v", ro))
	plan.ID = types.StringValue(rr.helper.generateUniqueID(ro.RoleID))

	planPtr, err := rr.helper.getAndMapModelToPlan(ctx, plan, rr.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get role",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map role model to plan", map[string]interface{}{
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
		"role": planPtr,
	})
}

func (rr *ResourceRole) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_role")

	if rr.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.RolePlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get role state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rr.helper.getAndMapModelToPlan(ctx, state, rr.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get role",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map role model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read role:  %#v", planPtr))
}

func (rr *ResourceRole) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_role")

	var plan, state britive_client.RolePlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	roleID, err := rr.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update role", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse role ID, error:%#v", err))
		return
	}

	var hasChanges bool
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) || !plan.Permissions.Equal(state.Permissions) {
		hasChanges = true
		role := britive_client.Role{}

		err := rr.helper.mapResourceToModel(plan, &role)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update role", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map role resource to model, error:%#v", err))
			return
		}

		old_name := state.Name.ValueString()
		oldPerm := state.Permissions.ValueString()
		ur, err := rr.client.UpdateRole(ctx, role, old_name)
		if err != nil {
			plan.Permissions = types.StringValue(oldPerm)
			resp.Diagnostics.AddError("Failed to update role", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update role, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated role: %#v", ur))
		plan.ID = types.StringValue(rr.helper.generateUniqueID(roleID))
	}
	if hasChanges {
		planPtr, err := rr.helper.getAndMapModelToPlan(ctx, plan, rr.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map role model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated role: %#v", planPtr))
	}
}

func (rr *ResourceRole) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_role")

	var state britive_client.RolePlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	roleID, err := rr.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete role", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse role ID, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting role: %s", roleID))
	err = rr.client.DeleteRole(ctx, roleID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete role", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete role, error:%#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Role %s deleted", roleID))
	resp.State.RemoveResource(ctx)
}

func (rr *ResourceRole) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rr.importHelper.ParseImportID([]string{"roles/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to parse import ID", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse import ID, error:%#v", err))
		return
	}
	roleName := importData.Fields["name"]
	if strings.TrimSpace(roleName) == "" {
		resp.Diagnostics.AddError("Failed to import role", "Invalid name")
		tflog.Error(ctx, "Failed to import role, Invalid name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing role: %s", roleName))

	role, err := rr.client.GetRoleByName(ctx, roleName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import role", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import role, error:%#v", err))
		return
	}

	plan := britive_client.RolePlan{
		ID: types.StringValue(rr.helper.generateUniqueID(role.RoleID)),
	}

	planPtr, err := rr.helper.getAndMapModelToPlan(ctx, plan, rr.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map role model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported role: %#v", planPtr))
}

func (rrh *ResourceRoleHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.RolePlan, c *britive_client.Client) (*britive_client.RolePlan, error) {
	roleID, err := rrh.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading role %s", roleID))

	roleRes, err := c.GetRole(ctx, roleID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("role %s", roleID)
	}
	if err != nil {
		return nil, err
	}

	role, err := c.GetRoleByName(ctx, roleRes.Name)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("role %s", roleRes.Name)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received role %#v", role))

	plan.Name = types.StringValue(role.Name)
	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && role.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(role.Description)
	}

	perm, err := json.Marshal(role.Permissions)
	if err != nil {
		return nil, err
	}

	newPerm := plan.Permissions.ValueString()
	if britive_client.ArrayOfMapsEqual(string(perm), newPerm) {
		plan.Permissions = types.StringValue(newPerm)
	} else {
		plan.Permissions = types.StringValue(string(perm))
	}

	return &plan, nil
}

func (rrh *ResourceRoleHelper) mapResourceToModel(plan britive_client.RolePlan, role *britive_client.Role) error {
	role.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		role.Description = plan.Description.ValueString()
	}

	if err := json.Unmarshal([]byte(plan.Permissions.ValueString()), &role.Permissions); err != nil {
		return err
	}
	return nil
}

func (rrh *ResourceRoleHelper) generateUniqueID(roleID string) string {
	return fmt.Sprintf("roles/%s", roleID)
}

func (rrh *ResourceRoleHelper) parseUniqueID(ID string) (roleID string, err error) {
	roleParts := strings.Split(ID, "/")
	if len(roleParts) < 2 {
		err = errs.NewInvalidResourceIDError("role", ID)
		return
	}

	roleID = roleParts[1]
	return
}
