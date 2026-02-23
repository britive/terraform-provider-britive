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
		tflog.Error(ctx, "Failed to read plan during resource label creation", map[string]interface{}{
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

	planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, *rt.client)
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

	newPlan, err := rt.helper.getAndMapModelToPlan(ctx, state, *rt.client)
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
	// tflog.Info(ctx, "Update called for britive_resource_manager_resource_type")

	// var plan, state britive_client.ResourceManagerResourceTypePlan
	// resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	// resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// if resp.Diagnostics.HasError() {
	// 	tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
	// 		"diagnostics": resp.Diagnostics,
	// 	})
	// 	return
	// }

	// resourceTypeID, err := rt.helper.parseUniqueID(d.Id())
	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	// var hasChanges bool
	// if d.HasChange("description") || d.HasChange("parameters") || d.HasChange("name") {
	// 	hasChanges = true
	// 	resourceType := britive.ResourceType{}

	// 	err := rt.helper.mapResourceToModel(d, m, &resourceType, true)
	// 	if err != nil {
	// 		return diag.FromErr(err)
	// 	}

	// 	ur, err := c.UpdateResourceType(resourceType, resourceTypeID)
	// 	if err != nil {
	// 		return diag.FromErr(err)
	// 	}

	// 	log.Printf("[INFO] updated resource type: %#v", ur)

	// 	d.SetId(rt.helper.generateUniqueID(resourceTypeID))
	// }
	// if d.HasChange("icon") {
	// 	hasChanges = true
	// 	log.Printf("[INFO] Updating icon to resource type: %#v", resourceTypeID)
	// 	userSVG := d.Get("icon").(string)
	// 	err = c.AddRemoveIcon(resourceTypeID, userSVG)
	// 	if err != nil {
	// 		return diag.FromErr(err)
	// 	}

	// 	log.Printf("[INFO] Added icon to resource type: %#v", resourceTypeID)
	// }
	// if hasChanges {
	// 	return rt.resourceRead(ctx, d, m)
	// }
}

func (rth *ResourceResourceManagerResourceTypeHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourceTypePlan, c britive_client.Client) (*britive_client.ResourceManagerResourceTypePlan, error) {
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

	stateParamameters, err := rth.mapParametersToList(ctx, plan)
	if err != nil {
		return nil, err
	}
	paramMap := make(map[string]string)

	for i := 0; i < len(stateParamameters); i++ {
		parameter := stateParamameters[i]
		paramName := parameter.Parametername.ValueString()
		paramType := parameter.ParameterType.ValueString()
		paramMap[paramName] = paramType
	}

	var parameterList []britive_client.ResourceManagerResourceTypeParameterPlan
	for _, parameter := range resourceType.Parameters {
		parameterList = append(parameterList, britive_client.ResourceManagerResourceTypeParameterPlan{
			Parametername: types.StringValue(parameter.ParamName),
			ParameterType: types.StringValue(paramMap[parameter.ParamName]),
			IsMandatory:   types.BoolValue(parameter.IsMandatory),
		})
	}

	plan.Parameters, err = rth.mapParameterListToSet(ctx, parameterList)
	if err != nil {
		return nil, err
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
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"param_name":   types.StringType,
				"param_type":   types.StringType,
				"is_mandatory": types.BoolType,
			},
		},
		paramList,
	)

	if diags.HasError() {
		return types.Set{}, fmt.Errorf("Failed to map parameters list")
	}

	return set, nil
}
