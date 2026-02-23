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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &ResourceResourceManagerResponseTemplate{}
	_ resource.ResourceWithConfigure      = &ResourceResourceManagerResponseTemplate{}
	_ resource.ResourceWithImportState    = &ResourceResourceManagerResponseTemplate{}
	_ resource.ResourceWithValidateConfig = &ResourceResourceManagerResponseTemplate{}
)

type ResourceResourceManagerResponseTemplate struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResponseTemplateHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResponseTemplateHelper struct{}

func NewResourceResourceManagerResponseTemplate() resource.Resource {
	return &ResourceResourceManagerResponseTemplate{}
}

func NewResourceResourceManagerResponseTemplateHelper() *ResourceResourceManagerResponseTemplateHelper {
	return &ResourceResourceManagerResponseTemplateHelper{}
}

func (rrt *ResourceResourceManagerResponseTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_response_template"
}

func (rrt *ResourceResourceManagerResponseTemplate) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Resource Manager Response Template resource")

	if req.ProviderData == nil {
		return
	}

	rrt.client = req.ProviderData.(*britive_client.Client)
	if rrt.client == nil {
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
	rrt.helper = NewResourceResourceManagerResponseTemplateHelper()
}

func (rrt *ResourceResourceManagerResponseTemplate) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Response Template resource",
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
				Description: "The name of template",
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
			"is_console_access_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "flag to enable console access",
				Default:     booldefault.StaticBool(false),
			},
			"show_on_ui": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Boolean flag to determine if the template is visible on the UI",
				Default:     booldefault.StaticBool(false),
			},
			"template_data": schema.StringAttribute{
				Optional:    true,
				Description: "The template content with placeholders",
			},
			"template_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the response template",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (rrt *ResourceResourceManagerResponseTemplate) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config britive_client.ResourceManagerResponseTemplatePlan

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		resp.Diagnostics.AddError("Invalid configuartion", "Failed to validate schema")
		tflog.Error(ctx, "Failed to validate configuration")
		return
	}

	if (!config.IsConsoleAccessEnabled.IsNull() || !config.IsConsoleAccessEnabled.IsUnknown()) && (!config.ShowOnUI.IsNull() || !config.ShowOnUI.IsUnknown()) && config.IsConsoleAccessEnabled.ValueBool() && config.ShowOnUI.ValueBool() {
		errorMsg := "both 'is_console_access_enabled' and 'show_on_ui' cannot be true at the same time"
		resp.Diagnostics.AddError("Invalid configuartion", errorMsg)
		tflog.Error(ctx, fmt.Sprintf("Failed to validate configuration, error: %s", errorMsg))
		return
	}
}

func (rrt *ResourceResourceManagerResponseTemplate) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_resource_manager_response_template")

	var plan britive_client.ResourceManagerResponseTemplatePlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Failed to read plan", "Invalid plan")
		tflog.Error(ctx, "Failed to read plan during response template creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	template := &britive_client.ResponseTemplate{}
	rrt.helper.mapResourceToModel(plan, template)

	tflog.Info(ctx, fmt.Sprintf("Creating response template: %#v", template))

	response, err := rrt.client.CreateResponseTemplate(ctx, *template)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create response template", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create response template, error:%#v", err))
		return
	}

	plan.ID = types.StringValue(rrt.helper.generateUniqueID(response.TemplateID))
	tflog.Info(ctx, fmt.Sprintf("Created response template: %#v", template))

	planPtr, err := rrt.helper.getAndMapModelToPlan(ctx, plan, *rrt.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get response template",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map response template model to plan", map[string]interface{}{
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
		"response template": planPtr,
	})
}

func (rrt *ResourceResourceManagerResponseTemplate) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_response_template")

	if rrt.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResponseTemplatePlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get response template state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := rrt.helper.getAndMapModelToPlan(ctx, state, *rrt.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get response template",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map response template model to plan failed in Read", map[string]interface{}{
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

	tflog.Info(ctx, "Read completed for britive_resource_manager_response_template")
}

func (rrt *ResourceResourceManagerResponseTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_resource_manager_response_template")

	var plan, state britive_client.ResourceManagerResponseTemplatePlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	templateID, err := rrt.helper.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update response template", "Invalid template ID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse response template id, error: %#v", err))
		return
	}

	var hasChanges bool
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) || !plan.IsConsoleAccessEnabled.Equal(state.IsConsoleAccessEnabled) || !plan.ShowOnUI.Equal(state.ShowOnUI) || !plan.TemplateData.Equal(state.TemplateData) {
		hasChanges = true

		template := &britive_client.ResponseTemplate{}
		rrt.helper.mapResourceToModel(plan, template)

		tflog.Info(ctx, fmt.Sprintf("Updating response template: %s", templateID))

		_, err := rrt.client.UpdateResponseTemplate(ctx, templateID, *template)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update response template", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update response template, error:%#v", err))
			return
		}
		tflog.Info(ctx, fmt.Sprintf("Updated response template: %s", templateID))
		plan.ID = types.StringValue(rrt.helper.generateUniqueID(templateID))
	}
	if hasChanges {
		planPtr, err := rrt.helper.getAndMapModelToPlan(ctx, plan, *rrt.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get response template",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map response temaplate model to plan", map[string]interface{}{
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

func (rrt *ResourceResourceManagerResponseTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_response_template")

	var state britive_client.ResourceManagerResponseTemplatePlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	templateID, err := rrt.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete response template", "Invalid response template ID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse response template, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting response template: %s", templateID))

	err = rrt.client.DeleteResponseTemplate(ctx, templateID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete response template", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete response template, error:%#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Resource template %s deleted", templateID))
	resp.State.RemoveResource(ctx)
}

func (rrt *ResourceResourceManagerResponseTemplate) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID

	importData := &imports.ImportHelperData{
		ID: importID,
	}

	if err := rrt.importHelper.ParseImportID([]string{"resource-manager/response-templates/(?P<id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import response template", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Faield to parse importID, error:%#v", err))
		return
	}
	templateID := importData.Fields["id"]
	tflog.Info(ctx, fmt.Sprintf("Importing response template: %s", templateID))

	response, err := rrt.client.GetResponseTemplate(ctx, templateID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import response template", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import response template, error:%#v", err))
		return
	}

	plan := britive_client.ResourceManagerResponseTemplatePlan{
		ID: types.StringValue(rrt.helper.generateUniqueID(response.TemplateID)),
	}
	planPtr, err := rrt.helper.getAndMapModelToPlan(ctx, plan, *rrt.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map response template model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported response template: %#v", planPtr))
}

func (rrth *ResourceResourceManagerResponseTemplateHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResponseTemplatePlan, c britive_client.Client) (*britive_client.ResourceManagerResponseTemplatePlan, error) {
	templateID, err := rrth.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, errs.NewInvalidResourceIDError("response template", plan.ID.ValueString())
	}

	tflog.Info(ctx, fmt.Sprintf("Reading response template: %s", templateID))

	template, err := c.GetResponseTemplate(ctx, templateID)
	if err != nil {
		if errors.Is(err, britive_client.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("response template %s", templateID)
		}
		return nil, err
	}

	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && template.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(template.Description)
	}

	plan.IsConsoleAccessEnabled = types.BoolValue(template.IsConsoleAccessEnabled)
	plan.ShowOnUI = types.BoolValue(template.ShowOnUI)

	if (plan.TemplateData.IsNull() || plan.TemplateData.IsUnknown()) && template.TemplateData == "" {
		plan.TemplateData = types.StringNull()
	} else {
		plan.TemplateData = types.StringValue(template.TemplateData)
	}

	plan.TemplateID = types.StringValue(template.TemplateID)

	return &plan, nil
}

func (rrth *ResourceResourceManagerResponseTemplateHelper) mapResourceToModel(plan britive_client.ResourceManagerResponseTemplatePlan, template *britive_client.ResponseTemplate) {
	template.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		template.Description = plan.Description.ValueString()
	}
	if !plan.IsConsoleAccessEnabled.IsNull() && !plan.IsConsoleAccessEnabled.IsUnknown() {
		template.IsConsoleAccessEnabled = plan.IsConsoleAccessEnabled.ValueBool()
	}
	if template.IsConsoleAccessEnabled {
		template.ShowOnUI = false
	} else {
		template.ShowOnUI = plan.ShowOnUI.ValueBool()
	}
	if !plan.TemplateData.IsNull() && !plan.TemplateData.IsUnknown() {
		template.TemplateData = plan.TemplateData.ValueString()
	}
}

func (rrth *ResourceResourceManagerResponseTemplateHelper) generateUniqueID(templateID string) string {
	return fmt.Sprintf("resource-manager/response-templates/%s", templateID)
}

func (rrth *ResourceResourceManagerResponseTemplateHelper) parseUniqueID(ID string) (responseTemplateID string, err error) {
	responseTemplatesParts := strings.Split(ID, "/")

	if len(responseTemplatesParts) < 3 {
		err = errs.NewInvalidResourceIDError("responseTemplates", ID)
		return
	}

	responseTemplateID = responseTemplatesParts[2]
	return
}
