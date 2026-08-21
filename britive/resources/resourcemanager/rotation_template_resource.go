package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RotationTemplateResource is the resource implementation.
type RotationTemplateResource struct {
	client *britive.Client
}

// RotationTemplateResourceModel describes the resource data model.
type RotationTemplateResourceModel struct {
	ID             types.String                    `tfsdk:"id"`
	TemplateID     types.String                    `tfsdk:"template_id"`
	ResourceTypeID types.String                    `tfsdk:"resource_type_id"`
	Name           types.String                    `tfsdk:"name"`
	Description    types.String                    `tfsdk:"description"`
	TimeLimit      types.Int64                     `tfsdk:"time_limit"`
	TemplateType   types.String                    `tfsdk:"template_type"`
	ScriptFilePath types.String                    `tfsdk:"script_file_path"`
	ScriptFileHash types.String                    `tfsdk:"script_file_hash"`
	ScriptContent  types.String                    `tfsdk:"script_content"`
	ScriptLanguage types.String                    `tfsdk:"script_language"`
	ScriptName     types.String                    `tfsdk:"script_name"`
	Variables      []RotationTemplateVariableModel `tfsdk:"variables"`
}

// RotationTemplateVariableModel describes a single variable exposed to the template's script.
type RotationTemplateVariableModel struct {
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	MultiValued types.Bool   `tfsdk:"multi_valued"`
}

// NewRotationTemplateResource is a helper function to simplify the provider implementation.
func NewRotationTemplateResource() resource.Resource {
	return &RotationTemplateResource{}
}

// Metadata returns the resource type name.
func (r *RotationTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_type_rotation_template"
}

// Schema defines the schema for the resource.
func (r *RotationTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager rotation template",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the rotation template",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_id": schema.StringAttribute{
				Description: "The unique identifier of the rotation template",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Description: "The ID of the associated resource type",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the rotation template. Never observed in any update payload during API capture, so treated as immutable - confirm with the API owner if a rename endpoint exists.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the rotation template. Cannot be changed once set - the API never accepts a description update, so (unlike name) changing this fails the plan outright rather than replacing the resource. Confirm with the API owner if a separate update endpoint exists.",
				Optional:    true,
				Computed:    true,
			},
			"time_limit": schema.Int64Attribute{
				Description: "The time limit in minutes for the rotation script to run (converted to seconds on the wire). Defaults to 1. The valid range is enforced by the API, not this provider, since it may change independently of a provider release.",
				Optional:    true,
				Computed:    true,
				Default:     staticInt64Default{1},
			},
			"template_type": schema.StringAttribute{
				Description: "The template mode: 'Local' (rotation handled by logic already deployed on the target, no script), 'InlineFile' (inline script content authored via script_content/script_language), or 'FilePath' (a local file uploaded via script_file_path)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Local", "InlineFile", "FilePath"),
				},
			},
			"script_file_path": schema.StringAttribute{
				Description: "Path to a local file to upload as the rotation script. Required when template_type = \"FilePath\"; must be unset otherwise.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("script_content"),
						path.MatchRoot("script_language"),
					),
				},
			},
			"script_file_hash": schema.StringAttribute{
				Description: "SHA-256 hash of the file at script_file_path, used to detect content drift since the path string alone doesn't change when the file's contents do",
				Computed:    true,
			},
			"script_content": schema.StringAttribute{
				Description: "Inline rotation script content. Required when template_type = \"InlineFile\"; must be unset otherwise.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("script_file_path"),
					),
				},
			},
			"script_language": schema.StringAttribute{
				Description: "The language of script_content. One of Text, Python, Batch, JavaScript, PowerShell, Shell. Only meaningful when template_type = \"InlineFile\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("text"),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Text", "Python", "Batch", "JavaScript", "PowerShell", "Shell"),
					stringvalidator.ConflictsWith(
						path.MatchRoot("script_file_path"),
					),
				},
			},
			"script_name": schema.StringAttribute{
				Description: "Server-derived script file name: the basename of script_file_path for FilePath mode, an auto-generated name for InlineFile mode, absent for Local mode",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"variables": schema.SetNestedBlock{
				Description: "Variables exposed to the rotation script",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The variable name",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "The variable type. One of String, Number, Date.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive("String", "Number", "Date"),
							},
						},
						"multi_valued": schema.BoolAttribute{
							Description: "Whether the variable accepts multiple values",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RotationTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig validates the resource configuration.
func (r *RotationTemplateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data RotationTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown values (e.g. computed from another resource) cannot be validated at this
	// stage; defer to apply time, mirroring resource_type_permission's guard.
	if data.TemplateType.IsUnknown() || data.ScriptFilePath.IsUnknown() || data.ScriptContent.IsUnknown() {
		return
	}

	hasFilePath := !data.ScriptFilePath.IsNull() && data.ScriptFilePath.ValueString() != ""
	hasContent := !data.ScriptContent.IsNull() && data.ScriptContent.ValueString() != ""

	switch strings.ToLower(data.TemplateType.ValueString()) {
	case "local":
		if hasContent || hasFilePath {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'template_type = "Local"' requires script_content and script_file_path to be unset`,
			)
		}
	case "inlinefile":
		if !hasContent {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'template_type = "InlineFile"' requires script_content to be set`,
			)
		}
		if hasFilePath {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'template_type = "InlineFile"' cannot be combined with script_file_path`,
			)
		}
	case "filepath":
		if !hasFilePath {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'template_type = "FilePath"' requires script_file_path to be set`,
			)
		}
		if hasContent {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`'template_type = "FilePath"' cannot be combined with script_content/script_language`,
			)
		}
	}
}

// ModifyPlan handles plan modification for file hash computation and description normalization.
func (r *RotationTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan RotationTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config RotationTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When description is absent from config (null), force plan to null so that a
	// Computed+Optional field does not carry forward a prior-state value.
	if config.Description.IsNull() && !plan.Description.IsNull() {
		plan.Description = types.StringNull()
	}

	// description cannot be changed once the resource exists: the API never accepts a
	// description update (confirmed by capture - absent from every PUT body observed).
	// Unlike name, this is deliberately NOT RequiresReplace - reject the change outright
	// instead of silently forcing a destroy+recreate the user didn't ask for.
	if !req.State.Raw.IsNull() {
		var state RotationTemplateResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !plan.Description.Equal(state.Description) {
			resp.Diagnostics.AddAttributeError(
				path.Root("description"),
				"Cannot Update Description",
				"The rotation template's description cannot be changed after creation - the API does not support updating it. Revert description to its current value, or destroy and recreate the resource if a different description is required.",
			)
			return
		}
	}

	// Compute the script file hash if a file is specified; clear to null when not using
	// a file. A plain else (not else-if IsUnknown) ensures the hash is cleared even when
	// removing a previously-set file.
	if !plan.ScriptFilePath.IsNull() && plan.ScriptFilePath.ValueString() != "" {
		newHash, err := hashFileContent(plan.ScriptFilePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Hashing Script File", err.Error())
			return
		}
		plan.ScriptFileHash = types.StringValue(newHash)
	} else {
		plan.ScriptFileHash = types.StringNull()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create creates the resource and sets the initial Terraform state.
func (r *RotationTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RotationTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(plan.ResourceTypeID.ValueString())

	createReq := britive.RotationTemplateCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	log.Printf("[INFO] Creating rotation template draft: %#v", createReq)

	created, err := r.client.CreateRotationTemplate(resourceTypeID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Rotation Template", err.Error())
		return
	}

	templateID := created.TemplateID
	log.Printf("[INFO] Finalizing rotation template: %s", templateID)

	template := r.buildUpdatePayload(&plan)

	// From this point on, a failure leaves a real draft on the server that Terraform
	// doesn't yet track in state - the next apply would retry CreateRotationTemplate with
	// the same name and hit "name must be unique for a resource type". Roll the draft back
	// on any failure past this point, mirroring resource_type_resource.go's cleanup of a
	// just-created resource type when the icon upload step fails.
	if err := r.uploadScript(resourceTypeID, templateID, &plan, &template); err != nil {
		resp.Diagnostics.AddError("Error Uploading Rotation Template Script", err.Error())
		if delErr := r.client.DeleteRotationTemplate(resourceTypeID, templateID); delErr != nil {
			resp.Diagnostics.AddError("Error Cleaning Up Rotation Template", delErr.Error())
		}
		return
	}

	if _, err := r.client.UpdateRotationTemplate(resourceTypeID, templateID, template); err != nil {
		resp.Diagnostics.AddError("Error Updating Rotation Template", err.Error())
		if delErr := r.client.DeleteRotationTemplate(resourceTypeID, templateID); delErr != nil {
			resp.Diagnostics.AddError("Error Cleaning Up Rotation Template", delErr.Error())
		}
		return
	}

	plan.ID = types.StringValue(rotationTemplateCompositeID(resourceTypeID, templateID))
	plan.TemplateID = types.StringValue(templateID)

	// Read back to get computed values
	full, err := r.client.GetRotationTemplate(resourceTypeID, templateID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Rotation Template", err.Error())
		return
	}

	r.mapModelToResource(full, &plan, false)

	log.Printf("[INFO] Created rotation template: %s", templateID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RotationTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RotationTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())
	templateID := state.TemplateID.ValueString()

	log.Printf("[INFO] Reading rotation template: %s", templateID)

	template, err := r.client.GetRotationTemplate(resourceTypeID, templateID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Rotation Template", err.Error())
		return
	}

	r.mapModelToResource(template, &state, false)

	log.Printf("[INFO] Retrieved rotation template: %s", templateID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RotationTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RotationTemplateResourceModel
	var state RotationTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())
	templateID := state.TemplateID.ValueString()

	template := r.buildUpdatePayload(&plan)

	if err := r.uploadScript(resourceTypeID, templateID, &plan, &template); err != nil {
		resp.Diagnostics.AddError("Error Uploading Rotation Template Script", err.Error())
		return
	}

	log.Printf("[INFO] Updating rotation template: %s", templateID)

	if _, err := r.client.UpdateRotationTemplate(resourceTypeID, templateID, template); err != nil {
		resp.Diagnostics.AddError("Error Updating Rotation Template", err.Error())
		return
	}

	log.Printf("[INFO] Updated rotation template: %s", templateID)

	// Read back to get updated values
	full, err := r.client.GetRotationTemplate(resourceTypeID, templateID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Rotation Template", err.Error())
		return
	}

	plan.ID = state.ID
	plan.TemplateID = state.TemplateID

	r.mapModelToResource(full, &plan, false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RotationTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RotationTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())
	templateID := state.TemplateID.ValueString()

	log.Printf("[INFO] Deleting rotation template: %s", templateID)

	if err := r.client.DeleteRotationTemplate(resourceTypeID, templateID); err != nil {
		resp.Diagnostics.AddError("Error Deleting Rotation Template", err.Error())
		return
	}

	log.Printf("[INFO] Deleted rotation template: %s", templateID)
}

// ImportState imports the resource state.
func (r *RotationTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceTypeID, templateID, err := parseRotationTemplateCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	log.Printf("[INFO] Importing rotation template: %s", templateID)

	template, err := r.client.GetRotationTemplate(resourceTypeID, templateID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Rotation Template Not Found", fmt.Sprintf("Template %s not found under resource type %s", templateID, resourceTypeID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Rotation Template", err.Error())
		return
	}

	var state RotationTemplateResourceModel
	state.ID = types.StringValue(rotationTemplateCompositeID(resourceTypeID, templateID))
	state.ResourceTypeID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", resourceTypeID))
	state.TemplateID = types.StringValue(templateID)

	r.mapModelToResource(template, &state, true)

	// script_file_path/script_content aren't recoverable from the API on import (the API
	// only exposes uploaded content via a presigned download URL, not as plan-comparable
	// local state) - they stay null; the next plan will show them as needing to be set if
	// drift detection against a local source is wanted after import.

	log.Printf("[INFO] Imported rotation template: %s", templateID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// buildUpdatePayload maps the plan into the API's update request shape, deriving
// isLocal/inlineFile/editorType from template_type. Does not set Name/Description
// (never accepted by the update call) or ScriptName (set by uploadScript, or left
// unset for Local mode).
func (r *RotationTemplateResource) buildUpdatePayload(plan *RotationTemplateResourceModel) britive.RotationTemplate {
	template := britive.RotationTemplate{
		// The API's timeoutLimit is in seconds (bounds 60-900); time_limit is exposed to
		// users in minutes (matching the UI's own "In Minutes(MM)" label), so convert here.
		TimeoutLimit: int(plan.TimeLimit.ValueInt64()) * 60,
		EditorType:   "text",
		Variables:    make([]britive.RotationTemplateVariable, 0, len(plan.Variables)),
	}

	for _, v := range plan.Variables {
		template.Variables = append(template.Variables, britive.RotationTemplateVariable{
			Name:        v.Name.ValueString(),
			Type:        canonicalVariableType(v.Type.ValueString()),
			MultiValued: v.MultiValued.ValueBool(),
		})
	}

	switch strings.ToLower(plan.TemplateType.ValueString()) {
	case "local":
		template.IsLocal = true
		template.InlineFile = false
	case "inlinefile":
		template.IsLocal = false
		template.InlineFile = true
		template.EditorType = strings.ToLower(plan.ScriptLanguage.ValueString())
	case "filepath":
		template.IsLocal = false
		template.InlineFile = false
	}

	return template
}

// uploadScript uploads the script content/file for InlineFile/FilePath modes and stamps
// the resulting scriptName onto template. A no-op for Local mode. Always re-uploads when
// called (no diff check against prior content), mirroring resource_type_permission's
// Create/Update, which does the same for its checkin/checkout file and code uploads.
func (r *RotationTemplateResource) uploadScript(resourceTypeID, templateID string, plan *RotationTemplateResourceModel, template *britive.RotationTemplate) error {
	switch strings.ToLower(plan.TemplateType.ValueString()) {
	case "inlinefile":
		if err := r.client.UploadRotationTemplateScriptCode(resourceTypeID, templateID, plan.ScriptContent.ValueString(), plan.ScriptLanguage.ValueString()); err != nil {
			return err
		}
		template.ScriptName = templateID + "_rotation_template_file"
	case "filepath":
		if err := r.client.UploadRotationTemplateScriptFile(resourceTypeID, templateID, plan.ScriptFilePath.ValueString()); err != nil {
			return err
		}
		template.ScriptName = filepath.Base(plan.ScriptFilePath.ValueString())
	}
	return nil
}

// canonicalVariableType normalizes a case-insensitive `type` input to the exact casing
// confirmed to work against the API by capture ("String"/"Number"/"Date") - the API's
// case-sensitivity for this field is unverified, so this always sends the one casing
// actually observed working rather than passing the user's casing through as-is.
func canonicalVariableType(t string) string {
	switch strings.ToLower(t) {
	case "string":
		return "String"
	case "number":
		return "Number"
	case "date":
		return "Date"
	default:
		return t
	}
}

// mapModelToResource maps the API's rotation template detail onto Terraform state.
//
// template_type, script_language, and variables[].type are validated case-insensitively
// but sent to the API in one fixed casing (see canonicalVariableType and the ToLower calls
// in buildUpdatePayload/uploadScript) - so the value echoed back from the API is not
// necessarily the casing the user typed. Without correction that would show a perpetual
// case-only diff on every plan. Mirroring resource_type_resource.go's paramMap trick: when
// not importing, and the API's value case-insensitively matches what's already in state,
// keep the user's original casing instead of overwriting it.
func (r *RotationTemplateResource) mapModelToResource(template *britive.RotationTemplate, state *RotationTemplateResourceModel, imported bool) {
	state.Name = types.StringValue(template.Name)
	state.Description = preserveOptionalString(template.Description, state.Description)
	// Converting back from the wire's seconds to the schema's minutes; see buildUpdatePayload.
	state.TimeLimit = types.Int64Value(int64(template.TimeoutLimit) / 60)
	state.ScriptName = optionalStringValue(template.ScriptName)

	computedTemplateType := "FilePath"
	switch {
	case template.IsLocal:
		computedTemplateType = "Local"
	case template.InlineFile:
		computedTemplateType = "InlineFile"
	}
	if !imported && strings.EqualFold(state.TemplateType.ValueString(), computedTemplateType) {
		// Prior state already matches (case-insensitively) - keep the user's casing.
	} else {
		state.TemplateType = types.StringValue(computedTemplateType)
	}

	computedLanguage := optionalStringValue(template.EditorType)
	if !imported && !state.ScriptLanguage.IsNull() && strings.EqualFold(state.ScriptLanguage.ValueString(), computedLanguage.ValueString()) {
		// Prior state already matches (case-insensitively) - keep the user's casing.
	} else {
		state.ScriptLanguage = computedLanguage
	}

	variableTypeMap := make(map[string]string)
	if !imported {
		for _, v := range state.Variables {
			variableTypeMap[v.Name.ValueString()] = v.Type.ValueString()
		}
	}

	variables := make([]RotationTemplateVariableModel, 0, len(template.Variables))
	for _, v := range template.Variables {
		varType := v.Type
		if !imported {
			if userType, ok := variableTypeMap[v.Name]; ok && strings.EqualFold(userType, v.Type) {
				varType = userType
			}
		}
		variables = append(variables, RotationTemplateVariableModel{
			Name:        types.StringValue(v.Name),
			Type:        types.StringValue(varType),
			MultiValued: types.BoolValue(v.MultiValued),
		})
	}
	state.Variables = variables
}

// lastPathSegment strips any composite-ID path prefix (e.g. a cross-resource reference
// like britive_resource_manager_resource_type.example.id, which is itself
// "resource-manager/resource-types/{id}"), returning just the final segment.
func lastPathSegment(id string) string {
	parts := strings.Split(id, "/")
	return parts[len(parts)-1]
}

// rotationTemplateCompositeID builds the resource's `id` value from resourceTypeID/templateID.
func rotationTemplateCompositeID(resourceTypeID, templateID string) string {
	return fmt.Sprintf("resource-manager/resource-types/%s/rotation-templates/%s", resourceTypeID, templateID)
}

// parseRotationTemplateCompositeID extracts resourceTypeID and templateID from either the
// composite ID ("resource-manager/resource-types/{resource_type_id}/rotation-templates/{template_id}")
// or a bare "{resource_type_id}/{template_id}" pair, for use on import.
func parseRotationTemplateCompositeID(id string) (resourceTypeID string, templateID string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 5:
		return parts[2], parts[4], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", errs.NewInvalidResourceIDError("rotation template", id)
	}
}
