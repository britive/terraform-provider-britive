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
	_ resource.Resource                = &ResourceProfileAdditionalSettings{}
	_ resource.ResourceWithConfigure   = &ResourceProfileAdditionalSettings{}
	_ resource.ResourceWithImportState = &ResourceProfileAdditionalSettings{}
)

type ResourceProfileAdditionalSettings struct {
	client       *britive_client.Client
	helper       *ResourceProfileAdditionalSettingsHelper
	importHelper *imports.ImportHelper
}

type ResourceProfileAdditionalSettingsHelper struct{}

func NewResourceProfileAdditionalSettings() resource.Resource {
	return &ResourceProfileAdditionalSettings{}
}

func NewResourceProfileAdditionalSettingsHelper() *ResourceProfileAdditionalSettingsHelper {
	return &ResourceProfileAdditionalSettingsHelper{}
}

func (rpas *ResourceProfileAdditionalSettings) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_profile_additional_settings"
}

func (rpas *ResourceProfileAdditionalSettings) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Profile Additional Settings resource")

	if req.ProviderData == nil {
		return
	}

	rpas.client = req.ProviderData.(*britive_client.Client)
	if rpas.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Profile Additional Settings")
	rpas.helper = NewResourceProfileAdditionalSettingsHelper()
}

func (rpas *ResourceProfileAdditionalSettings) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for profile additional settings resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
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
			"use_app_credential_type": schema.BoolAttribute{
				Optional:    true,
				Description: "Inherit the credential type settings from the application",
			},
			"console_access": schema.BoolAttribute{
				Optional:    true,
				Description: "Provide the console access for the profile, overriden if use_app_credential_type is set to true",
			},
			"programmatic_access": schema.BoolAttribute{
				Optional:    true,
				Description: "Provide the programmatic access for the profile, overriden if use_app_credential_type is set to true",
			},
			"project_id_for_service_account": schema.StringAttribute{
				Optional:    true,
				Description: "The project id for creating service accounts",
			},
		},
	}
}

func (rpas *ResourceProfileAdditionalSettings) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_profile_permission")

	var plan britive_client.ProfileAdditionalSettingsPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during profile_permission creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileAdditionalSettings := britive_client.ProfileAdditionalSettings{}

	err := rpas.helper.mapResourceToModel(plan, &profileAdditionalSettings, false)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile additional settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map profile additional settings to model, %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new profile additional settings: %#v", profileAdditionalSettings))

	pas, err := rpas.client.UpdateProfileAdditionalSettings(ctx, profileAdditionalSettings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile additional settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create profile additional settings, %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new profile additional settings: %#v", pas))
	plan.ID = types.StringValue(rpas.helper.generateUniqueID(pas.ProfileID))

	planPtr, err := rpas.helper.getAndMapModelToPlan(ctx, plan, *rpas.client, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile additional settings",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile additional settings model to plan", map[string]interface{}{
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
		"profile_additional_settings": planPtr,
	})
}

func (rpas *ResourceProfileAdditionalSettings) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_profile_additional_settings")

	if rpas.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ProfileAdditionalSettingsPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get profile additional settings state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rpas.helper.getAndMapModelToPlan(ctx, state, *rpas.client, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile additional settings",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile additional settings model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Fetched profile additional settings:  %s", planPtr.ProfileID.ValueString()))
}

func (rpas *ResourceProfileAdditionalSettings) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_profile_additional_settings")

	var plan, state britive_client.ProfileAdditionalSettingsPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	var hasChanges bool
	if !plan.ProfileID.Equal(state.ProfileID) || !plan.UserAppCredentialType.Equal(state.UserAppCredentialType) || !plan.ConsoleAccess.Equal(state.ConsoleAccess) || !plan.ProgrammaticAccess.Equal(state.ProgrammaticAccess) || !plan.ProjectIDForServiceAccount.Equal(state.ProjectIDForServiceAccount) {
		hasChanges = true
		profileID, err := rpas.helper.parseUniqueID(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to update orofile additional settings", "Unable to fetch profile additional settings ID")
			tflog.Error(ctx, fmt.Sprintf("Failed to parse ID, %#v", err))
			return
		}

		profileAdditionalSettings := britive_client.ProfileAdditionalSettings{}

		err = rpas.helper.mapResourceToModel(plan, &profileAdditionalSettings, false)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update profile additional settings", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map resource to model, %#v", err))
			return
		}

		profileAdditionalSettings.ProfileID = profileID

		upas, err := rpas.client.UpdateProfileAdditionalSettings(ctx, profileAdditionalSettings)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update profile additional settings", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update profile additional settings: %#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated profile additional settings: %#v", upas))
		plan.ID = types.StringValue(rpas.helper.generateUniqueID(profileID))
	}
	if hasChanges {
		planPtr, err := rpas.helper.getAndMapModelToPlan(ctx, plan, *rpas.client, false)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get profile additional settings",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map profile additional settings model to plan", map[string]interface{}{
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

		tflog.Info(ctx, fmt.Sprintf("Updated profile additional settings:  %s", planPtr.ProfileID.ValueString()))
	}
}

func (rpas *ResourceProfileAdditionalSettings) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_profile_additional_settings")

	var state britive_client.ProfileAdditionalSettingsPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID, err := rpas.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete profile additional settings", "Unable to fetch profile additional settings ID")
		tflog.Error(ctx, fmt.Sprintf("Unable to fetch profile additional settings ID: %#v", err))
		return
	}

	profileAdditionalSettings := britive_client.ProfileAdditionalSettings{}

	err = rpas.helper.mapResourceToModel(state, &profileAdditionalSettings, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete profile additional settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete profile additional settings: %#v", err))
		return
	}

	profileAdditionalSettings.ProfileID = profileID

	tflog.Info(ctx, fmt.Sprintf("Deleting profile additional settings for %s", profileID))

	_, err = rpas.client.UpdateProfileAdditionalSettings(ctx, profileAdditionalSettings)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update profile additional settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to update profile additional settings: %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleted profile additional settings for %s", profileID))
	resp.State.RemoveResource(ctx)
}

func (rpas *ResourceProfileAdditionalSettings) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}
	if err := rpas.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/additional-settings"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import profile additional settings", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import profile additional settings: %#v", err))
		return
	}

	profileID := importData.Fields["profile_id"]
	if strings.TrimSpace(profileID) == "" {
		resp.Diagnostics.AddError("Failed to import profile additional settings", "ProfileID not found")
		tflog.Error(ctx, "ProfileID not found")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing profile additional settings for %s", profileID))

	plan := &britive_client.ProfileAdditionalSettingsPlan{
		ID: types.StringValue(rpas.helper.generateUniqueID(profileID)),
	}
	planPtr, err := rpas.helper.getAndMapModelToPlan(ctx, *plan, *rpas.client, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import profile additional settings",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import profile additional settings model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported profile additional settings for %s", profileID))
}

func (rpash *ResourceProfileAdditionalSettingsHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ProfileAdditionalSettingsPlan, c britive_client.Client, isImport bool) (*britive_client.ProfileAdditionalSettingsPlan, error) {
	profileID, err := rpash.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading profile additional settings: %s", profileID))

	profileAdditionalSettings, err := c.GetProfileAdditionalSettings(ctx, profileID)
	if err != nil {
		return nil, err
	}

	plan.ProfileID = types.StringValue(profileID)
	if (!plan.ConsoleAccess.IsNull() && !plan.ConsoleAccess.IsUnknown()) || isImport {
		plan.ConsoleAccess =
			types.BoolValue(profileAdditionalSettings.ConsoleAccess)
	}
	if (!plan.UserAppCredentialType.IsNull() && !plan.UserAppCredentialType.IsUnknown()) || isImport {
		plan.UserAppCredentialType =
			types.BoolValue(profileAdditionalSettings.UseApplicationCredentialType)
	}
	if (!plan.ProgrammaticAccess.IsNull() && !plan.ProgrammaticAccess.IsUnknown()) || isImport {
		plan.ProgrammaticAccess =
			types.BoolValue(profileAdditionalSettings.ProgrammaticAccess)
	}

	if (!plan.ProjectIDForServiceAccount.IsNull() && !plan.ProjectIDForServiceAccount.IsUnknown()) || isImport {
		plan.ProjectIDForServiceAccount = types.StringValue(profileAdditionalSettings.ProjectIdForServiceAccount)
	}

	return &plan, nil
}

func (rpash *ResourceProfileAdditionalSettingsHelper) mapResourceToModel(plan britive_client.ProfileAdditionalSettingsPlan, profileAdditionalSettings *britive_client.ProfileAdditionalSettings, isDelete bool) error {
	profileAdditionalSettings.ProfileID = plan.ProfileID.ValueString()
	var isProjectIdSet bool
	if !plan.ProjectIDForServiceAccount.IsNull() && !plan.ProjectIDForServiceAccount.IsUnknown() {
		isProjectIdSet = true
	}

	if isDelete {
		profileAdditionalSettings.UseApplicationCredentialType = true
		profileAdditionalSettings.ConsoleAccess = false
		profileAdditionalSettings.ProgrammaticAccess = false
		if isProjectIdSet == true {
			profileAdditionalSettings.ProjectIdForServiceAccount = ""
		}
	} else {
		profileAdditionalSettings.UseApplicationCredentialType = plan.UserAppCredentialType.ValueBool()
		profileAdditionalSettings.ConsoleAccess = plan.ConsoleAccess.ValueBool()
		profileAdditionalSettings.ProgrammaticAccess = plan.ProgrammaticAccess.ValueBool()
		if isProjectIdSet == true {
			profileAdditionalSettings.ProjectIdForServiceAccount = plan.ProjectIDForServiceAccount.ValueString()
		}
	}

	return nil
}

func (resourceProfileAdditionalSettingsHelper *ResourceProfileAdditionalSettingsHelper) generateUniqueID(profileID string) string {
	return fmt.Sprintf("paps/%s/additional-settings", profileID)
}

func (resourceProfileAdditionalSettingsHelper *ResourceProfileAdditionalSettingsHelper) parseUniqueID(ID string) (profileID string, err error) {
	profileAdditionalSettings := strings.Split(ID, "/")
	if len(profileAdditionalSettings) < 3 {
		err = errs.NewInvalidResourceIDError("profile additional settings", ID)
		return
	}

	profileID = profileAdditionalSettings[1]
	return
}
