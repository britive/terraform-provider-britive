package resourcemanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceResourceManagerResourceLabel{}
	_ resource.ResourceWithConfigure   = &ResourceResourceManagerResourceLabel{}
	_ resource.ResourceWithImportState = &ResourceResourceManagerResourceLabel{}
)

type ResourceResourceManagerResourceLabel struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResourceLabelHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResourceLabelHelper struct{}

func NewResourceResourceManagerResourceLabel() resource.Resource {
	return &ResourceResourceManagerResourceLabel{}
}

func NewResourceResourceManagerResourceLabelHelper() *ResourceResourceManagerResourceLabelHelper {
	return &ResourceResourceManagerResourceLabelHelper{}
}

func (rrmrl *ResourceResourceManagerResourceLabel) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_resource_label"
}

func (rrmrl *ResourceResourceManagerResourceLabel) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Resource Manager Resource Label resource")

	if req.ProviderData == nil {
		return
	}

	rrmrl.client = req.ProviderData.(*britive_client.Client)
	if rrmrl.client == nil {
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
	rrmrl.helper = NewResourceResourceManagerResourceLabelHelper()
}

type caseInsensitiveStringModifier struct{}

func (m caseInsensitiveStringModifier) Description(ctx context.Context) string {
	return "Suppress diff if values differ only by case"
}

func (m caseInsensitiveStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m caseInsensitiveStringModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}

	old := req.StateValue.ValueString()
	new := req.PlanValue.ValueString()

	if strings.EqualFold(old, new) {
		resp.PlanValue = req.StateValue
	}
}

func CaseInsensitiveString() planmodifier.String {
	return caseInsensitiveStringModifier{}
}

func (rrmrl *ResourceResourceManagerResourceLabel) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Resource Label resource",
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
				Description: "The name of Label",
				Validators: []validator.String{
					validate.StringFunc(
						"name",
						validate.StringWithNoSpecialChar(),
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Resource label",
			},
			"internal": schema.BoolAttribute{
				Computed:    true,
				Description: "Resource Label internal",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"label_color": schema.StringAttribute{
				Optional:    true,
				Description: "Color of label",
				PlanModifiers: []planmodifier.String{
					CaseInsensitiveString(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"values": schema.SetNestedBlock{
				Description: "Resource label value",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value_id": schema.StringAttribute{
							Computed:    true,
							Description: "Resource label value ID",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Resource label value name",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Resource label value description",
						},
						"created_by": schema.Int64Attribute{
							Computed:    true,
							Description: "Resource label value createdBy",
						},
						"updated_by": schema.Int64Attribute{
							Computed:    true,
							Description: "Resource label value updatedBy",
						},
						"created_on": schema.StringAttribute{
							Computed:    true,
							Description: "Resource label value createdOn",
						},
						"updated_on": schema.StringAttribute{
							Computed:    true,
							Description: "Resource label value updatedOn",
						},
					},
				},
			},
		},
	}
}

func (rrmrl *ResourceResourceManagerResourceLabel) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_resource_manager_resource_label")
	var plan britive_client.ResourceManagerResourceLabelPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during resource label creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Mapping Resource Label Resource to Model")

	resourceLabel := &britive_client.ResourceLabel{}
	err := rrmrl.helper.mapResourceToModel(ctx, plan, resourceLabel)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource label", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map resource label to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, "Creating Resource Label Resource")
	resourceLabel, err = rrmrl.client.CreateUpdateResourceLabel(ctx, *resourceLabel, false)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource label", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource label, error:%#v", err))
		return
	}

	plan.ID = types.StringValue(rrmrl.helper.generateUniqueID(resourceLabel.LabelId))

	tflog.Info(ctx, "Created Resource Label Resource")

	planPtr, err := rrmrl.helper.getAndMapModelToPlan(ctx, plan, rrmrl.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource label",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource label model to plan", map[string]interface{}{
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
		"resource_label": planPtr,
	})
}

func (rrmrl *ResourceResourceManagerResourceLabel) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_resource_label")

	if rrmrl.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResourceLabelPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get resource label state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := rrmrl.helper.getAndMapModelToPlan(ctx, state, rrmrl.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource label",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map resource label model to plan failed in Read", map[string]interface{}{
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

	tflog.Info(ctx, "Read completed for britive_resource_manager_resource_label")
}

func (rrmrl *ResourceResourceManagerResourceLabel) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_resource_manager_resource_label")

	var plan, state britive_client.ResourceManagerResourceLabelPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	labelId, err := rrmrl.helper.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource label", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse label id, error:%#v", err))
		return
	}

	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) || !plan.LabelColor.Equal(state.LabelColor) || !plan.Values.Equal(state.Values) {
		resourceLabel := &britive_client.ResourceLabel{
			LabelId: labelId,
		}
		err := rrmrl.helper.mapResourceToModel(ctx, plan, resourceLabel)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource label", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map resource label to model, error:%#v", err))
			return
		}

		resourceLabel, err = rrmrl.client.CreateUpdateResourceLabel(ctx, *resourceLabel, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource_label", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource_label, error:%#v", err))
			return
		}
		plan.ID = types.StringValue(rrmrl.helper.generateUniqueID(resourceLabel.LabelId))
	}

	planPtr, err := rrmrl.helper.getAndMapModelToPlan(ctx, plan, rrmrl.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource label",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource label model to plan", map[string]interface{}{
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

func (rrmrl *ResourceResourceManagerResourceLabel) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_resource_label")

	var state britive_client.ResourceManagerResourceLabelPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	labelId, err := rrmrl.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource label", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse label id, error:%#v", err))
		return
	}

	tflog.Info(ctx, "Deleting Resource Label Resource")
	err = rrmrl.client.DeleteResourceLabel(ctx, labelId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource label", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete resource label, error:%#v", err))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "Deleted Resource Label Resource")
}

func (rrmrl *ResourceResourceManagerResourceLabel) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rrmrl.importHelper.ParseImportID([]string{"resource-manager/resource-labels/(?P<id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Invalid importID", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Invalid importID, error:%#v", err))
		return
	}
	labelId := importData.Fields["id"]
	tflog.Info(ctx, fmt.Sprintf("Importing resource label: %s", labelId))

	body, err := rrmrl.client.GetResourceLabel(ctx, labelId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import resource label", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import resource label, error:%#v", err))
		return
	}

	plan := britive_client.ResourceManagerResourceLabelPlan{
		ID: types.StringValue(body.LabelId),
	}

	planPtr, err := rrmrl.helper.getAndMapModelToPlan(ctx, plan, rrmrl.client)
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

	tflog.Info(ctx, fmt.Sprintf("Imported resource label: %#v", planPtr))
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourceLabelPlan, c *britive_client.Client) (*britive_client.ResourceManagerResourceLabelPlan, error) {
	labelId, err := rrmrlh.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading Resource Label Resource of %s", labelId))
	resourceLabel, err := c.GetResourceLabel(ctx, labelId)
	if err != nil {
		return nil, err
	}

	plan.Name = types.StringValue(resourceLabel.Name)

	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && resourceLabel.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(resourceLabel.Description)
	}

	plan.Internal = types.BoolValue(resourceLabel.Internal)

	if (plan.LabelColor.IsNull() || plan.LabelColor.IsUnknown()) && resourceLabel.LabelColor == "" {
		plan.LabelColor = types.StringNull()
	} else {
		plan.LabelColor = types.StringValue(resourceLabel.LabelColor)
	}

	if (plan.Values.IsNull() || plan.Values.IsUnknown()) && len(resourceLabel.Values) == 0 {
		plan.Values = types.SetNull(rrmrlh.getResourceLabelValueType())
	} else {
		var resourceLabelValues []britive_client.ResourceManagerResourceLabelValuePlan
		for _, val := range resourceLabel.Values {
			resourceLabelValue := britive_client.ResourceManagerResourceLabelValuePlan{
				ValueID:     types.StringValue(val.ValueId),
				Name:        types.StringValue(val.Name),
				Description: types.StringValue(val.Description),
				CreatedBy:   types.Int64Value(int64(val.CreatedBy)),
				UpdatedBy:   types.Int64Value(int64(val.UpdatedBy)),
				CreatedOn:   types.StringValue(val.CreatedOn),
				UpdatedOn:   types.StringValue(val.UpdatedOn),
			}

			resourceLabelValues = append(resourceLabelValues, resourceLabelValue)
		}
		plan.Values, err = rrmrlh.mapValuesListToSet(ctx, resourceLabelValues)
		if err != nil {
			return nil, err
		}
	}
	return &plan, nil
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) mapResourceToModel(ctx context.Context, plan britive_client.ResourceManagerResourceLabelPlan, resourceLabel *britive_client.ResourceLabel) error {
	resourceLabel.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		resourceLabel.Description = plan.Description.ValueString()
	}
	if !plan.LabelColor.IsNull() && !plan.LabelColor.IsUnknown() {
		resourceLabel.LabelColor = plan.LabelColor.ValueString()
	}

	var resourceLabelValues []britive_client.ResourceManagerResourceLabelValuePlan
	if !plan.Values.IsNull() && !plan.Values.IsUnknown() {
		var err error
		resourceLabelValues, err = rrmrlh.mapValuesSetToList(ctx, plan)
		if err != nil {
			return nil
		}
	}
	for _, val := range resourceLabelValues {
		resourceLabelValue := &britive_client.ResourceLabelValue{
			Name: val.Name.ValueString(),
		}
		if !val.Description.IsNull() && !val.Description.IsUnknown() {
			resourceLabelValue.Description = val.Description.ValueString()
		}
		resourceLabel.Values = append(resourceLabel.Values, *resourceLabelValue)
	}

	return nil
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) generateUniqueID(labelId string) string {
	return fmt.Sprintf("resource-manager/resource-labels/%s", labelId)
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) parseUniqueID(labelId string) (string, error) {
	labelIdArr := strings.Split(labelId, "/")
	if len(labelIdArr) != 3 {
		return "", errs.NewNotFoundErrorf("Resource Label Id")
	}
	labelId = labelIdArr[len(labelIdArr)-1]
	return labelId, nil
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) mapValuesSetToList(ctx context.Context, plan britive_client.ResourceManagerResourceLabelPlan) ([]britive_client.ResourceManagerResourceLabelValuePlan, error) {
	set := plan.Values

	var valueList []britive_client.ResourceManagerResourceLabelValuePlan
	diags := set.ElementsAs(ctx, &valueList, true)
	if diags.HasError() {
		return nil, fmt.Errorf("Failed to map values")
	}

	return valueList, nil
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) mapValuesListToSet(ctx context.Context, list []britive_client.ResourceManagerResourceLabelValuePlan) (types.Set, error) {
	attrType := rrmrlh.getResourceLabelValueType()
	set, diags := types.SetValueFrom(
		ctx,
		attrType,
		list,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("Failed to map values")
	}

	return set, nil
}

func (rrmrlh *ResourceResourceManagerResourceLabelHelper) getResourceLabelValueType() attr.Type {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"value_id":    types.StringType,
			"name":        types.StringType,
			"description": types.StringType,
			"created_by":  types.Int64Type,
			"updated_by":  types.Int64Type,
			"created_on":  types.StringType,
			"updated_on":  types.StringType,
		},
	}
}
