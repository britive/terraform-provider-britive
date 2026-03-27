package resourcemanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
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

// ResourceTypePermissionsResource is the resource implementation.
type ResourceTypePermissionsResource struct {
	client *britive.Client
}

// ResourceTypePermissionsResourceModel describes the resource data model.
type ResourceTypePermissionsResourceModel struct {
	ID                    types.String   `tfsdk:"id"`
	PermissionID          types.String   `tfsdk:"permission_id"`
	Name                  types.String   `tfsdk:"name"`
	Version               types.String   `tfsdk:"version"`
	ResourceTypeID        types.String   `tfsdk:"resource_type_id"`
	Description           types.String   `tfsdk:"description"`
	CheckinTimeLimit      types.Int64    `tfsdk:"checkin_time_limit"`
	CheckoutTimeLimit     types.Int64    `tfsdk:"checkout_time_limit"`
	IsDraft               types.Bool     `tfsdk:"is_draft"`
	InlineFileExists      types.Bool     `tfsdk:"inline_file_exists"`
	ResponseTemplates     []types.String `tfsdk:"response_templates"`
	ShowOrigCreds         types.Bool     `tfsdk:"show_orig_creds"`
	CheckinCodeFile       types.String   `tfsdk:"checkin_code_file"`
	CheckoutCodeFile      types.String   `tfsdk:"checkout_code_file"`
	CheckinCodeFileHash   types.String   `tfsdk:"checkin_code_file_hash"`
	CheckoutCodeFileHash  types.String   `tfsdk:"checkout_code_file_hash"`
	CheckinCode           types.String   `tfsdk:"checkin_code"`
	CheckoutCode          types.String   `tfsdk:"checkout_code"`
	CodeLanguage          types.String   `tfsdk:"code_language"`
	CheckinFileName       types.String   `tfsdk:"checkin_file_name"`
	CheckoutFileName      types.String   `tfsdk:"checkout_file_name"`
	Variables             []types.String `tfsdk:"variables"`
}

// NewResourceTypePermissionsResource is a helper function to simplify the provider implementation.
func NewResourceTypePermissionsResource() resource.Resource {
	return &ResourceTypePermissionsResource{}
}

// Metadata returns the resource type name.
func (r *ResourceTypePermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_type_permission"
}

// Schema defines the schema for the resource.
func (r *ResourceTypePermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager resource type permission",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The permission ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permission_id": schema.StringAttribute{
				Description: "The ID of the permission",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the permission",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The version for the permission",
				Computed:    true,
			},
			"resource_type_id": schema.StringAttribute{
				Description: "The ID of the associated resource type",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the permission",
				Optional:    true,
			},
			"checkin_time_limit": schema.Int64Attribute{
				Description: "The check-in time limit in minutes",
				Optional:    true,
				Computed:    true,
			},
			"checkout_time_limit": schema.Int64Attribute{
				Description: "The check-out time limit in minutes",
				Optional:    true,
				Computed:    true,
			},
			"is_draft": schema.BoolAttribute{
				Description: "Indicates if the permission is a draft",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"inline_file_exists": schema.BoolAttribute{
				Description: "Indicates if an inline file exists",
				Computed:    true,
			},
			"response_templates": schema.SetAttribute{
				Description: "List of response template names",
				Optional:    true,
				ElementType: types.StringType,
			},
			"show_orig_creds": schema.BoolAttribute{
				Description: "Indicates if original credentials should be shown",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"checkin_code_file": schema.StringAttribute{
				Description: "The file path for check-in code",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code"),
						path.MatchRoot("checkout_code"),
						path.MatchRoot("code_language"),
					),
				},
			},
			"checkout_code_file": schema.StringAttribute{
				Description: "The file path for check-out code",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code"),
						path.MatchRoot("checkout_code"),
						path.MatchRoot("code_language"),
					),
				},
			},
			"checkin_code_file_hash": schema.StringAttribute{
				Description: "The file hash for check-in code",
				Computed:    true,
			},
			"checkout_code_file_hash": schema.StringAttribute{
				Description: "The file hash for check-out code",
				Computed:    true,
			},
			"checkin_code": schema.StringAttribute{
				Description: "The inline check-in code",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code_file"),
						path.MatchRoot("checkout_code_file"),
					),
				},
			},
			"checkout_code": schema.StringAttribute{
				Description: "The inline check-out code",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code_file"),
						path.MatchRoot("checkout_code_file"),
					),
				},
			},
			"code_language": schema.StringAttribute{
				Description: "The inline code language (Text, Batch, Node, PowerShell, Python, Shell)",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Text"),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Text", "Batch", "Node", "PowerShell", "Python", "Shell"),
				},
			},
			"checkin_file_name": schema.StringAttribute{
				Description: "The name of the check-in file",
				Computed:    true,
			},
			"checkout_file_name": schema.StringAttribute{
				Description: "The name of the check-out file",
				Computed:    true,
			},
			"variables": schema.SetAttribute{
				Description: "List of variables",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ResourceTypePermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ResourceTypePermissionsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ResourceTypePermissionsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate response_templates when show_orig_creds is false
	if !data.ShowOrigCreds.IsNull() && !data.ShowOrigCreds.ValueBool() {
		if len(data.ResponseTemplates) == 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("show_orig_creds"),
				"Invalid Configuration",
				"'show_orig_creds' can be set to false only if response_templates are available",
			)
		}
	}

	// Validate file paths must be set together
	hasCheckinFile := !data.CheckinCodeFile.IsNull() && data.CheckinCodeFile.ValueString() != ""
	hasCheckoutFile := !data.CheckoutCodeFile.IsNull() && data.CheckoutCodeFile.ValueString() != ""
	if hasCheckinFile != hasCheckoutFile {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'checkin_code_file' and 'checkout_code_file' must be set together or left unset together",
		)
	}

	// Validate inline code must be set together
	hasCheckinCode := !data.CheckinCode.IsNull() && data.CheckinCode.ValueString() != ""
	hasCheckoutCode := !data.CheckoutCode.IsNull() && data.CheckoutCode.ValueString() != ""
	if hasCheckinCode != hasCheckoutCode {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'checkin_code' and 'checkout_code' must be set together or left unset together",
		)
	}
}

// ModifyPlan handles plan modification for file hash computation.
func (r *ResourceTypePermissionsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only compute hashes during planning
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan ResourceTypePermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ResourceTypePermissionsResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Compute file hashes if files are specified; set to empty when not using files
	if !plan.CheckinCodeFile.IsNull() && plan.CheckinCodeFile.ValueString() != "" {
		newHash, err := hashFileContent(plan.CheckinCodeFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Hashing Check-in File", err.Error())
			return
		}

		// Update hash if changed
		if plan.CheckinCodeFileHash.IsNull() || plan.CheckinCodeFileHash.ValueString() != newHash {
			plan.CheckinCodeFileHash = types.StringValue(newHash)
		}
	} else if plan.CheckinCodeFileHash.IsUnknown() {
		// Not using file upload; hash is not applicable
		plan.CheckinCodeFileHash = types.StringValue("")
	}

	if !plan.CheckoutCodeFile.IsNull() && plan.CheckoutCodeFile.ValueString() != "" {
		newHash, err := hashFileContent(plan.CheckoutCodeFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error Hashing Check-out File", err.Error())
			return
		}

		// Update hash if changed
		if plan.CheckoutCodeFileHash.IsNull() || plan.CheckoutCodeFileHash.ValueString() != newHash {
			plan.CheckoutCodeFileHash = types.StringValue(newHash)
		}
	} else if plan.CheckoutCodeFileHash.IsUnknown() {
		// Not using file upload; hash is not applicable
		plan.CheckoutCodeFileHash = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create creates the resource and sets the initial Terraform state.
func (r *ResourceTypePermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceTypePermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map resource to API model
	permission, err := r.mapResourceToModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Permission", err.Error())
		return
	}

	log.Printf("[INFO] Creating resource type permission draft: %#v", permission)

	// Create draft permission
	createdPerm, err := r.client.CreateResourceTypePermission(*permission)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Permission", err.Error())
		return
	}

	permission.PermissionID = createdPerm.PermissionID
	log.Printf("[INFO] Finalizing resource type permission: %s", permission.PermissionID)

	// Upload files or code
	if !plan.CheckinCodeFile.IsNull() && plan.CheckinCodeFile.ValueString() != "" {
		checkInFilePath := plan.CheckinCodeFile.ValueString()
		checkOutFilePath := plan.CheckoutCodeFile.ValueString()

		err = r.client.UploadPermissionFiles(permission.PermissionID, checkInFilePath, checkOutFilePath)
		if err != nil {
			resp.Diagnostics.AddError("Error Uploading Permission Files", err.Error())
			return
		}

		permission.CheckinFileName = filepath.Base(checkInFilePath)
		permission.CheckoutFileName = filepath.Base(checkOutFilePath)
	} else if !plan.CheckinCode.IsNull() && plan.CheckinCode.ValueString() != "" {
		checkInCode := plan.CheckinCode.ValueString()
		checkOutCode := plan.CheckoutCode.ValueString()
		codeLanguage := plan.CodeLanguage.ValueString()

		err = r.client.UploadPermissionCodes(permission.PermissionID, checkInCode, checkOutCode, codeLanguage)
		if err != nil {
			resp.Diagnostics.AddError("Error Uploading Permission Codes", err.Error())
			return
		}

		permission.CheckinFileName = permission.PermissionID + "_latest_checkin"
		permission.CheckoutFileName = permission.PermissionID + "_latest_checkout"
		permission.InlineFileExists = true
	}

	// Update permission with file names
	_, err = r.client.UpdateResourceTypePermission(*permission)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Permission", err.Error())
		return
	}

	// Set ID
	plan.ID = types.StringValue(createdPerm.PermissionID)
	plan.PermissionID = types.StringValue(createdPerm.PermissionID)

	// Read back to get computed values
	permission, err = r.client.GetResourceTypePermission(createdPerm.PermissionID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Permission", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(permission, &plan)

	log.Printf("[INFO] Created resource type permission: %s", createdPerm.PermissionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ResourceTypePermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceTypePermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID := state.ID.ValueString()

	log.Printf("[INFO] Reading resource type permission: %s", permissionID)

	permission, err := r.client.GetResourceTypePermission(permissionID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Permission", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(permission, &state)

	log.Printf("[INFO] Retrieved resource type permission: %s", permissionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ResourceTypePermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceTypePermissionsResourceModel
	var state ResourceTypePermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID := state.ID.ValueString()

	// Map resource to API model
	permission, err := r.mapResourceToModel(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Permission", err.Error())
		return
	}
	permission.PermissionID = permissionID

	// Upload files or code
	if !plan.CheckinCodeFile.IsNull() && plan.CheckinCodeFile.ValueString() != "" {
		checkInFilePath := plan.CheckinCodeFile.ValueString()
		checkOutFilePath := plan.CheckoutCodeFile.ValueString()

		err = r.client.UploadPermissionFiles(permission.PermissionID, checkInFilePath, checkOutFilePath)
		if err != nil {
			resp.Diagnostics.AddError("Error Uploading Permission Files", err.Error())
			return
		}

		permission.CheckinFileName = filepath.Base(checkInFilePath)
		permission.CheckoutFileName = filepath.Base(checkOutFilePath)
	} else if !plan.CheckinCode.IsNull() && plan.CheckinCode.ValueString() != "" {
		checkInCode := plan.CheckinCode.ValueString()
		checkOutCode := plan.CheckoutCode.ValueString()
		codeLanguage := plan.CodeLanguage.ValueString()

		err = r.client.UploadPermissionCodes(permission.PermissionID, checkInCode, checkOutCode, codeLanguage)
		if err != nil {
			resp.Diagnostics.AddError("Error Uploading Permission Codes", err.Error())
			return
		}

		permission.CheckinFileName = "test_123_checkin"
		permission.CheckoutFileName = "test_123_checkout"
		permission.InlineFileExists = true
	}

	log.Printf("[INFO] Updating resource type permission: %s", permissionID)

	_, err = r.client.UpdateResourceTypePermission(*permission)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Permission", err.Error())
		return
	}

	log.Printf("[INFO] Updated resource type permission: %s", permissionID)

	// Read back to get updated values
	permission, err = r.client.GetResourceTypePermission(permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Permission", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(permission, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ResourceTypePermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceTypePermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID := state.ID.ValueString()

	log.Printf("[INFO] Deleting resource type permission: %s", permissionID)

	err := r.client.DeleteResourceTypePermission(permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Permission", err.Error())
		return
	}

	log.Printf("[INFO] Deleted resource type permission: %s", permissionID)
}

// ImportState imports the resource state.
func (r *ResourceTypePermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two formats:
	// 1. resource-manager/permissions/{id}
	// 2. {id}

	importID := req.ID
	var permissionID string

	if strings.HasPrefix(importID, "resource-manager/permissions/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Import ID must be in format 'resource-manager/permissions/{id}' or '{id}', got: %s", importID),
			)
			return
		}
		permissionID = parts[2]
	} else {
		permissionID = importID
	}

	if strings.TrimSpace(permissionID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Permission ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing resource type permission: %s", permissionID)

	permission, err := r.client.GetResourceTypePermission(permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Permission", err.Error())
		return
	}

	// Set the state
	var state ResourceTypePermissionsResourceModel
	state.ID = types.StringValue(permission.PermissionID)

	// Map model to resource
	r.mapModelToResource(permission, &state)

	log.Printf("[INFO] Imported resource type permission: %s", permissionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ResourceTypePermissionsResource) mapResourceToModel(plan *ResourceTypePermissionsResourceModel) (*britive.ResourceTypePermission, error) {
	// Apply defaults for time limits
	checkinTimeLimit := int64(60)
	if !plan.CheckinTimeLimit.IsNull() {
		checkinTimeLimit = plan.CheckinTimeLimit.ValueInt64()
	}

	checkoutTimeLimit := int64(60)
	if !plan.CheckoutTimeLimit.IsNull() {
		checkoutTimeLimit = plan.CheckoutTimeLimit.ValueInt64()
	}

	permission := &britive.ResourceTypePermission{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		IsDraft:     plan.IsDraft.ValueBool(),
		CheckinTimeLimit: int(checkinTimeLimit),
		CheckoutTimeLimit: int(checkoutTimeLimit),
		ShowOrigCreds: plan.ShowOrigCreds.ValueBool(),
		Variables:     make([]interface{}, 0),
		ResponseTemplates: make([]interface{}, 0),
	}

	// Extract resource type ID from potential composite ID
	rawResourceTypeID := plan.ResourceTypeID.ValueString()
	parts := strings.Split(rawResourceTypeID, "/")
	permission.ResourceTypeID = parts[len(parts)-1]

	// Handle variables
	for _, v := range plan.Variables {
		permission.Variables = append(permission.Variables, v.ValueString())
	}

	// Handle response templates - need to map names to IDs
	if len(plan.ResponseTemplates) > 0 {
		allResponseTemplates, err := r.client.GetAllResponseTemplate()
		if err != nil {
			return nil, fmt.Errorf("unable to find response templates: %w", err)
		}

		mapResponseTemplates := make(map[string]string)
		for _, resp := range allResponseTemplates {
			mapResponseTemplates[resp.Name] = resp.TemplateID
		}

		for _, templateName := range plan.ResponseTemplates {
			name := templateName.ValueString()
			if tempID, ok := mapResponseTemplates[name]; !ok {
				return nil, errs.NewNotSupportedError(name + " Response Template")
			} else {
				respTemp := map[string]string{
					"templateId": tempID,
					"name":       name,
				}
				permission.ResponseTemplates = append(permission.ResponseTemplates, respTemp)
			}
		}
	}

	return permission, nil
}

func (r *ResourceTypePermissionsResource) mapModelToResource(permission *britive.ResourceTypePermission, state *ResourceTypePermissionsResourceModel) {
	state.PermissionID = types.StringValue(permission.PermissionID)
	state.Name = types.StringValue(permission.Name)
	state.Description = types.StringValue(permission.Description)
	state.Version = types.StringValue(permission.Version)
	state.IsDraft = types.BoolValue(permission.IsDraft)
	state.CheckinTimeLimit = types.Int64Value(int64(permission.CheckinTimeLimit))
	state.CheckoutTimeLimit = types.Int64Value(int64(permission.CheckoutTimeLimit))
	state.InlineFileExists = types.BoolValue(permission.InlineFileExists)
	state.ShowOrigCreds = types.BoolValue(permission.ShowOrigCreds)
	state.CheckinFileName = types.StringValue(permission.CheckinFileName)
	state.CheckoutFileName = types.StringValue(permission.CheckoutFileName)

	// Map response templates
	var templateNames []types.String
	for _, rt := range permission.ResponseTemplates {
		if rtMap, ok := rt.(map[string]interface{}); ok {
			if name, ok := rtMap["name"].(string); ok {
				templateNames = append(templateNames, types.StringValue(name))
			}
		}
	}
	state.ResponseTemplates = templateNames
}

// hashFileContent computes SHA256 hash of file content
func hashFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}
