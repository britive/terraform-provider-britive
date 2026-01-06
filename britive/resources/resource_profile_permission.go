package resources

import (
	"context"
	"fmt"
	"regexp"
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
	_ resource.Resource                = &ResourceProfilePermission{}
	_ resource.ResourceWithConfigure   = &ResourceProfilePermission{}
	_ resource.ResourceWithImportState = &ResourceProfilePermission{}
)

type ResourceProfilePermission struct {
	client       *britive_client.Client
	helper       *ResourceProfilePermissionHelper
	importHelper *imports.ImportHelper
}

type ResourceProfilePermissionHelper struct{}

func NewResourceProfilePermission() resource.Resource {
	return &ResourceProfilePermission{}
}

func NewResourceProfilePermissionHelper() *ResourceProfilePermissionHelper {
	return &ResourceProfilePermissionHelper{}
}

func (rpp *ResourceProfilePermission) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_profile_permission"
}

func (rpp *ResourceProfilePermission) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Profile Permission resource")

	if req.ProviderData == nil {
		return
	}

	rpp.client = req.ProviderData.(*britive_client.Client)
	if rpp.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Profile Permission")
	rpp.helper = NewResourceProfilePermissionHelper()
}

func (rpp *ResourceProfilePermission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for profile permission resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of application, profile is associated with",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"profileId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"profile_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permission_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"profileId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"permission_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of the permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"profileId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
		},
	}
}

func (rpp *ResourceProfilePermission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_profile_permission")

	var plan britive_client.ProfilePermissionPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during profile_permission creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID := plan.ProfileID.ValueString()

	profilePermissionRequest := britive_client.ProfilePermissionRequest{
		Operation: "add",
		Permission: britive_client.ProfilePermission{
			ProfileID: profileID,
			Name:      plan.PermissionName.ValueString(),
			Type:      plan.PermissionType.ValueString(),
		},
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new profile permission: %#v", profilePermissionRequest))

	err := rpp.client.ExecuteProfilePermissionRequest(ctx, profileID, profilePermissionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create profile permission: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Created profile permission: %#v", profilePermissionRequest))

	plan.ID = types.StringValue(rpp.helper.generateUniqueID(profilePermissionRequest.Permission))

	planPtr, err := rpp.helper.getAndMapModelToPlan(ctx, plan, *rpp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get application",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile permission model to plan", map[string]interface{}{
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
		"profile_permission": planPtr,
	})

}

func (rpp *ResourceProfilePermission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_profile_permission")

	if rpp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ProfilePermissionPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get profile permission state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rpp.helper.getAndMapModelToPlan(ctx, state, *rpp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile permission",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile permission model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Fetched profile permission:  %s", planPtr.ProfileID.ValueString()))
}

func (rpp *ResourceProfilePermission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (rpp *ResourceProfilePermission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_profile_permission")

	var state britive_client.ProfilePermissionPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	profilePermission, err := rpp.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch profile permission", state.ID.ValueString())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile permission, %#v", err))
		return
	}
	profilePermissionRequest := britive_client.ProfilePermissionRequest{
		Operation:  "remove",
		Permission: *profilePermission,
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting profile permission: %s, %#v", profilePermission.ProfileID, profilePermissionRequest))

	err = rpp.client.ExecuteProfilePermissionRequest(ctx, profilePermission.ProfileID, profilePermissionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete profile permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete profile permission, %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("[INFO] Deleted profile permission: %s, %#v", profilePermission.ProfileID, profilePermissionRequest))

	resp.State.RemoveResource(ctx)
}

func (rpp *ResourceProfilePermission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}
	if err := rpp.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/permissions/(?P<permission_name>.+)/type/(?P<permission_type>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<permission_name>.+)/(?P<permission_type>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to parse import ID", err.Error())
		tflog.Error(ctx, "Failed to parse import ID", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	appName := importData.Fields["app_name"]
	profileName := importData.Fields["profile_name"]
	permissionName := importData.Fields["permission_name"]
	permissionType := importData.Fields["permission_type"]
	if strings.TrimSpace(appName) == "" {
		resp.Diagnostics.AddError("Failed to import profile permission", "Application not found")
		tflog.Error(ctx, "Failed to import profile permission, Application name is empty ('')")
		return
	}
	if strings.TrimSpace(profileName) == "" {
		resp.Diagnostics.AddError("Failed to import profile permission", "Profile not found")
		tflog.Error(ctx, "Failed to import profile permission, Profile Name is empty ('')")
		return
	}
	if strings.TrimSpace(permissionName) == "" {
		resp.Diagnostics.AddError("Failed to import profile permission", "Permission name not found")
		tflog.Error(ctx, "Failed to import profile permission, Permission Name is empty ('')")
		return
	}
	if strings.TrimSpace(permissionType) == "" {
		resp.Diagnostics.AddError("Failed to import profile permission", "Permission type not found")
		tflog.Error(ctx, "Failed to import profile permission, Permission type is empty ('')")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing profile permission: %s/%s/%s/%s", appName, profileName, permissionName, permissionType))

	app, err := rpp.client.GetApplicationByName(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile permission", fmt.Sprintf("Application with `%s` not found, error: %#v", appName, err))
		tflog.Error(ctx, fmt.Sprintf("Application not found during profile permission import, %#v", err))
		return
	}
	profile, err := rpp.client.GetProfileByName(ctx, app.AppContainerID, profileName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile permission", fmt.Sprintf("Profile with `%s` not found, error: %#v", profileName, err))
		tflog.Error(ctx, fmt.Sprintf("Profile not found during profile permission import, %#v", err))
		return
	}

	profilePermission, err := rpp.client.GetProfilePermission(profile.ProfileID, britive_client.ProfilePermission{Name: permissionName, Type: permissionType})
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile permission", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import profile permission, %#v", err))
		return
	}
	profilePermission.ProfileID = profile.ProfileID
	plan := &britive_client.ProfilePermissionPlan{
		ID:          types.StringValue(rpp.helper.generateUniqueID(*profilePermission)),
		ProfileID:   types.StringValue(profile.ProfileID),
		AppName:     types.StringValue(""),
		ProfileName: types.StringValue(""),
	}

	diags := resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state in Import", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Imported profile permission: %s/%s/%s/%s", appName, profileName, permissionName, permissionType))
}

func (rpph *ResourceProfilePermissionHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ProfilePermissionPlan, c britive_client.Client) (*britive_client.ProfilePermissionPlan, error) {
	profilePermission, err := rpph.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}
	pp, err := c.GetProfilePermission(profilePermission.ProfileID, *profilePermission)
	if err != nil {
		return nil, err
	}

	plan.ID = types.StringValue(rpph.generateUniqueID(*pp))
	plan.ProfileID = types.StringValue(profilePermission.ProfileID)
	plan.PermissionName = types.StringValue(profilePermission.Name)
	plan.PermissionType = types.StringValue(profilePermission.Type)
	plan.AppName = types.StringNull()
	plan.ProfileName = types.StringNull()
	return &plan, nil
}

func (rpph *ResourceProfilePermissionHelper) generateUniqueID(profilePermission britive_client.ProfilePermission) string {
	return fmt.Sprintf("paps/%s/permissions/%s/type/%s", profilePermission.ProfileID, profilePermission.Name, profilePermission.Type)
}

func (rpph *ResourceProfilePermissionHelper) parseUniqueID(ID string) (*britive_client.ProfilePermission, error) {
	idFormat := "paps/([^/]+)/permissions/(.+)/type/([^/]+)"

	re, err := regexp.Compile(idFormat)
	if err != nil {
		return nil, err
	}

	fieldValues := re.FindStringSubmatch(ID)
	if fieldValues != nil {
		profilePermission := &britive_client.ProfilePermission{
			ProfileID: fieldValues[1],
			Name:      fieldValues[2],
			Type:      fieldValues[3],
		}
		return profilePermission, nil
	} else {
		return nil, errs.NewInvalidResourceIDError("profile permission", ID)
	}
}
