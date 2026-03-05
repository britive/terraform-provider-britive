package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/utils"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

var (
	_ resource.Resource                   = &ResourceResourceManagerResourceTypePermission{}
	_ resource.ResourceWithConfigure      = &ResourceResourceManagerResourceTypePermission{}
	_ resource.ResourceWithImportState    = &ResourceResourceManagerResourceTypePermission{}
	_ resource.ResourceWithValidateConfig = &ResourceResourceManagerResourceTypePermission{}
	_ resource.ResourceWithModifyPlan     = &ResourceResourceManagerResourceTypePermission{}
)

const responseTemplate = "responseTemplate"
const variables = "variables"

type ResourceResourceManagerResourceTypePermission struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResourceTypePermissionHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResourceTypePermissionHelper struct{}

func NewResourceResourceManagerResourceTypePermission() resource.Resource {
	return &ResourceResourceManagerResourceTypePermission{}
}

func NewResourceResourceManagerResourceTypePermissionHelper() *ResourceResourceManagerResourceTypePermissionHelper {
	return &ResourceResourceManagerResourceTypePermissionHelper{}
}

func (rtp *ResourceResourceManagerResourceTypePermission) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_resource_type_permission"
}

func (rtp *ResourceResourceManagerResourceTypePermission) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring the Britive Resource Manager Resource Type Permission resource")

	if req.ProviderData == nil {
		return
	}

	rtp.client = req.ProviderData.(*britive_client.Client)
	if rtp.client == nil {
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
	rtp.helper = NewResourceResourceManagerResourceTypePermissionHelper()
}

func (rtp *ResourceResourceManagerResourceTypePermission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Resource Type Permission resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permission_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of resource type",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The version of permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of resource type",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive resource type permission",
			},
			"checkin_time_limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
				Description: "Checkin time limit minute",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"checkout_time_limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
				Description: "Checkout time limit minute",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"is_draft": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Indicates if permission is a draft",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"inline_file_exists": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates if an inline file exists.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"response_templates": schema.SetAttribute{
				Optional:    true,
				Description: "List of response template names.",
				ElementType: types.StringType,
			},
			"show_orig_creds": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Indicates if original credentials should be shown.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"checkin_code_file": schema.StringAttribute{
				Optional:    true,
				Description: "The file path for check-in code.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code"),
						path.MatchRoot("checkout_code"),
						path.MatchRoot("code_language"),
					),
				},
			},
			"checkout_code_file": schema.StringAttribute{
				Optional:    true,
				Description: "The file path for check-out code.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code"),
						path.MatchRoot("checkout_code"),
						path.MatchRoot("code_language"),
					),
				},
			},
			"checkin_code_file_hash": schema.StringAttribute{
				Computed:    true,
				Description: "The file hash for check-in code",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checkout_code_file_hash": schema.StringAttribute{
				Computed:    true,
				Description: "The file hash for check-out code.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checkin_code": schema.StringAttribute{
				Optional:    true,
				Description: "The inline check-in code.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code_file"),
						path.MatchRoot("checkout_code_file"),
					),
				},
			},
			"checkout_code": schema.StringAttribute{
				Optional:    true,
				Description: "Te inline check-out code.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("checkin_code_file"),
						path.MatchRoot("checkout_code_file"),
					),
				},
			},
			"code_language": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Text"),
				Description: "The inline code language. Select one of Test, Batch, Node, PoerShell, Python, Shell.",
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"Text",
						"Batch",
						"Node",
						"PowerShell",
						"Python",
						"Shell",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checkin_file_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the check-in file.",
			},
			"checkout_file_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the check-out file.",
			},
			"variables": schema.SetAttribute{
				Optional:    true,
				Description: "List of variables",
				ElementType: types.StringType,
			},
		},
	}
}

func (rtp *ResourceResourceManagerResourceTypePermission) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var config britive_client.ResourceManagerResourceTypePermissionPlan
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	responseTemplates, err := rtp.helper.mapSetToList(ctx, config, responseTemplate)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map response template to list", err.Error())
		tflog.Error(ctx, err.Error())
		return
	}
	// show_orig_creds validation
	if len(responseTemplates) == 0 &&
		!config.ShowOrigCreds.ValueBool() {

		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'show_orig_creds' can be set to false only if response templates are available",
		)
	}

	// checkin_code_file + checkout_code_file must be together
	if (config.CheckinCodeFile.ValueString() != "") !=
		(config.CheckoutCodeFile.ValueString() != "") {

		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'checkin_code_file' and 'checkout_code_file' must be set together or left unset together",
		)
	}

	// inline code must be together
	if (config.CheckinCode.ValueString() != "") !=
		(config.CheckoutCode.ValueString() != "") {

		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"'checkin_code' and 'checkout_code' must be set together or left unset together",
		)
	}
}

func (rtp *ResourceResourceManagerResourceTypePermission) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state *britive_client.ResourceManagerResourceTypePermissionPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// File hash calculation
	if !plan.CheckinCodeFile.IsNull() &&
		!plan.CheckoutCodeFile.IsNull() &&
		plan.CheckinCodeFile.ValueString() != "" &&
		plan.CheckoutCodeFile.ValueString() != "" {

		checkinHash, err := utils.HashFileContent(plan.CheckinCodeFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("File Hash Error", err.Error())
			return
		}

		checkoutHash, err := utils.HashFileContent(plan.CheckoutCodeFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("File Hash Error", err.Error())
			return
		}

		plan.CheckinCodeFileHash = types.StringValue(checkinHash)
		plan.CheckoutCodeFileHash = types.StringValue(checkoutHash)
		plan.InlineFileExists = types.BoolValue(false)

		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	} else {
		plan.CheckinCodeFileHash = types.StringNull()
		plan.CheckoutCodeFileHash = types.StringNull()
		plan.InlineFileExists = types.BoolValue(true)

		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}

	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Invalid configuration", "Failed to save chekin and checkout file hash")
		return
	}

	if state == nil || state.Name.IsNull() {
		return
	}

	oldVal := state.Name.ValueString()
	newVal := plan.Name.ValueString()
	if !plan.Name.Equal(state.Name) && oldVal != "" {
		resp.Diagnostics.AddError("invalid configuration", fmt.Sprintf("field %q is immutable and cannot be changed (from '%v' to '%v')", plan.Name.ValueString(), oldVal, newVal))
		tflog.Error(ctx, fmt.Sprintf("field %q is immutable and cannot be changed (from '%v' to '%v')", plan.Name.ValueString(), oldVal, newVal))
		return
	}

	oldVal = state.ResourceTypeID.ValueString()
	newVal = plan.ResourceTypeID.ValueString()
	if !plan.ResourceTypeID.Equal(state.ResourceTypeID) && oldVal != "" {
		resp.Diagnostics.AddError("Invalid configurations", fmt.Sprintf("field %q is immutable and cannot be changed (from '%v' to '%v')", plan.ResourceTypeID.ValueString(), oldVal, newVal))
		tflog.Error(ctx, fmt.Sprintf("field %q is immutable and cannot be changed (from '%v' to '%v')", plan.ResourceTypeID.ValueString(), oldVal, newVal))
		return
	}
}

func (rtp *ResourceResourceManagerResourceTypePermission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Called create for britive_resource_manager_resource_type_permission")

	var plan britive_client.ResourceManagerResourceTypePermissionPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during resource label creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	permission := &britive_client.ResourceTypePermission{}
	err := rtp.helper.mapResourceToModel(ctx, plan, permission, *rtp.client)
	if err != nil {
		diag.FromErr(err)
	}

	tflog.Info(ctx, fmt.Sprintf("Creating resource type permission draft: %#v", permission))

	// Create draft permission
	respTypePerm, err := rtp.client.CreateResourceTypePermission(ctx, *permission)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource type permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource type permission, error:%#v", err))
		return
	}

	// If is_draft is false, finalize the permission
	permission.PermissionID = respTypePerm.PermissionID
	tflog.Info(ctx, fmt.Sprintf("Finalizing resource type permission: %s", permission.PermissionID))

	//upload files or code
	var checkInFilePath, checkOutFilePath string
	if !plan.CheckinCodeFile.IsNull() && !plan.CheckinCodeFile.IsUnknown() {
		checkInFilePath = plan.CheckinCodeFile.ValueString()
	}
	if !plan.CheckoutCodeFile.IsNull() && !plan.CheckoutCodeFile.IsUnknown() {
		checkOutFilePath = plan.CheckoutCodeFile.ValueString()
	}

	if checkInFilePath != "" && checkOutFilePath != "" {
		err = rtp.client.UploadPermissionFiles(ctx, permission.PermissionID, checkInFilePath, checkOutFilePath)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create resource type permission", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to upload permision files while creating resource type permission, errro:%#v", err))
			return
		}
		permission.CheckinFileName = filepath.Base(checkInFilePath)
		permission.CheckoutFileName = filepath.Base(checkOutFilePath)
	}

	var checkInCode, checkOutCode, codeLanguage string
	if !plan.CheckinCode.IsNull() && !plan.CheckinCode.IsUnknown() {
		checkInCode = plan.CheckinCode.ValueString()
	}
	if !plan.CheckoutCode.IsNull() && !plan.CheckoutCode.IsUnknown() {
		checkOutCode = plan.CheckoutCode.ValueString()
	}
	if !plan.CodeLanguage.IsNull() && !plan.CodeLanguage.IsUnknown() {
		codeLanguage = plan.CodeLanguage.ValueString()
	}

	if checkInCode != "" && checkOutCode != "" {
		err = rtp.client.UploadPermissionCodes(ctx, permission.PermissionID, checkInCode, checkOutCode, codeLanguage)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create resource type permission", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to upload permission codes, error:%#v", err))
			return
		}
		permission.CheckinFileName = permission.PermissionID + "_latest_checkin"
		permission.CheckoutFileName = permission.PermissionID + "_latest_checkout"
		permission.InlineFileExists = true
	}

	_, err = rtp.client.UpdateResourceTypePermission(ctx, *permission)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource type permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource type permission, error:%#v", err))
		return
	}

	plan.ID = types.StringValue(respTypePerm.PermissionID)
	tflog.Info(ctx, fmt.Sprintf("Created resource type permission: %s", respTypePerm.PermissionID))

	planPtr, err := rtp.helper.getAndMapModelToPlan(ctx, plan, *rtp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource type permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource type permission model to plan", map[string]interface{}{
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
		"resource_type_permission": planPtr,
	})
}

func (rtp *ResourceResourceManagerResourceTypePermission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_resource_type_permission")

	if rtp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResourceTypePermissionPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get resource type permission state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := rtp.helper.getAndMapModelToPlan(ctx, state, *rtp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource type permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map resource type permission model to plan failed in Read", map[string]interface{}{
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

	tflog.Info(ctx, "Read completed for britive_resource_manager_resource_type_permission")
}

func (rtp *ResourceResourceManagerResourceTypePermission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_resource_manager_resource_type_permission")

	var plan, state britive_client.ResourceManagerResourceTypePermissionPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	permissionID := state.ID.ValueString()

	permission := &britive_client.ResourceTypePermission{}
	err := rtp.helper.mapResourceToModel(ctx, plan, permission, *rtp.client)
	if err != nil {
		diag.FromErr(err)
	}
	permission.PermissionID = permissionID

	//upload files or code
	var checkInFilePath, checkOutFilePath string
	if !plan.CheckinCodeFile.IsNull() && !plan.CheckinCodeFile.IsUnknown() {
		checkInFilePath = plan.CheckinCodeFile.ValueString()
	}
	if !plan.CheckoutCodeFile.IsNull() && !plan.CheckoutCodeFile.IsUnknown() {
		checkOutFilePath = plan.CheckoutCodeFile.ValueString()
	}
	if checkInFilePath != "" && checkOutFilePath != "" {
		err = rtp.client.UploadPermissionFiles(ctx, permission.PermissionID, checkInFilePath, checkOutFilePath)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource type permission", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource type permission, error: %#v", err))
			return
		}
		permission.CheckinFileName = filepath.Base(checkInFilePath)
		permission.CheckoutFileName = filepath.Base(checkOutFilePath)
	}

	var checkInCode, checkOutCode, codeLanguage string
	if !plan.CheckinCode.IsNull() && !plan.CheckinCode.IsUnknown() {
		checkInCode = plan.CheckinCode.ValueString()
	}
	if !plan.CheckoutCode.IsNull() && !plan.CheckoutCode.IsUnknown() {
		checkOutCode = plan.CheckoutCode.ValueString()
	}
	if !plan.CodeLanguage.IsNull() && !plan.CodeLanguage.IsUnknown() {
		codeLanguage = plan.CodeLanguage.ValueString()
	}
	if checkInCode != "" && checkOutCode != "" {
		err = rtp.client.UploadPermissionCodes(ctx, permission.PermissionID, checkInCode, checkOutCode, codeLanguage)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource type permission", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update resource type permission, error:%#v", err))
			return
		}
		permission.CheckinFileName = "test_123_checkin"
		permission.CheckoutFileName = "test_123_checkout"
		permission.InlineFileExists = true
	}

	tflog.Info(ctx, fmt.Sprintf("Updating resource type permission: %s", permissionID))

	_, err = rtp.client.UpdateResourceTypePermission(ctx, *permission)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource type permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to update resource type permission, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Updated resource type permission: %s", permissionID))
	plan.ID = types.StringValue(permissionID)

	planPtr, err := rtp.helper.getAndMapModelToPlan(ctx, plan, *rtp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource type permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource type permission model to plan", map[string]interface{}{
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

func (rtp *ResourceResourceManagerResourceTypePermission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_resource_type_permission")

	var state britive_client.ResourceManagerResourceTypePermissionPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	permissionID := state.ID.ValueString()

	tflog.Info(ctx, fmt.Sprintf("Deleting resource type permission: %s", permissionID))

	err := rtp.client.DeleteResourceTypePermission(ctx, permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource type permissio", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete resource type permission, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleted resource type permission: %s", permissionID))
	resp.State.RemoveResource(ctx)
}

func (rtp *ResourceResourceManagerResourceTypePermission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rtp.importHelper.ParseImportID([]string{"resource-manager/permissions/(?P<id>[^/]+)", "(?P<id>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import resource type permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, error%#v", err))
		return
	}
	permissionID := importData.Fields["id"]

	tflog.Info(ctx, fmt.Sprintf("Importing resource type permission: %s", permissionID))

	permission, err := rtp.client.GetResourceTypePermission(ctx, permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import resource type permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import resource type permission, error:%#v", err))
		return
	}

	plan := britive_client.ResourceManagerResourceTypePermissionPlan{
		ID:                types.StringValue(permission.PermissionID),
		ResponseTemplates: types.SetNull(types.StringType),
		Variables:         types.SetNull(types.StringType),
	}

	planPtr, err := rtp.helper.getAndMapModelToPlan(ctx, plan, *rtp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource type permission model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported resource type permission: %#v", planPtr))
}

func (rtph *ResourceResourceManagerResourceTypePermissionHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourceTypePermissionPlan, c britive_client.Client) (*britive_client.ResourceManagerResourceTypePermissionPlan, error) {
	permissionID := plan.ID.ValueString()

	permission, err := c.GetResourceTypePermission(ctx, permissionID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	plan.PermissionID = types.StringValue(permission.PermissionID)
	plan.Name = types.StringValue(permission.Name)
	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && permission.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(permission.Description)
	}

	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && permission.Description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(permission.Description)
	}

	if plan.ResourceTypeID.IsNull() || plan.ResourceTypeID.IsUnknown() {
		plan.ResourceTypeID = types.StringValue(permission.ResourceTypeID)
	}

	plan.IsDraft = types.BoolValue(permission.IsDraft)
	plan.CheckinTimeLimit = types.Int64Value(int64(permission.CheckinTimeLimit))
	plan.CheckoutTimeLimit = types.Int64Value(int64(permission.CheckoutTimeLimit))
	plan.InlineFileExists = types.BoolValue(permission.InlineFileExists)
	plan.ShowOrigCreds = types.BoolValue(permission.ShowOrigCreds)
	plan.CheckinFileName = types.StringValue(permission.CheckinFileName)
	plan.CheckoutFileName = types.StringValue(permission.CheckoutFileName)
	plan.Version = types.StringValue(permission.Version)
	if (plan.CheckinCodeFile.IsNull() || plan.CheckinCodeFile.IsUnknown()) && (plan.CheckoutCodeFile.IsNull() || plan.CheckoutCodeFile.IsUnknown()) {
		plan.CheckinCodeFileHash = types.StringNull()
		plan.CheckoutCodeFileHash = types.StringNull()
	} else {
		checkinHash, err := utils.HashFileContent(plan.CheckinCodeFile.ValueString())
		if err != nil {
			return nil, err
		}

		checkoutHash, err := utils.HashFileContent(plan.CheckoutCodeFile.ValueString())
		if err != nil {
			return nil, err
		}

		plan.CheckinCodeFileHash = types.StringValue(checkinHash)
		plan.CheckoutCodeFileHash = types.StringValue(checkoutHash)
	}

	var templateNames []string
	for _, rt := range permission.ResponseTemplates {
		if rtMap, ok := rt.(map[string]interface{}); ok {
			if name, ok := rtMap["name"].(string); ok {
				templateNames = append(templateNames, name)
			}
		}
	}
	if (plan.ResponseTemplates.IsNull() || plan.ResponseTemplates.IsUnknown()) && len(templateNames) == 0 {
		plan.ResponseTemplates = types.SetNull(types.StringType)
	} else {
		plan.ResponseTemplates, err = rtph.mapListToSet(ctx, templateNames)
		if err != nil {
			return nil, err
		}
	}

	var userVariables []string
	for _, rt := range permission.Variables {
		userVariables = append(userVariables, rt.(string))
	}
	if (plan.Variables.IsNull() || plan.Variables.IsUnknown()) && len(permission.Variables) == 0 {
		plan.Variables = types.SetNull(types.StringType)
	} else {
		plan.Variables, err = rtph.mapListToSet(ctx, userVariables)
		if err != nil {
			return nil, err
		}
	}

	return &plan, nil
}

func (rtph *ResourceResourceManagerResourceTypePermissionHelper) mapResourceToModel(ctx context.Context, plan britive_client.ResourceManagerResourceTypePermissionPlan, permission *britive_client.ResourceTypePermission, c britive_client.Client) error {
	allResponseTemplates, err := c.GetAllResponseTemplate(ctx)
	if err != nil {
		diag.FromErr(errs.NewNotSupportedError("Unable to find response templates :"))
	}

	if len(allResponseTemplates) == 0 {
		tflog.Warn(ctx, "No response templates returned from backend")
	}

	mapResponseTemplates := make(map[string]string)
	for _, resp := range allResponseTemplates {
		mapResponseTemplates[resp.Name] = resp.TemplateID
	}

	permission.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		permission.Description = plan.Description.ValueString()
	}

	rawResourceTypeID := strings.Split(plan.ResourceTypeID.ValueString(), "/")
	permission.ResourceTypeID = rawResourceTypeID[len(rawResourceTypeID)-1]

	permission.IsDraft = plan.IsDraft.ValueBool()
	if !plan.CheckinTimeLimit.IsNull() && !plan.CheckinTimeLimit.IsUnknown() {
		permission.CheckinTimeLimit = int(plan.CheckinTimeLimit.ValueInt64())
	}
	if !plan.CheckoutTimeLimit.IsNull() && !plan.CheckoutTimeLimit.IsUnknown() {
		permission.CheckoutTimeLimit = int(plan.CheckoutTimeLimit.ValueInt64())
	}

	if !plan.Variables.IsNull() && !plan.Variables.IsUnknown() {
		variables, err := rtph.mapSetToList(ctx, plan, variables)
		if err != nil {
			return err
		}

		var userVars []interface{}
		for _, v := range variables {
			userVars = append(userVars, v)
		}

		permission.Variables = append(permission.Variables, userVars...)
	}

	if !plan.ShowOrigCreds.IsNull() && !plan.ShowOrigCreds.IsUnknown() {
		permission.ShowOrigCreds = plan.ShowOrigCreds.ValueBool()
	}

	if !plan.ResponseTemplates.IsNull() && !plan.ResponseTemplates.IsUnknown() {
		responseTemplates, err := rtph.mapSetToList(ctx, plan, responseTemplate)
		if err != nil {
			return err
		}
		for _, val := range responseTemplates {
			if tempID, ok := mapResponseTemplates[val]; !ok {
				return errs.NewNotSupportedError(val + " Response Template")
			} else {
				respTemp := map[string]string{
					"templateId": tempID,
					"name":       val,
				}
				permission.ResponseTemplates = append(permission.ResponseTemplates, respTemp)
			}
		}
	}

	return nil
}

func (rtph *ResourceResourceManagerResourceTypePermissionHelper) mapSetToList(ctx context.Context, plan britive_client.ResourceManagerResourceTypePermissionPlan, field string) ([]string, error) {
	var set types.Set
	if strings.EqualFold(field, responseTemplate) {
		set = plan.ResponseTemplates
	} else if strings.EqualFold(field, variables) {
		set = plan.Variables
	} else {
		return nil, fmt.Errorf("failed to map response templates list")
	}
	var list []string

	diags := set.ElementsAs(ctx, &list, true)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to map response templates list")
	}

	return list, nil
}

func (rtph *ResourceResourceManagerResourceTypePermissionHelper) mapListToSet(ctx context.Context, list []string) (types.Set, error) {
	set, diags := types.SetValueFrom(
		ctx,
		types.StringType,
		list,
	)

	if diags.HasError() {
		return types.Set{}, fmt.Errorf("Failed to map response templates set")
	}

	return set, nil
}
