package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResponseTemplateResource struct {
	client *britive.Client
}

type ResponseTemplateResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	TemplateID              types.String `tfsdk:"template_id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	IsConsoleAccessEnabled  types.Bool   `tfsdk:"is_console_access_enabled"`
	ShowOnUI                types.Bool   `tfsdk:"show_on_ui"`
	TemplateData            types.String `tfsdk:"template_data"`
}

func NewResponseTemplateResource() resource.Resource {
	return &ResponseTemplateResource{}
}

func (r *ResponseTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_response_template"
}

func (r *ResponseTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager response template",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The template ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_id": schema.StringAttribute{
				Description: "The unique identifier of the response template",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the response template",
				Required:    true,
				Validators: []validator.String{
					validators.Alphanumeric(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the response template",
				Optional:    true,
			},
			"is_console_access_enabled": schema.BoolAttribute{
				Description: "Boolean flag to enable console access",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"show_on_ui": schema.BoolAttribute{
				Description: "Boolean flag to determine if the template is visible on the UI",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"template_data": schema.StringAttribute{
				Description: "The template content with placeholders",
				Optional:    true,
			},
		},
	}
}

func (r *ResponseTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *ResponseTemplateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ResponseTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.IsConsoleAccessEnabled.ValueBool() && data.ShowOnUI.ValueBool() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Both 'is_console_access_enabled' and 'show_on_ui' cannot be true at the same time",
		)
	}
}

func (r *ResponseTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResponseTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template := britive.ResponseTemplate{
		Name:                   plan.Name.ValueString(),
		Description:            plan.Description.ValueString(),
		IsConsoleAccessEnabled: plan.IsConsoleAccessEnabled.ValueBool(),
		ShowOnUI:               plan.ShowOnUI.ValueBool(),
		TemplateData:           plan.TemplateData.ValueString(),
	}

	log.Printf("[INFO] Creating response template: %#v", template)

	created, err := r.client.CreateResponseTemplate(template)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Response Template", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/response-templates/%s", created.TemplateID))
	plan.TemplateID = types.StringValue(created.TemplateID)

	log.Printf("[INFO] Created response template: %s", created.TemplateID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResponseTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResponseTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := parseTemplateID(state.ID.ValueString())

	log.Printf("[INFO] Reading response template: %s", templateID)

	template, err := r.client.GetResponseTemplate(templateID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Response Template", err.Error())
		return
	}

	state.TemplateID = types.StringValue(template.TemplateID)
	state.Name = types.StringValue(template.Name)
	state.Description = types.StringValue(template.Description)
	state.IsConsoleAccessEnabled = types.BoolValue(template.IsConsoleAccessEnabled)
	state.ShowOnUI = types.BoolValue(template.ShowOnUI)
	state.TemplateData = types.StringValue(template.TemplateData)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResponseTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResponseTemplateResourceModel
	var state ResponseTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := parseTemplateID(state.ID.ValueString())

	template := britive.ResponseTemplate{
		TemplateID:             templateID,
		Name:                   plan.Name.ValueString(),
		Description:            plan.Description.ValueString(),
		IsConsoleAccessEnabled: plan.IsConsoleAccessEnabled.ValueBool(),
		ShowOnUI:               plan.ShowOnUI.ValueBool(),
		TemplateData:           plan.TemplateData.ValueString(),
	}

	log.Printf("[INFO] Updating response template: %s", templateID)

	_, err := r.client.UpdateResponseTemplate(templateID, template)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Response Template", err.Error())
		return
	}

	log.Printf("[INFO] Updated response template: %s", templateID)

	plan.TemplateID = types.StringValue(templateID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResponseTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResponseTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := parseTemplateID(state.ID.ValueString())

	log.Printf("[INFO] Deleting response template: %s", templateID)

	err := r.client.DeleteResponseTemplate(templateID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Response Template", err.Error())
		return
	}

	log.Printf("[INFO] Deleted response template: %s", templateID)
}

func (r *ResponseTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var templateID string

	if strings.HasPrefix(importID, "resource-manager/response-templates/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Import ID must be 'resource-manager/response-templates/{id}' or '{id}', got: %s", importID),
			)
			return
		}
		templateID = parts[2]
	} else {
		templateID = importID
	}

	if strings.TrimSpace(templateID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Template ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing response template: %s", templateID)

	template, err := r.client.GetResponseTemplate(templateID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Response Template Not Found", fmt.Sprintf("Template %s not found", templateID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Response Template", err.Error())
		return
	}

	var state ResponseTemplateResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/response-templates/%s", template.TemplateID))
	state.TemplateID = types.StringValue(template.TemplateID)
	state.Name = types.StringValue(template.Name)
	state.Description = types.StringValue(template.Description)
	state.IsConsoleAccessEnabled = types.BoolValue(template.IsConsoleAccessEnabled)
	state.ShowOnUI = types.BoolValue(template.ShowOnUI)
	state.TemplateData = types.StringValue(template.TemplateData)

	log.Printf("[INFO] Imported response template: %s", templateID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func parseTemplateID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return id
}
