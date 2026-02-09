package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourcePermission{}
	_ resource.ResourceWithConfigure   = &ResourcePermission{}
	_ resource.ResourceWithImportState = &ResourcePermission{}
)

type ResourcePermission struct {
	client       *britive_client.Client
	helper       *ResourcePermissionHelper
	importHelper *imports.ImportHelper
}

type ResourcePermissionHelper struct{}

func NewResourcePermission() resource.Resource {
	return &ResourcePermission{}
}

func NewResourcePermissionHelper() *ResourcePermissionHelper {
	return &ResourcePermissionHelper{}
}

func (rp *ResourcePermission) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_permission"
}

func (rp *ResourcePermission) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Permission resource")

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

	tflog.Info(ctx, "Provider client configured for ResourceTag")
	rp.helper = NewResourcePermissionHelper()
}

func (rp *ResourcePermission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Permission resource",
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
				Description: "The name of Britive Permission",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive Permission",
			},
			"consumer": schema.StringAttribute{
				Required:    true,
				Description: "The consumer service",
			},
			"resources": schema.SetAttribute{
				Required:    true,
				Description: "Comma separated list of resources",
				ElementType: types.StringType,
			},
			"actions": schema.SetAttribute{
				Required:    true,
				Description: "Actions to be performed on the resource",
				ElementType: types.StringType,
			},
		},
	}
}

func (rp *ResourcePermission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_permission")

	var plan britive_client.PermissionPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during permission creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})

		return
	}

	permission := britive_client.Permission{}

	err := rp.helper.mapResourceToModel(ctx, plan, &permission)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create permission, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Adding new permission: %#v", permission))

	pm, err := rp.client.AddPermission(ctx, permission)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create permission, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new permission: %#v", pm))
	plan.ID = types.StringValue(rp.helper.generateUniqueID(pm.PermissionID))

	planPtr, err := rp.helper.getAndMapModelToResource(ctx, plan, rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map permission model to plan", map[string]interface{}{
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
		"permission": planPtr,
	})
}

func (rp *ResourcePermission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_permission")

	if rp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.PermissionPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get permission state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rp.helper.getAndMapModelToResource(ctx, state, rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map permission model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read permission: %#v", planPtr))
}

func (rp *ResourcePermission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_permission")

	var plan, state britive_client.PermissionPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	permissionID, err := rp.helper.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update permission", err.Error())
		tflog.Info(ctx, fmt.Sprintf("Failed to update permission, error:%#v", err))
		return
	}
	var hasChanges bool
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) || !plan.Consumer.Equal(state.Consumer) || !plan.Resources.Equal(state.Resources) || !plan.Actions.Equal(state.Actions) {
		hasChanges = true
		permission := britive_client.Permission{}

		err := rp.helper.mapResourceToModel(ctx, plan, &permission)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update permission", err.Error())
			tflog.Info(ctx, fmt.Sprintf("Failed to update permission, error:%#v", err))
			return
		}

		old_name := state.Name.ValueString()
		up, err := rp.client.UpdatePermission(ctx, permission, old_name)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update permission", err.Error())
			tflog.Info(ctx, fmt.Sprintf("Failed to update permission, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated permission: %#v", up))
		plan.ID = types.StringValue(rp.helper.generateUniqueID(permissionID))
	}
	if hasChanges {
		planPtr, err := rp.helper.getAndMapModelToResource(ctx, plan, rp.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map permission model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated permission: %#v", planPtr))
	}
}

func (rp *ResourcePermission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_permission")

	var state britive_client.PermissionPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	permissionID, err := rp.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete permission, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting permission: %s", permissionID))
	err = rp.client.DeletePermission(ctx, permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete permission, error:%#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Permission %s deleted", permissionID))
	resp.State.RemoveResource(ctx)
}

func (rp *ResourcePermission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rp.importHelper.ParseImportID([]string{"permissions/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import permission", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	permissionName := importData.Fields["name"]
	if strings.TrimSpace(permissionName) == "" {
		resp.Diagnostics.AddError("Failed to import permission", "Invalid name")
		tflog.Error(ctx, "Failed to import permission, Invalid name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing permission: %s", permissionName))

	permission, err := rp.client.GetPermissionByName(ctx, permissionName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import permission, error:%#v", err))
		return
	}

	plan := britive_client.PermissionPlan{
		ID: types.StringValue(rp.helper.generateUniqueID(permission.PermissionID)),
	}

	planPtr, err := rp.helper.getAndMapModelToResource(ctx, plan, rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import permission model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported permission : %#v", planPtr))
}

func (rph *ResourcePermissionHelper) getAndMapModelToResource(ctx context.Context, plan britive_client.PermissionPlan, c *britive_client.Client) (*britive_client.PermissionPlan, error) {
	permissionID, err := rph.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading permission %s", permissionID))

	permissionRes, err := c.GetPermission(ctx, permissionID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("permissi)on %s", permissionID)
	}
	if err != nil {
		return nil, err
	}
	permission, err := c.GetPermissionByName(ctx, permissionRes.Name)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("role %s", permissionRes.Name)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received permission %#v", permission))

	plan.Name = types.StringValue(permission.Name)
	plan.Description = types.StringValue(permission.Description)
	plan.Consumer = types.StringValue(permission.Consumer)
	resourcesSet, err := rph.mapResourcesListToSet(ctx, permission.Resources)
	if err != nil {
		return nil, err
	}
	plan.Resources = resourcesSet
	actionSet, err := rph.mapActionsListToSet(ctx, permission.Actions)
	if err != nil {
		return nil, err
	}
	plan.Actions = actionSet

	return &plan, nil
}

func (rph *ResourcePermissionHelper) mapResourceToModel(ctx context.Context, plan britive_client.PermissionPlan, permission *britive_client.Permission) error {
	permission.Name = plan.Name.ValueString()
	permission.Description = plan.Description.ValueString()
	permission.Consumer = plan.Consumer.ValueString()

	resList, err := rph.mapSetToList(ctx, plan.Resources)
	if err != nil {
		return err
	}
	permission.Resources = append(permission.Resources, rph.stringSliceToInterfaceSlice(resList)...)

	actList, err := rph.mapSetToList(ctx, plan.Actions)
	if err != nil {
		return err
	}
	permission.Actions = append(permission.Actions, rph.stringSliceToInterfaceSlice(actList)...)

	return nil
}

func (rph *ResourcePermissionHelper) stringSliceToInterfaceSlice(in []string) []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func (rph *ResourcePermissionHelper) generateUniqueID(permissionID string) string {
	return fmt.Sprintf("permissions/%s", permissionID)
}

func (rph *ResourcePermissionHelper) parseUniqueID(ID string) (permissionID string, err error) {
	permissionParts := strings.Split(ID, "/")
	if len(permissionParts) < 2 {
		err = errs.NewInvalidResourceIDError("permission", ID)
		return
	}

	permissionID = permissionParts[1]
	return
}

func (rph *ResourcePermissionHelper) mapSetToList(ctx context.Context, set types.Set) ([]string, error) {

	var list []string

	diags := set.ElementsAs(ctx, &list, false)
	if diags.HasError() {
		return nil, errors.New(diags.Errors()[0].Detail())
	}

	return list, nil
}

func (rph *ResourcePermissionHelper) mapResourcesListToSet(ctx context.Context, resources []interface{}) (types.Set, error) {

	set, diags := types.SetValueFrom(ctx, types.StringType, resources)
	if diags.HasError() {
		return types.SetNull(types.StringType), errors.New(diags.Errors()[0].Detail())
	}

	return set, nil
}

func (rph *ResourcePermissionHelper) mapActionsListToSet(ctx context.Context, actions []interface{}) (types.Set, error) {

	set, diags := types.SetValueFrom(ctx, types.StringType, actions)
	if diags.HasError() {
		return types.SetNull(types.StringType), errors.New(diags.Errors()[0].Detail())
	}

	return set, nil
}
