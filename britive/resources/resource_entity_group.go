package resources

import (
	"context"
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
	_ resource.Resource                = &ResourceEntityGroup{}
	_ resource.ResourceWithConfigure   = &ResourceEntityGroup{}
	_ resource.ResourceWithImportState = &ResourceEntityGroup{}
)

type ResourceEntityGroup struct {
	client       *britive_client.Client
	helper       *ResourceEntityGroupHelper
	importHelper *imports.ImportHelper
}

type ResourceEntityGroupHelper struct{}

func NewResourceEntityGroup() resource.Resource {
	return &ResourceEntityGroup{}
}

func NewResourceEntityGroupHelper() *ResourceEntityGroupHelper {
	return &ResourceEntityGroupHelper{}
}

func (reg *ResourceEntityGroup) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_entity_group"
}

func (reg *ResourceEntityGroup) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Application Entity Group resource")

	if req.ProviderData == nil {
		return
	}

	reg.client = req.ProviderData.(*britive_client.Client)
	if reg.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Entity Group")
	reg.helper = NewResourceEntityGroupHelper()
}

func (reg *ResourceEntityGroup) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for entity group resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The identity of the application entity of type environment group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The identity of the Britive application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"applicationID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"entity_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the entity",
				Validators: []validator.String{
					validate.StringFunc(
						"entityName",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"entity_description": schema.StringAttribute{
				Required:    true,
				Description: "The description of the entity",
				Validators: []validator.String{
					validate.StringFunc(
						"entityDescription",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent id under which the environment group will be created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"parentID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
		},
	}
}

func (reg *ResourceEntityGroup) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_entity_group")

	var plan britive_client.EntityGroupPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during entity_group creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	applicationEntity := britive_client.ApplicationEntityGroup{}

	reg.helper.mapResourceToModel(plan, &applicationEntity)

	tflog.Info(ctx, fmt.Sprintf("Creating new application entity group: %#v", applicationEntity))

	applicationID := plan.ApplicationID.ValueString()

	ae, err := reg.client.CreateEntityGroup(ctx, applicationEntity, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create entity group", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create entity group, error:%#v", err))
		return
	}

	plan.ID = types.StringValue(reg.helper.generateUniqueID(applicationID, ae.EntityID))

	planPtr, err := reg.helper.getAndMapModelToPlan(ctx, plan, *reg.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after create",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map entity group model to plan", map[string]interface{}{
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
		"entity_group": planPtr,
	})
}

func (reg *ResourceEntityGroup) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_entity_group")

	if reg.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.EntityGroupPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get entity group state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := reg.helper.getAndMapModelToPlan(ctx, state, *reg.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get entity group",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map entity group model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Fetched entity group:  %#v", planPtr))
}

func (reg *ResourceEntityGroup) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_entity_group")

	var plan, state britive_client.EntityGroupPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	var hasChanges bool
	if !plan.ApplicationID.Equal(state.ApplicationID) || !plan.EntityName.Equal(state.EntityName) || !plan.EntityDescription.Equal(state.EntityDescription) || !plan.ParentID.Equal(state.ParentID) {
		hasChanges = true
		applicationID, _, err := reg.helper.parseUniqueID(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to update entity group", "ApplicationID or EntityID not found")
			tflog.Error(ctx, fmt.Sprintf("Failed to parse application or entity id: %#v", err))
			return
		}

		applicationEntity := britive_client.ApplicationEntityGroup{}

		reg.helper.mapResourceToModel(plan, &applicationEntity)

		tflog.Info(ctx, fmt.Sprintf("Updating the entity group %#v for application %s", applicationEntity, applicationID))

		ae, err := reg.client.UpdateEntityGroup(ctx, applicationEntity, applicationID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update entity group", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update entity group: %#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated application entity group: %#v", ae))
		plan.ID = types.StringValue(reg.helper.generateUniqueID(applicationID, ae.EntityID))
	}
	if hasChanges {
		planPtr, err := reg.helper.getAndMapModelToPlan(ctx, plan, *reg.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after create",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map entity group model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated entity group: %#v", planPtr))
	}
}

func (reg *ResourceEntityGroup) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_entity_group")

	var state britive_client.EntityGroupPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	applicationID, entityID, err := reg.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete entity group", "appicationID or entityID not found")
		tflog.Error(ctx, fmt.Sprintf("Failed to delete entity group: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting entity group %s for application %s", entityID, applicationID))
	err = reg.client.DeleteEntityGroup(ctx, applicationID, entityID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete entity group", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete entity group: %#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Deleted entity group %s for application %s", entityID, applicationID))
	resp.State.RemoveResource(ctx)
}

func (reg *ResourceEntityGroup) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}
	if err := reg.importHelper.ParseImportID([]string{"apps/(?P<application_id>[^/]+)/root-environment-group/groups/(?P<entity_id>[^/]+)", "(?P<application_id>[^/]+)/groups/(?P<entity_id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import entity group", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	applicationID := importData.Fields["application_id"]
	entityID := importData.Fields["entity_id"]
	if strings.TrimSpace(applicationID) == "" {
		resp.Diagnostics.AddError("Failed to import entitty group", "Invalid applicationID")
		tflog.Error(ctx, "Failed to import entity group, Invalid applicationID")
		return
	}
	if strings.TrimSpace(entityID) == "" {
		resp.Diagnostics.AddError("Failed to import entitty group", "Invalid entityID")
		tflog.Error(ctx, "Failed to import entity group, Invalid entityID")
		return
	}

	appEnvs := make([]britive_client.ApplicationEnvironment, 0)
	appEnvs, err := reg.client.GetAppEnvs(ctx, applicationID, "environmentGroups")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import entity group", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import entity group: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing entity group %s for application %s", entityID, applicationID))

	envIdList, err := reg.client.GetEnvDetails(appEnvs, "id")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import entity group", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import entity group, %#v", err))
		return
	}

	for _, id := range envIdList {
		if id == entityID {
			plan := &britive_client.EntityGroupPlan{
				ID: types.StringValue(reg.helper.generateUniqueID(applicationID, entityID)),
			}

			planPtr, err := reg.helper.getAndMapModelToPlan(ctx, *plan, *reg.client)
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to import entity group",
					fmt.Sprintf("Error: %v", err),
				)
				tflog.Error(ctx, "Failed import entity group model to plan", map[string]interface{}{
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

			tflog.Info(ctx, fmt.Sprintf("Imported entity group : %#v", planPtr))
			return
		}
	}

	resp.Diagnostics.AddError("Failed to import entity group", "Not found")
}

func (regh *ResourceEntityGroupHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.EntityGroupPlan, c britive_client.Client) (*britive_client.EntityGroupPlan, error) {
	applicationID, entityID, err := regh.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading entity group %s for application %s", entityID, applicationID))

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(ctx, applicationID)
	if err != nil || appRootEnvironmentGroup == nil {
		return nil, err
	}

	for _, association := range appRootEnvironmentGroup.EnvironmentGroups {
		if association.ID == entityID {
			tflog.Info(ctx, fmt.Sprintf("Received entity group: %#v", association))
			// To not allow the import of root environment group
			if association.ParentID == "" {
				return nil, fmt.Errorf("`parent_id` cannot be empty")
			}
			plan.EntityID = types.StringValue(entityID)
			plan.EntityName = types.StringValue(association.Name)
			plan.EntityDescription = types.StringValue(association.Description.(string))
			plan.ParentID = types.StringValue(association.ParentID)
			return &plan, nil
		}
	}

	return nil, errs.NewNotFoundErrorf("entity group %s for application %s", entityID, applicationID)
}

func (regh *ResourceEntityGroupHelper) mapResourceToModel(plan britive_client.EntityGroupPlan, applicationEntity *britive_client.ApplicationEntityGroup) {
	applicationEntity.Name = plan.EntityName.ValueString()
	applicationEntity.Description = plan.EntityDescription.ValueString()
	applicationEntity.ParentID = plan.ParentID.ValueString()
	applicationEntity.EntityID = plan.EntityID.ValueString()
}

func (resourceEntityGroupHelper *ResourceEntityGroupHelper) generateUniqueID(applicationID, entityID string) string {
	return fmt.Sprintf("apps/%s/root-environment-group/groups/%s", applicationID, entityID)
}

func (resourceEntityGroupHelper *ResourceEntityGroupHelper) parseUniqueID(ID string) (applicationID, entityID string, err error) {
	applicationEntityParts := strings.Split(ID, "/")
	if len(applicationEntityParts) < 5 {
		err = errs.NewInvalidResourceIDError("application entity group", ID)
		return
	}

	applicationID = applicationEntityParts[1]
	entityID = applicationEntityParts[4]
	return
}
