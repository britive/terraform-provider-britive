package resourcemanager

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// ScanSettingsResource manages a resource type's scan settings.
//
// Unlike RotationTemplateResource, this is a resource-type-scoped singleton: exactly one
// scan settings object per resource type, no name/description, and the API creates/updates
// it via the same idempotent PUT (201 the first time, 200 after) rather than a
// POST-then-PUT two-step - so Create and Update share one upsert() implementation here.
// There's also no DELETE endpoint; Delete() resets the settings to Local mode with the
// API's own observed defaults, the closest equivalent to "un-configuring" a singleton that
// can't actually be removed.
type ScanSettingsResource struct {
	client *britive.Client
}

// ScanSettingsResourceModel describes the resource data model.
type ScanSettingsResourceModel struct {
	ID             types.String                    `tfsdk:"id"`
	ResourceTypeID types.String                    `tfsdk:"resource_type_id"`
	TimeLimit      types.Int64                     `tfsdk:"time_limit"`
	TemplateType   types.String                    `tfsdk:"template_type"`
	ScriptFilePath types.String                    `tfsdk:"script_file_path"`
	ScriptFileHash types.String                    `tfsdk:"script_file_hash"`
	ScriptContent  types.String                    `tfsdk:"script_content"`
	ScriptLanguage types.String                    `tfsdk:"script_language"`
	ScriptName     types.String                    `tfsdk:"script_name"`
	Variables      []RotationTemplateVariableModel `tfsdk:"variables"`
}

// scanSettingsDefaultTimeLimitMinutes matches the API's own observed default (1200 seconds)
// when scan settings is first created without an explicit timeoutLimit.
const scanSettingsDefaultTimeLimitMinutes = 20

// NewScanSettingsResource is a helper function to simplify the provider implementation.
func NewScanSettingsResource() resource.Resource {
	return &ScanSettingsResource{}
}

// Metadata returns the resource type name.
func (r *ScanSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_type_scan_settings"
}

// Schema defines the schema for the resource.
func (r *ScanSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager resource type's scan settings",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the scan settings, derived from resource_type_id alone",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Description: "The ID of the associated resource type. Exactly one scan settings resource exists per resource type.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"time_limit": schema.Int64Attribute{
				Description: "The time limit in minutes for the scan script to run (converted to seconds on the wire). Defaults to 20, matching the API's own default. The valid range is enforced by the API, not this provider.",
				Optional:    true,
				Computed:    true,
				Default:     staticInt64Default{scanSettingsDefaultTimeLimitMinutes},
			},
			"template_type": schema.StringAttribute{
				Description: "The scan settings mode: 'Local' (scanning handled by logic already deployed on the target, no script), 'InlineFile' (inline script content authored via script_content/script_language), or 'FilePath' (a local file uploaded via script_file_path)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Local", "InlineFile", "FilePath"),
				},
			},
			"script_file_path": schema.StringAttribute{
				Description: "Path to a local file to upload as the scan script. Required when template_type = \"FilePath\"; must be unset otherwise. Content-Type on upload is derived from the file's own extension.",
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
				Description: "Inline scan script content. Required when template_type = \"InlineFile\"; must be unset otherwise.",
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
				Description: "Server-derived script file name: the basename of script_file_path for FilePath mode, an auto-generated name for InlineFile mode, empty for Local mode. Predicted at plan time by ModifyPlan rather than via a plan modifier.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"variables": schema.SetNestedBlock{
				Description: "Variables exposed to the scan script",
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
func (r *ScanSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig validates the resource configuration. Mirrors
// RotationTemplateResource.ValidateConfig's template_type cross-checks exactly - same
// three-way mode split, same mutual-exclusivity rules.
func (r *ScanSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ScanSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

// ModifyPlan handles script_file_hash computation and script_name prediction.
func (r *ScanSettingsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan ScanSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
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

	// Predict script_name to match what uploadScript will actually produce at apply time -
	// same "inconsistent result after apply" concern as rotation template's equivalent, but
	// simpler here: resource_type_id is user-supplied (not server-generated), so the
	// InlineFile auto-name is fully computable even on create, and Local mode explicitly
	// clears script_name (confirmed by capture the API honors an explicit ""), so there's
	// no need to carry forward prior state.
	switch strings.ToLower(plan.TemplateType.ValueString()) {
	case "filepath":
		if !plan.ScriptFilePath.IsNull() && plan.ScriptFilePath.ValueString() != "" {
			plan.ScriptName = types.StringValue(filepath.Base(plan.ScriptFilePath.ValueString()))
		} else {
			plan.ScriptName = types.StringNull()
		}
	case "inlinefile":
		plan.ScriptName = types.StringValue(lastPathSegment(plan.ResourceTypeID.ValueString()) + "_scan_file")
	default: // local
		plan.ScriptName = types.StringNull()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create creates the resource and sets the initial Terraform state.
func (r *ScanSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScanSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Created scan settings for resource type: %s", plan.ResourceTypeID.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ScanSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScanSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())

	log.Printf("[INFO] Reading scan settings for resource type: %s", resourceTypeID)

	settings, err := r.client.GetScanSettings(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Scan Settings", err.Error())
		return
	}

	// Unlike rotation templates, GetScanSettings never 404s - a never-configured resource
	// type returns 200 with {}. An empty ID means someone deleted the parent resource type
	// (or reset it out-of-band in some way this provider can't distinguish) - treat as gone.
	if settings.ID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := r.mapModelToResource(settings, &state, false); err != nil {
		resp.Diagnostics.AddError("Error Reading Scan Settings Script Content", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ScanSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScanSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Updated scan settings for resource type: %s", plan.ResourceTypeID.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete resets the scan settings to Local mode with default values. There's no DELETE
// endpoint (unconfirmed by any capture) and no evidence the singleton can be removed
// outright, so this is the closest equivalent to "un-configuring" it.
func (r *ScanSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScanSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID := lastPathSegment(state.ResourceTypeID.ValueString())

	log.Printf("[INFO] Resetting scan settings for resource type: %s", resourceTypeID)

	reset := britive.ScanSettings{
		IsLocal:      true,
		InlineFile:   false,
		EditorType:   "text",
		ScriptName:   "",
		TimeoutLimit: scanSettingsDefaultTimeLimitMinutes * 60,
		Variables:    make([]britive.RotationTemplateVariable, 0),
	}

	if _, err := r.client.UpsertScanSettings(resourceTypeID, reset); err != nil {
		resp.Diagnostics.AddError("Error Resetting Scan Settings", err.Error())
		return
	}

	log.Printf("[INFO] Reset scan settings for resource type: %s", resourceTypeID)
}

// ImportState imports the resource state.
func (r *ScanSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceTypeID, err := parseScanSettingsCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	log.Printf("[INFO] Importing scan settings for resource type: %s", resourceTypeID)

	settings, err := r.client.GetScanSettings(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Scan Settings", err.Error())
		return
	}
	if settings.ID == "" {
		resp.Diagnostics.AddError("Scan Settings Not Found", fmt.Sprintf("No scan settings configured for resource type %s", resourceTypeID))
		return
	}

	var state ScanSettingsResourceModel
	state.ID = types.StringValue(scanSettingsCompositeID(resourceTypeID))
	state.ResourceTypeID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", resourceTypeID))

	if err := r.mapModelToResource(settings, &state, true); err != nil {
		resp.Diagnostics.AddError("Error Getting Scan Settings Script Content", err.Error())
		return
	}

	// script_file_path (FilePath mode) isn't recoverable from the API on import - only the
	// original local path was ever known, not by the provider. script_content (InlineFile
	// mode) IS recovered above via presigned download, so it's already populated correctly.

	log.Printf("[INFO] Imported scan settings for resource type: %s", resourceTypeID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// upsert builds the settings payload from plan, uploads the script if applicable, saves
// via the single PUT that backs both create and update, and reads back to populate
// computed fields. Shared by Create and Update since the API doesn't distinguish them.
func (r *ScanSettingsResource) upsert(ctx context.Context, plan *ScanSettingsResourceModel, diags *diag.Diagnostics) {
	resourceTypeID := lastPathSegment(plan.ResourceTypeID.ValueString())

	settings := r.buildUpsertPayload(plan)

	if err := r.uploadScript(resourceTypeID, plan, &settings); err != nil {
		diags.AddError("Error Uploading Scan Settings Script", err.Error())
		return
	}

	if _, err := r.client.UpsertScanSettings(resourceTypeID, settings); err != nil {
		diags.AddError("Error Saving Scan Settings", err.Error())
		return
	}

	plan.ID = types.StringValue(scanSettingsCompositeID(resourceTypeID))

	full, err := r.client.GetScanSettings(resourceTypeID)
	if err != nil {
		diags.AddError("Error Reading Scan Settings", err.Error())
		return
	}

	if err := r.mapModelToResource(full, plan, false); err != nil {
		diags.AddError("Error Reading Scan Settings Script Content", err.Error())
		return
	}
}

// buildUpsertPayload maps the plan into the API's PUT request shape, deriving
// isLocal/inlineFile/editorType from template_type. Does not set ScriptName (set by
// uploadScript, or explicitly cleared to "" for Local mode below).
func (r *ScanSettingsResource) buildUpsertPayload(plan *ScanSettingsResourceModel) britive.ScanSettings {
	settings := britive.ScanSettings{
		// The API's timeoutLimit is in seconds; time_limit is exposed to users in minutes,
		// same convention as rotation template's time_limit.
		TimeoutLimit: int(plan.TimeLimit.ValueInt64()) * 60,
		EditorType:   "text",
		Variables:    make([]britive.RotationTemplateVariable, 0, len(plan.Variables)),
	}

	for _, v := range plan.Variables {
		settings.Variables = append(settings.Variables, britive.RotationTemplateVariable{
			Name:        v.Name.ValueString(),
			Type:        canonicalVariableType(v.Type.ValueString()),
			MultiValued: v.MultiValued.ValueBool(),
		})
	}

	switch strings.ToLower(plan.TemplateType.ValueString()) {
	case "local":
		settings.IsLocal = true
		settings.InlineFile = false
		// Confirmed by capture: the API honors an explicit "" here, clearing any
		// previously-set script name - unlike rotation templates, no state carry-forward
		// workaround is needed.
		settings.ScriptName = ""
	case "inlinefile":
		settings.IsLocal = false
		settings.InlineFile = true
		settings.EditorType = strings.ToLower(plan.ScriptLanguage.ValueString())
	case "filepath":
		settings.IsLocal = false
		settings.InlineFile = false
	}

	return settings
}

// uploadScript uploads the script content/file for InlineFile/FilePath modes and stamps
// the resulting scriptName onto settings. A no-op for Local mode (scriptName was already
// explicitly cleared in buildUpsertPayload).
func (r *ScanSettingsResource) uploadScript(resourceTypeID string, plan *ScanSettingsResourceModel, settings *britive.ScanSettings) error {
	switch strings.ToLower(plan.TemplateType.ValueString()) {
	case "inlinefile":
		if err := r.client.UploadScanSettingsScriptCode(resourceTypeID, plan.ScriptContent.ValueString(), plan.ScriptLanguage.ValueString()); err != nil {
			return err
		}
		settings.ScriptName = resourceTypeID + "_scan_file"
	case "filepath":
		if err := r.client.UploadScanSettingsScriptFile(resourceTypeID, plan.ScriptFilePath.ValueString()); err != nil {
			return err
		}
		settings.ScriptName = filepath.Base(plan.ScriptFilePath.ValueString())
	}
	return nil
}

// mapModelToResource maps the API's scan settings detail onto Terraform state. Follows the
// same case-preservation and drift-detection approach as
// RotationTemplateResource.mapModelToResource - see its doc comment for the full rationale.
func (r *ScanSettingsResource) mapModelToResource(settings *britive.ScanSettings, state *ScanSettingsResourceModel, imported bool) error {
	// Converting back from the wire's seconds to the schema's minutes; see buildUpsertPayload.
	state.TimeLimit = types.Int64Value(int64(settings.TimeoutLimit) / 60)
	state.ScriptName = optionalStringValue(settings.ScriptName)

	computedTemplateType := "FilePath"
	switch {
	case settings.IsLocal:
		computedTemplateType = "Local"
	case settings.InlineFile:
		computedTemplateType = "InlineFile"
	}
	if !imported && strings.EqualFold(state.TemplateType.ValueString(), computedTemplateType) {
		// Prior state already matches (case-insensitively) - keep the user's casing.
	} else {
		state.TemplateType = types.StringValue(computedTemplateType)
	}

	computedLanguage := optionalStringValue(settings.EditorType)
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

	variables := make([]RotationTemplateVariableModel, 0, len(settings.Variables))
	for _, v := range settings.Variables {
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

	switch {
	case settings.IsLocal:
		state.ScriptContent = types.StringNull()
		state.ScriptFileHash = types.StringNull()

	case settings.InlineFile:
		if settings.PresignedURL != "" {
			content, err := r.client.DownloadPresignedContent(settings.PresignedURL)
			if err != nil {
				return err
			}
			state.ScriptContent = types.StringValue(content)
		}
		state.ScriptFileHash = types.StringNull()

	default: // FilePath
		state.ScriptContent = types.StringNull()
		if settings.PresignedURL != "" {
			content, err := r.client.DownloadPresignedContent(settings.PresignedURL)
			if err != nil {
				return err
			}
			state.ScriptFileHash = types.StringValue(hashBytes([]byte(content)))
		}
	}

	return nil
}

// scanSettingsCompositeID builds the resource's `id` value from resourceTypeID alone -
// unlike rotation templates, there's no server-generated component to wait for.
func scanSettingsCompositeID(resourceTypeID string) string {
	return fmt.Sprintf("resource-manager/resource-types/%s/scan-settings", resourceTypeID)
}

// parseScanSettingsCompositeID extracts resourceTypeID from either the composite ID
// ("resource-manager/resource-types/{resource_type_id}/scan-settings") or a bare
// resource_type_id, for use on import.
func parseScanSettingsCompositeID(id string) (resourceTypeID string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 4:
		return parts[2], nil
	case 1:
		return parts[0], nil
	default:
		return "", errs.NewInvalidResourceIDError("scan settings", id)
	}
}
