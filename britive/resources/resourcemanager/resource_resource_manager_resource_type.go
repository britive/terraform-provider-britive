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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceResourceManagerResourceType{}
	_ resource.ResourceWithConfigure   = &ResourceResourceManagerResourceType{}
	_ resource.ResourceWithImportState = &ResourceResourceManagerResourceType{}
)

type ResourceResourceManagerResourceType struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResourceTypeHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResourceTypeHelper struct{}

func NewResourceResourceManagerResourceType() resource.Resource {
	return &ResourceResourceManagerResourceType{}
}

func NewResourceResourceManagerResourceTypeHelper() *ResourceResourceManagerResourceTypeHelper {
	return &ResourceResourceManagerResourceTypeHelper{}
}

func (rt *ResourceResourceManagerResourceType) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_resource_type"
}

func (rt *ResourceResourceManagerResourceType) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Resource Manager Resource Type resource")

	if req.ProviderData == nil {
		return
	}

	rt.client = req.ProviderData.(*britive_client.Client)
	if rt.client == nil {
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
	rt.helper = NewResourceResourceManagerResourceTypeHelper()
}

func (rt *ResourceResourceManagerResourceType) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Resource Type resource",
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
				Description: "The name of resource type",
				Validators: []validator.String{
					validate.StringFunc(
						"name",
						validate.StringWithNoSpecialChar(),
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive resource type",
			},
			"icon": schema.StringAttribute{
				Optional:    true,
				Description: "Icon of resource type",
				Validators: []validator.String{
					validate.StringFunc(
						"icon",
						validate.ValidateSVGString(),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"parameters": schema.SetNestedBlock{
				Description: "Parameters/Fields of the resource type",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"param_name": schema.StringAttribute{
							Required:    true,
							Description: "Parameter name",
						},
						"param_type": schema.StringAttribute{
							Required:    true,
							Description: "Parameter Type",
							Validators: []validator.String{
								validate.StringFunc(
									"parameter_type",
									validate.ValidateResourceManagerResourceTypeParameter(),
								),
							},
						},
						"is_mandatory": schema.BoolAttribute{
							Required:    true,
							Description: "Is parameter mandatory",
						},
					},
				},
			},
		},
	}
}

func (rt *ResourceResourceManagerResourceType) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Called create for britive_resource_manager_resource_type")

	var plan britive_client.ResourceManagerResourceTypePlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during resource type creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	resourceType := britive_client.ResourceType{}

	err := rt.helper.mapResourceToModel(ctx, plan, &resourceType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource type", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resource type to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Adding new resource type: %#v", resourceType))

	rto, err := rt.client.CreateResourceType(ctx, resourceType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource type", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource type, error: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new resource type: %#v", rto))

	tflog.Info(ctx, fmt.Sprintf("Adding icon to resource type: %#v", rto))
	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() {
		userSVG := plan.Icon.ValueString()
		err = rt.client.AddRemoveIcon(ctx, rto.ResourceTypeID, userSVG)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create resource type", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to create resource type, error:%#v", err))
			if err := rt.client.DeleteResourceType(ctx, rto.ResourceTypeID); err != nil {
				resp.Diagnostics.AddError("Failed to delete created resource type", err.Error())
				tflog.Error(ctx, fmt.Sprintf("Failed to delete created resource type, error:%#v", err))
			}
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Added icon to resource type: %#v", rto))
	}

	plan.ID = types.StringValue(rt.helper.generateUniqueID(rto.ResourceTypeID))

	planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, *rt.client, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource type",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource type model to plan", map[string]interface{}{
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
		"resource_type": planPtr,
	})
}

func (rt *ResourceResourceManagerResourceType) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_resource_type")

	if rt.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResourceTypePlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get resource type state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := rt.helper.getAndMapModelToPlan(ctx, state, *rt.client, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource type",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map resource type model to plan failed in Read", map[string]interface{}{
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

	tflog.Info(ctx, "Read completed for britive_resource_manager_resource_type")
}

func (rt *ResourceResourceManagerResourceType) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_resource_manager_resource_type")

	var plan, state britive_client.ResourceManagerResourceTypePlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	resourceTypeID, err := rt.helper.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource type", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse resource type ID, error:%#v", err))
		return
	}

	var hasChanges bool
	if !plan.Description.Equal(state.Description) || !plan.Parameters.Equal(state.Parameters) || !plan.Name.Equal(state.Name) {
		hasChanges = true
		resourceType := britive_client.ResourceType{}

		err := rt.helper.mapResourceToModel(ctx, plan, &resourceType)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource type", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map resource type to model, error:%#v", err))
			return
		}

		ur, err := rt.client.UpdateResourceType(ctx, resourceType, resourceTypeID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource type", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource type, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("updated resource type: %#v", ur))

		plan.ID = types.StringValue(rt.helper.generateUniqueID(resourceTypeID))
	}
	if !plan.Icon.Equal(state.Icon) {
		hasChanges = true
		tflog.Info(ctx, fmt.Sprintf("Updating icon to resource types: %#v", resourceTypeID))
		userSVG := plan.Icon.ValueString()
		err = rt.client.AddRemoveIcon(ctx, resourceTypeID, userSVG)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource type", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource type, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Added icon to resource type: %#v", resourceTypeID))
		plan.ID = types.StringValue(rt.helper.generateUniqueID(resourceTypeID))
	}
	if hasChanges {
		planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, *rt.client, false)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get resource type",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map resource type model to plan", map[string]interface{}{
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

func (rt *ResourceResourceManagerResourceType) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_resource_type")

	var state britive_client.ResourceManagerResourceTypePlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	resourceTypeID, err := rt.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource type", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse resource type ID, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting resource type: %s", resourceTypeID))
	err = rt.client.DeleteResourceType(ctx, resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource type", err.Error())
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Resource type %s deleted", resourceTypeID))
	resp.State.RemoveResource(ctx)
}

func (rt *ResourceResourceManagerResourceType) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rt.importHelper.ParseImportID([]string{"resource-manager/resource-types/(?P<id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import resource type", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, error%#v", err))
		return
	}

	resourceTypeID := importData.Fields["id"]
	if strings.TrimSpace(resourceTypeID) == "" {
		resp.Diagnostics.AddError("Failed to import resource type", "Invalid Resource type ID")
		tflog.Error(ctx, "Failed to import resource type, Invalid Resource type ID")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing resource type: %s", resourceTypeID))

	resourceType, err := rt.client.GetResourceType(ctx, resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import resource type", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import resource type, error:%#v", err))
		return
	}

	plan := britive_client.ResourceManagerResourceTypePlan{
		ID:         types.StringValue(rt.helper.generateUniqueID(resourceType.ResourceTypeID)),
		Parameters: types.SetNull(rt.helper.getParameterAttrType()),
	}

	planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, *rt.client, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource type model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported resource type: %#v", planPtr))
}

func (rth *ResourceResourceManagerResourceTypeHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourceTypePlan, c britive_client.Client, imported bool) (*britive_client.ResourceManagerResourceTypePlan, error) {
	resourceTypeID, err := rth.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading resource type %s", resourceTypeID))

	resourceType, err := c.GetResourceType(ctx, resourceTypeID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("resourceType %s", resourceTypeID)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received resource type %#v", resourceType))

	plan.Name = types.StringValue(resourceType.Name)
	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && resourceType.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(resourceType.Description)
	}

	if (plan.Parameters.IsNull() || plan.Parameters.IsUnknown()) && len(resourceType.Parameters) == 0 {
		plan.Parameters = types.SetNull(rth.getParameterAttrType())
	} else {
		paramMap := make(map[string]string)
		if !imported {
			stateParamameters, err := rth.mapParametersToList(ctx, plan)
			if err != nil {
				return nil, err
			}

			for i := 0; i < len(stateParamameters); i++ {
				parameter := stateParamameters[i]
				paramName := parameter.Parametername.ValueString()
				paramType := parameter.ParameterType.ValueString()
				paramMap[paramName] = paramType
			}
		}

		var parameterList []britive_client.ResourceManagerResourceTypeParameterPlan
		for _, parameter := range resourceType.Parameters {
			paramType := parameter.ParamType
			if !imported {
				paramType = paramMap[parameter.ParamName]
			}
			parameterList = append(parameterList, britive_client.ResourceManagerResourceTypeParameterPlan{
				Parametername: types.StringValue(parameter.ParamName),
				ParameterType: types.StringValue(paramType),
				IsMandatory:   types.BoolValue(parameter.IsMandatory),
			})
		}

		plan.Parameters, err = rth.mapParameterListToSet(ctx, parameterList)
		if err != nil {
			return nil, err
		}
	}

	return &plan, nil
}

func (rth *ResourceResourceManagerResourceTypeHelper) mapResourceToModel(ctx context.Context, plan britive_client.ResourceManagerResourceTypePlan, resourceType *britive_client.ResourceType) error {
	resourceType.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		resourceType.Description = plan.Description.ValueString()
	}
	if !plan.Parameters.IsNull() && !plan.Parameters.IsUnknown() {
		parameters, err := rth.mapParametersToList(ctx, plan)
		if err != nil {
			return err
		}

		for _, param := range parameters {
			paramName := param.Parametername.ValueString()
			paramType := param.ParameterType.ValueString()

			resourceType.Parameters = append(resourceType.Parameters,
				britive_client.Parameter{
					ParamName:   paramName,
					ParamType:   strings.ToLower(paramType),
					IsMandatory: param.IsMandatory.ValueBool(),
				})
		}
	}

	return nil
}

func (rth *ResourceResourceManagerResourceTypeHelper) generateUniqueID(resourceTypeID string) string {
	return fmt.Sprintf("resource-manager/resource-types/%s", resourceTypeID)
}

func (rth *ResourceResourceManagerResourceTypeHelper) parseUniqueID(ID string) (resourceTypeID string, err error) {
	resourceTypeParts := strings.Split(ID, "/")
	if len(resourceTypeParts) < 3 {
		err = errs.NewInvalidResourceIDError("resourceType", ID)
		return
	}

	resourceTypeID = resourceTypeParts[2]
	return
}

func (rth *ResourceResourceManagerResourceTypeHelper) mapParametersToList(ctx context.Context, plan britive_client.ResourceManagerResourceTypePlan) ([]britive_client.ResourceManagerResourceTypeParameterPlan, error) {
	set := plan.Parameters

	var paramList []britive_client.ResourceManagerResourceTypeParameterPlan
	diags := set.ElementsAs(ctx, &paramList, true)
	if diags.HasError() {
		return nil, fmt.Errorf("Failed to map parameters")
	}

	return paramList, nil
}

func (rth *ResourceResourceManagerResourceTypeHelper) mapParameterListToSet(ctx context.Context, paramList []britive_client.ResourceManagerResourceTypeParameterPlan) (types.Set, error) {
	set, diags := types.SetValueFrom(
		ctx,
		rth.getParameterAttrType(),
		paramList,
	)

	if diags.HasError() {
		return types.Set{}, fmt.Errorf("Failed to map parameters list")
	}

	return set, nil
}

func (rth *ResourceResourceManagerResourceTypeHelper) getParameterAttrType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"param_name":   types.StringType,
			"param_type":   types.StringType,
			"is_mandatory": types.BoolType,
		},
	}
}
