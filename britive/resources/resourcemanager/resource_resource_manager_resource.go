package resourcemanager

import (
	"context"
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
	_ resource.Resource                = &ResourceResourceManagerResource{}
	_ resource.ResourceWithConfigure   = &ResourceResourceManagerResource{}
	_ resource.ResourceWithImportState = &ResourceResourceManagerResource{}
	_ resource.ResourceWithModifyPlan  = &ResourceResourceManagerResource{}
)

type ResourceResourceManagerResource struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResourceHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResourceHelper struct{}

func NewResourceResourceManagerResource() resource.Resource {
	return &ResourceResourceManagerResource{}
}

func NewResourceResourceManagerResourceHelper() *ResourceResourceManagerResourceHelper {
	return &ResourceResourceManagerResourceHelper{}
}

func (r *ResourceResourceManagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_resource"
}

func (r *ResourceResourceManagerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Resource Manager Resource resource")

	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*britive_client.Client)
	if r.client == nil {
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
	r.helper = NewResourceResourceManagerResourceHelper()
}

func (r *ResourceResourceManagerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Resource resource",
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
				Description: "The name of resource",
				Validators: []validator.String{
					validate.StringFunc(
						"name",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive resource",
				Validators: []validator.String{
					validate.StringFunc(
						"description",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"resource_type": schema.StringAttribute{
				Required:    true,
				Description: "The resource type name associated with the server access resource",
				Validators: []validator.String{
					validate.StringFunc(
						"resource_type",
						validate.StringIsNotWhiteSpace(),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of resource type",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parameter_values": schema.MapAttribute{
				Optional:    true,
				Description: "The parameter values for the fields of the resource type",
				ElementType: types.StringType,
			},
			"resource_labels": schema.MapAttribute{
				Optional:    true,
				Description: "The resource labels associated with the server access resource",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *ResourceResourceManagerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() {
		return
	}

	var plan, state *britive_client.ResourceManagerResourcePlan

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan == nil {
		return
	}

	if !state.ResourceType.IsNull() &&
		!plan.ResourceType.Equal(state.ResourceType) {

		resp.Diagnostics.AddError(
			"Invalid ResourceType",
			fmt.Sprintf(
				"field 'resource_type' is immutable and cannot be changed (from '%s' to '%s')",
				state.ResourceType.ValueString(),
				plan.ResourceType.ValueString(),
			),
		)
	}
}

func (r *ResourceResourceManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Called create for britive_resource_manager_resource")

	var plan britive_client.ResourceManagerResourcePlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during resource creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	serverAccessResource := britive_client.ServerAccessResource{}

	err := r.helper.mapResourceToModel(ctx, plan, &serverAccessResource, false)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Adding new server access resource: %#v", serverAccessResource))

	sa, err := r.client.AddServerAccessResource(ctx, serverAccessResource)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new server access resource: %#v", sa))
	plan.ID = types.StringValue(sa.ResourceID)

	planPtr, err := r.helper.getAndMapModelToPlan(ctx, plan, *r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource model to plan", map[string]interface{}{
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
		"resource": planPtr,
	})
}

func (r *ResourceResourceManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_resource")

	if r.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResourcePlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get resource state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := r.helper.getAndMapModelToPlan(ctx, state, *r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map resource model to plan failed in Read", map[string]interface{}{
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

	tflog.Info(ctx, "Read completed for britive_resource_manager_resource")
}

func (r *ResourceResourceManagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_resource_manager_resource")

	var plan, state britive_client.ResourceManagerResourcePlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	serverAccessResourceID := state.ID.ValueString()

	var hasChanges bool
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) || !plan.ResourceType.Equal(state.ResourceType) || !plan.ParameterValues.Equal(state.ParameterValues) || !plan.ResourceLabels.Equal(state.ResourceLabels) {
		hasChanges = true
		serverAccessResource := britive_client.ServerAccessResource{}

		err := r.helper.mapResourceToModel(ctx, plan, &serverAccessResource, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, erro:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Updating resource: %#v", serverAccessResource))

		ursa, err := r.client.UpdateServerAccessResource(ctx, serverAccessResource, serverAccessResourceID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource, erro:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated server access resource: %#v", ursa))
		plan.ID = types.StringValue(serverAccessResourceID)
	}
	if hasChanges {
		planPtr, err := r.helper.getAndMapModelToPlan(ctx, plan, *r.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get resource",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map resource model to plan", map[string]interface{}{
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

func (r *ResourceResourceManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_resource")

	var state britive_client.ResourceManagerResourcePlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	serverAccessResourceID := state.ID.ValueString()
	tflog.Info(ctx, fmt.Sprintf("Deleting server access resource: %s", serverAccessResourceID))
	err := r.client.DeleteServerAccessResource(ctx, serverAccessResourceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete resource, error:%#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Resource %s deleted", serverAccessResourceID))
	resp.State.RemoveResource(ctx)
}

func (r *ResourceResourceManagerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := r.importHelper.ParseImportID([]string{"resources/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import resource", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, error:%#v", err))
		return
	}
	serverAccessResourceName := importData.Fields["name"]
	if strings.TrimSpace(serverAccessResourceName) == "" {
		resp.Diagnostics.AddError("Failed to import resource", "Invalid name")
		tflog.Error(ctx, "Failed to import resource, Invalid name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing server access resource : %s", serverAccessResourceName))

	serverAccessResource, err := r.client.GetServerAccessResourceByName(ctx, serverAccessResourceName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import resource", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import resource, error:%#v", err))
		return
	}

	plan := britive_client.ResourceManagerResourcePlan{
		ID:              types.StringValue(serverAccessResource.ResourceID),
		ParameterValues: types.MapNull(types.StringType),
		ResourceLabels:  types.MapNull(types.StringType),
	}

	planPtr, err := r.helper.getAndMapModelToPlan(ctx, plan, *r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported resource: %#v", planPtr))
}

func (rh *ResourceResourceManagerResourceHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourcePlan, c britive_client.Client) (*britive_client.ResourceManagerResourcePlan, error) {
	serverAccessResourceID := plan.ID.ValueString()

	tflog.Info(ctx, fmt.Sprintf("Reading server access resource %s", serverAccessResourceID))

	serverAccessResource, err := c.GetServerAccessResource(ctx, serverAccessResourceID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("role %s", serverAccessResourceID)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received server access resource %#v", serverAccessResource))

	plan.Name = types.StringValue(serverAccessResource.Name)
	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && serverAccessResource.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(serverAccessResource.Description)
	}
	plan.ResourceType = types.StringValue(serverAccessResource.ResourceType.Name)
	plan.ResourceTypeID = types.StringValue(serverAccessResource.ResourceType.ResourceTypeID)

	if (plan.ParameterValues.IsNull() || plan.ParameterValues.IsUnknown()) && serverAccessResource.ResourceTypeParameterValues == nil {
		plan.ParameterValues = types.MapNull(types.StringType)
	} else {
		paramValues, err := rh.mapMapToTypeMap(ctx, serverAccessResource.ResourceTypeParameterValues)
		if err != nil {
			return nil, err
		}
		plan.ParameterValues = paramValues
	}

	delete(serverAccessResource.ResourceLabels, "Resource-Type")
	resourceLabelMap := make(map[string]string)
	for key, value := range serverAccessResource.ResourceLabels {
		resourceLabelMap[key] = strings.Join(value, ",")
	}
	delete(resourceLabelMap, "Resource-Type")

	if (plan.ResourceLabels.IsNull() || plan.ResourceLabels.IsUnknown()) && serverAccessResource.ResourceLabels == nil {
		plan.ResourceLabels = types.MapNull(types.StringType)
	} else {
		var convertedMap types.Map
		newResLabelsMap, err := rh.mapTypeMapToMap(ctx, plan.ResourceLabels)
		if err != nil {
			return nil, err
		}
		delete(newResLabelsMap, "Resource-Type")
		if britive_client.ResourceLabelsMapEqual(resourceLabelMap, newResLabelsMap) {
			convertedMap, err = rh.mapMapToTypeMap(ctx, newResLabelsMap)
			if err != nil {
				return nil, err
			}
		} else {
			convertedMap, err = rh.mapMapToTypeMap(ctx, resourceLabelMap)
			if err != nil {
				return nil, err
			}
		}

		plan.ResourceLabels = convertedMap
	}

	return &plan, nil
}

func (rh *ResourceResourceManagerResourceHelper) mapResourceToModel(ctx context.Context, plan britive_client.ResourceManagerResourcePlan, serverAccessResource *britive_client.ServerAccessResource, isUpdate bool) error {
	serverAccessResource.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		serverAccessResource.Description = plan.Description.ValueString()
	}
	serverAccessResource.ResourceType.Name = plan.ResourceType.ValueString()
	if !plan.ParameterValues.IsNull() && !plan.ParameterValues.IsUnknown() {
		convertedMap, err := rh.mapTypeMapToMap(ctx, plan.ParameterValues)
		if err != nil {
			return err
		}
		serverAccessResource.ResourceTypeParameterValues = convertedMap
	}

	if !plan.ResourceLabels.IsNull() && !plan.ResourceLabels.IsUnknown() {
		convertedResourceLables, err := rh.mapTypeMapToMap(ctx, plan.ResourceLabels)
		if err != nil {
			return err
		}
		revertedSliceMap := make(map[string][]string)
		for key, value := range convertedResourceLables {
			revertedSliceMap[key] = strings.Split(value, ",")
		}
		revertedSliceMap["Resource-Type"] = []string{serverAccessResource.ResourceType.Name}
		serverAccessResource.ResourceLabels = revertedSliceMap
	} else {
		serverAccessResource.ResourceLabels = map[string][]string{}
	}
	if isUpdate {
		serverAccessResource.ResourceType.ResourceTypeID = plan.ResourceTypeID.ValueString()
	}

	return nil
}

func (rh *ResourceResourceManagerResourceHelper) mapTypeMapToMap(ctx context.Context, userMap types.Map) (map[string]string, error) {
	convertedMap := make(map[string]string)

	diags := userMap.ElementsAs(ctx, &convertedMap, true)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert Terraform types.Map to map[string]string")
	}

	return convertedMap, nil
}

func (rh *ResourceResourceManagerResourceHelper) mapMapToTypeMap(ctx context.Context, stateMap map[string]string) (types.Map, error) {
	userMap, diags := types.MapValueFrom(ctx, types.StringType, stateMap)
	if diags.HasError() {
		return types.Map{}, fmt.Errorf("Failed to convert map[string]string to types.Map")
	}

	return userMap, nil
}
