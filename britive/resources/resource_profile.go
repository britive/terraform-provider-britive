package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceProfile{}
	_ resource.ResourceWithConfigure   = &ResourceProfile{}
	_ resource.ResourceWithImportState = &ResourceProfile{}
)

type ResourceProfile struct {
	client       *britive_client.Client
	helper       *ResourceProfileHelper
	importHelper *imports.ImportHelper
}

type ResourceProfileHelper struct{}

func NewResourceProfile() resource.Resource {
	return &ResourceProfile{}
}

func NewResourceProfileHelper() *ResourceProfileHelper {
	return &ResourceProfileHelper{}
}

func (rp *ResourceProfile) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_profile"
}

func (rp *ResourceProfile) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Profile resource")

	if req.ProviderData == nil {
		return
	}

	rp.client = req.ProviderData.(*britive_client.Client)
	if rp.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for ResourceProfile")
	rp.helper = NewResourceProfileHelper()
}

func (rp *ResourceProfile) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Profile resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app_container_id": schema.StringAttribute{
				Required:    true,
				Description: "The identity of the Britive application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Britive profile",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive profile",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "To disable the Britive profile",
				Default:     booldefault.StaticBool(false),
			},
			"expiration_duration": schema.StringAttribute{
				Required:    true,
				Description: "The expiration time for the Britive profile",
			},
			"extendable": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Indicates whether profile expiry is extendable",
				Default:     booldefault.StaticBool(false),
			},
			"notification_prior_to_expiration": schema.StringAttribute{
				Optional:    true,
				Description: "The profile expiry notification as a time value",
			},
			"extension_duration": schema.StringAttribute{
				Optional:    true,
				Description: "The profile expiry extension as a time value",
			},
			"extension_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "The repetition limit for extending the profile expiry",
			},
			"destination_url": schema.StringAttribute{
				Optional:    true,
				Description: "The destination URL to redirect user after checkout",
			},
			"allow_impersonation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "TO enable or disable impersonation settings",
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				Description: "The set of associations for the Britive profile (order is ignored)",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of association, should be one of [Environment, EnvironmentGroup, ApplicationResource]",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The association value",
						},
						"parent_name": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The parent name of the resource. Required only if the association type is ApplicationResource",
						},
					},
				},
			},
		},
	}
}

func (rp *ResourceProfile) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_profile")

	var plan britive_client.ProfilePlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during profile creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})

		return
	}

	profile, err := rp.helper.mapPlanToModel(plan, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create profile",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Failed to map profilePlan to model", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	profile, err = rp.client.CreateProfile(ctx, profile.AppContainerID, *profile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create profile",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Failed to create profile", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	tflog.Info(ctx, "Profile creation succeeeded", map[string]interface{}{
		"profile_id": profile.ProfileID,
	})

	plan.ID = types.StringValue(profile.ProfileID)

	err = rp.helper.saveProfileAssociations(ctx, plan, profile.AppContainerID, profile.ProfileID, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to save  profile associations",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "Failed to save profile associations", map[string]interface{}{
			"error":      err.Error(),
			"profile_id": profile.ProfileID,
		})
	}

	planPtr, err := rp.helper.getAndMapModelToPlan(ctx, *rp.client, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after create", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Create completed and state set", map[string]interface{}{
		"profile": planPtr,
	})
	if resp.Diagnostics.HasError() {
		return
	}
}

func (rp *ResourceProfile) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_profile")

	if rp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ProfilePlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get profile state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID := state.ID.ValueString()
	if profileID == "" {
		resp.Diagnostics.AddError(
			"Failed to get profile",
			"Profile Id not found",
		)
		tflog.Error(ctx, "Read failed: missing profile ID in state")
		return
	}

	newPlan, err := rp.helper.getAndMapModelToPlan(ctx, *rp.client, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map profile model to plan failed in Read", map[string]interface{}{
			"error":      err.Error(),
			"profile_id": profileID,
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

	tflog.Info(ctx, "Read completed for britive_profile", map[string]interface{}{
		"profile_id": profileID,
	})
}

func (rp *ResourceProfile) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_profile")

	var plan, state britive_client.ProfilePlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID := state.ID.ValueString()
	appContainerID := state.AppContainerID.ValueString()

	profile, err := rp.helper.mapPlanToModel(plan, true)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping profile for update", err.Error())
		tflog.Error(ctx, "mapPlanToModel failed in Update", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	_, err = rp.client.UpdateProfile(ctx, appContainerID, profileID, *profile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update profile",
			fmt.Sprintf("Error: %v. Please check the input values and try again.", err),
		)
		tflog.Error(ctx, "UpdateProfile API call failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	tflog.Info(ctx, "UpdateProfile API call succeeded", map[string]interface{}{
		"profile_id": profileID,
	})

	if !plan.Disabled.Equal(state.Disabled) {
		disabled := plan.Disabled.ValueBool()
		_, err := rp.client.EnableOrDisableProfile(ctx, appContainerID, profileID, disabled)
		if err != nil {
			resp.Diagnostics.AddError("Error updating profile status", err.Error())
			tflog.Error(ctx, "EnableOrDisableProfile failed", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
	}

	err = rp.helper.saveProfileAssociations(ctx, plan, appContainerID, profileID, *rp.client)
	if err != nil {
		resp.Diagnostics.AddError("Error saving profile associations", err.Error())
		tflog.Error(ctx, "failed save profile associations in Update", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	plan.ID = types.StringValue(profileID)
	newPlan, err := rp.helper.getAndMapModelToPlan(ctx, *rp.client, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error refreshing updated profile state", err.Error())
		tflog.Error(ctx, "get and map profile model to plan failed after update", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newPlan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after Update", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed for britive_profile", map[string]interface{}{
		"profile": newPlan,
	})
}

func (rp *ResourceProfile) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_profile")

	var state britive_client.ProfilePlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID := state.ID.ValueString()
	appContainerID := state.AppContainerID.ValueString()

	err := rp.client.DeleteProfile(ctx, appContainerID, profileID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting profile",
			"Reason: "+err.Error(),
		)
		tflog.Error(ctx, "Delete Profile API call failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	tflog.Info(ctx, "Delete Profile API call succeeded", map[string]interface{}{
		"profile_id": profileID,
	})
	resp.State.RemoveResource(ctx)
}

func (rp *ResourceProfile) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rp.importHelper.ParseImportID(
		[]string{
			"apps/(?P<app_name>[^/]+)/paps/(?P<name>[^/]+)",
			"(?P<app_name>[^/]+)/(?P<name>[^/]+)",
		},
		importData,
	); err != nil {
		resp.Diagnostics.AddError("Failed to parse import ID", err.Error())
		tflog.Error(ctx, "Failed to parse import ID", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	appName := importData.Fields["app_name"]
	profName := importData.Fields["name"]

	app, err := rp.client.GetApplicationByName(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("App '%s' not found", appName), "Unable to find application")
		tflog.Error(ctx, fmt.Sprintf("App '%s' not found", appName))
		return
	}

	profile, err := rp.client.GetProfileByName(ctx, app.AppContainerID, profName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Profile '%s' not found", profName), "Unable to find profile")
		tflog.Error(ctx, fmt.Sprintf("Profile '%s' not found", profName))
		return
	}

	plan := &britive_client.ProfilePlan{
		ID: types.StringValue(profile.ProfileID),
	}

	diags := resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state in Import", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Import completed for britive_profile", map[string]interface{}{
		"profile_id": profile.ProfileID,
	})

}

func (rph *ResourceProfileHelper) getAndMapModelToPlan(ctx context.Context, client britive_client.Client, plan britive_client.ProfilePlan) (*britive_client.ProfilePlan, error) {
	profileID := plan.ID.ValueString()
	profile, err := client.GetProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}

	plan.ID = types.StringValue(profileID)
	plan.AppContainerID = types.StringValue(profile.AppContainerID)
	plan.Name = types.StringValue(profile.Name)
	plan.Description = types.StringValue(profile.Description)
	plan.AllowImpersonation = types.BoolValue(profile.AllowImpersonation)
	plan.Disabled = types.BoolValue(strings.EqualFold(profile.Status, "inactive"))
	plan.ExpirationDuration = types.StringValue(time.Duration(profile.ExpirationDuration * int64(time.Millisecond)).String())
	plan.Extendable = types.BoolValue(profile.Extendable)
	if profile.Extendable {
		if profile.NotificationPriorToExpiration != nil {
			plan.NotificationPriorToExpiration = types.StringValue(time.Duration(*profile.NotificationPriorToExpiration * int64(time.Millisecond)).String())
		}
		if profile.ExtensionDuration != nil {
			plan.ExtensionDuration = types.StringValue(time.Duration(*profile.ExtensionDuration * int64(time.Millisecond)).String())
		}
		plan.ExtensionLimit = types.Int64Value(profile.ExtensionLimit.(int64))
	}
	if profile.DestinationUrl == "" {
		plan.DestinationUrl = types.StringNull()
	} else {
		plan.DestinationUrl = types.StringValue(profile.DestinationUrl)
	}

	associations, err := rph.mapProfileAssociationsModelToResource(&plan, ctx, client, profile.AppContainerID, profile.ProfileID, profile.Associations)
	if err != nil {
		return nil, err
	}
	plan.Associations, err = rph.mapProfileAssociationToTypesSet(associations)
	if err != nil {
		return nil, err
	}

	return &plan, nil
}

func (rph *ResourceProfileHelper) mapProfileAssociationsModelToResource(plan *britive_client.ProfilePlan, ctx context.Context, client britive_client.Client, appContainerID string, profileID string, associations []britive_client.ProfileAssociation) ([]britive_client.ProfileAssociationPlan, error) {
	appRootEnvironmentGroup, err := client.GetApplicationRootEnvironmentGroup(ctx, appContainerID)
	if err != nil {
		return nil, err
	}
	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return make([]britive_client.ProfileAssociationPlan, 0), nil
	}
	inputAssociations, err := rph.mapSetToProfileAssociations(ctx, plan.Associations)
	if err != nil {
		return nil, err
	}
	applicationType, err := client.GetApplicationType(ctx, appContainerID)
	if err != nil {
		return nil, err
	}
	appType := applicationType.ApplicationType
	profileAssociations := make([]britive_client.ProfileAssociationPlan, 0)
	for _, association := range associations {
		var rootAssociations []britive_client.Association
		switch association.Type {
		case "EnvironmentGroup", "Environment":
			if association.Type == "EnvironmentGroup" {
				rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
			} else {
				rootAssociations = appRootEnvironmentGroup.Environments
			}
			var a *britive_client.Association
			for _, aeg := range rootAssociations {
				if aeg.ID == association.Value {
					a = &aeg
					break
				}
			}
			if a == nil {
				return nil, errs.NewNotFoundErrorf("association %s", association.Value)
			}
			var profileAssociation britive_client.ProfileAssociationPlan
			associationValue := a.Name
			for _, inputAssociation := range inputAssociations {
				iat := inputAssociation.Type.ValueString()
				iav := inputAssociation.Value.ValueString()
				if association.Type == "EnvironmentGroup" && (appType == "AWS" || appType == "AWS Standalone") && strings.EqualFold("root", a.Name) && strings.EqualFold("root", iav) {
					associationValue = iav
				}
				if association.Type == iat && a.ID == iav {
					associationValue = a.ID
					break
				} else if association.Type == "Environment" && appType == "AWS Standalone" {
					envId := client.GetEnvId(ctx, appContainerID, iav)
					if association.Type == iat && a.ID == envId {
						associationValue = iav
						break
					}
				}
			}
			profileAssociation.Type = types.StringValue(association.Type)
			profileAssociation.Value = types.StringValue(associationValue)
			profileAssociations = append(profileAssociations, profileAssociation)
		case "ApplicationResource":
			par, err := client.GetProfileAssociationResourceByNativeID(ctx, profileID, association.Value)
			if errors.Is(err, britive_client.ErrNotFound) {
				return nil, errs.NewNotFoundErrorf("application resource %s", association.Value)
			} else if err != nil {
				return nil, err
			} else if par == nil {
				return nil, errs.NewNotFoundErrorf("application resource %s", association.Value)
			}
			var profileAssociation britive_client.ProfileAssociationPlan
			profileAssociation.Type = types.StringValue(association.Type)
			profileAssociation.Value = types.StringValue(par.Name)
			if par.ParentName == "" {
				profileAssociation.ParentName = types.StringNull()
			} else {
				profileAssociation.ParentName = types.StringValue(par.ParentName)
			}

			profileAssociations = append(profileAssociations, profileAssociation)
		}

	}
	return profileAssociations, nil

}

func (rph *ResourceProfileHelper) mapSetToProfileAssociations(ctx context.Context, associationsSet types.Set) ([]britive_client.ProfileAssociationPlan, error) {
	var result []britive_client.ProfileAssociationPlan

	if associationsSet.IsNull() || associationsSet.IsUnknown() {
		return result, nil
	}

	// Convert the Set to a slice of Objects
	var objs []types.Object
	diags := associationsSet.ElementsAs(ctx, &objs, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to set to a slice of objects: %v", diags)
	}

	// Convert each Object to ProfileAssociationPlan
	for _, obj := range objs {
		var assoc britive_client.ProfileAssociationPlan
		diags = obj.As(ctx, &assoc, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert each object to profileAssociationPlan: %v", diags)
		}
		result = append(result, assoc)
	}

	return result, nil
}

func (rph *ResourceProfileHelper) mapProfileAssociationToTypesSet(plans []britive_client.ProfileAssociationPlan) (types.Set, error) {
	objs := make([]attr.Value, 0, len(plans))

	for _, p := range plans {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"type":        types.StringType,
				"value":       types.StringType,
				"parent_name": types.StringType,
			},
			map[string]attr.Value{
				"type":        p.Type,
				"value":       p.Value,
				"parent_name": p.ParentName,
			},
		)
		if diags.HasError() {
			return types.Set{}, fmt.Errorf("failed to create object for association: %v", diags)
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":        types.StringType,
				"value":       types.StringType,
				"parent_name": types.StringType,
			},
		},
		objs,
	)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to create set of associations: %v", diags)
	}

	return set, nil
}

func (rph *ResourceProfileHelper) saveProfileAssociations(ctx context.Context, plan britive_client.ProfilePlan, appContainerID, profileID string, client britive_client.Client) error {
	appRootEnvironmentGroup, err := client.GetApplicationRootEnvironmentGroup(ctx, appContainerID)
	if err != nil {
		return err
	}
	if appRootEnvironmentGroup == nil {
		return nil
	}
	applicationType, err := client.GetApplicationType(ctx, appContainerID)
	if err != nil {
		return err
	}
	appType := applicationType.ApplicationType
	associationScopes := make([]britive_client.ProfileAssociation, 0)
	associationResources := make([]britive_client.ProfileAssociation, 0)
	as, err := rph.mapSetToProfileAssociations(ctx, plan.Associations)
	if err != nil {
		return err
	}
	unmatchedAssociations := make([]britive_client.ProfileAssociationPlan, 0)
	for _, a := range as {
		associationType := a.Type.ValueString()
		associationValue := a.Value.ValueString()
		var rootAssociations []britive_client.Association
		isAssociationExists := false
		switch associationType {
		case "EnvironmentGroup", "Environment":
			if associationType == "EnvironmentGroup" {
				rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
				if appType == "AWS" && strings.EqualFold("root", associationValue) {
					associationValue = "Root"
				} else if appType == "AWS Standalone" && strings.EqualFold("root", associationValue) {
					associationValue = "root"
				}
			} else {
				rootAssociations = appRootEnvironmentGroup.Environments
			}
			for _, aeg := range rootAssociations {
				if aeg.Name == associationValue || aeg.ID == associationValue {
					isAssociationExists = true
					associationScopes = rph.appendProfileAssociations(associationScopes, associationType, aeg.ID)
					break
				} else if associationType == "Environment" && appType == "AWS Standalone" {
					newAssociationValue := client.GetEnvId(ctx, appContainerID, associationValue)
					if aeg.ID == newAssociationValue {
						isAssociationExists = true
						associationScopes = rph.appendProfileAssociations(associationScopes, associationType, aeg.ID)
						break
					}
				}
			}
		case "ApplicationResource":
			associationParentName := a.ParentName.ValueString()
			if strings.TrimSpace(associationParentName) == "" {
				return errs.NewNotEmptyOrWhiteSpaceError("associations.parent_name")
			}
			r, err := client.GetProfileAssociationResource(profileID, associationValue, associationParentName)
			if errors.Is(err, britive_client.ErrNotFound) {
				isAssociationExists = false
			} else if err != nil {
				return err
			} else if r != nil {
				isAssociationExists = true
				associationResources = rph.appendProfileAssociations(associationResources, associationType, r.NativeID)
			}

		}
		if !isAssociationExists {
			unmatchedAssociations = append(unmatchedAssociations, a)
		}

	}
	if len(unmatchedAssociations) > 0 {
		return errs.NewNotFoundErrorf("associations %v", unmatchedAssociations)
	}
	err = client.SaveProfileAssociationScopes(ctx, profileID, associationScopes)
	if err != nil {
		return err
	}
	if len(associationResources) > 0 {
		err = client.SaveProfileAssociationResourceScopes(ctx, profileID, associationResources)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rph *ResourceProfileHelper) appendProfileAssociations(associations []britive_client.ProfileAssociation, associationType string, associationID string) []britive_client.ProfileAssociation {
	associations = append(associations, britive_client.ProfileAssociation{
		Type:  associationType,
		Value: associationID,
	})
	return associations
}

func (rph *ResourceProfileHelper) mapPlanToModel(plan britive_client.ProfilePlan, isUpdate bool) (*britive_client.Profile, error) {
	profile := britive_client.Profile{
		AppContainerID:     plan.AppContainerID.ValueString(),
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		AllowImpersonation: plan.AllowImpersonation.ValueBool(),
	}

	if !plan.DestinationUrl.IsNull() && !plan.DestinationUrl.IsUnknown() && plan.DestinationUrl.ValueString() != "" {
		profile.DestinationUrl = plan.DestinationUrl.ValueString()
	}

	if !isUpdate {
		if plan.Disabled.ValueBool() {
			profile.Status = "inactive"
		} else {
			profile.Status = "active"
		}
	}

	if !plan.ExpirationDuration.IsNull() && !plan.ExpirationDuration.IsUnknown() && plan.ExpirationDuration.ValueString() != "" {
		if d, err := time.ParseDuration(plan.ExpirationDuration.ValueString()); err == nil {
			profile.ExpirationDuration = int64(d / time.Millisecond)
		} else {
			return nil, err
		}
	}

	if plan.Extendable.ValueBool() {
		profile.Extendable = plan.Extendable.ValueBool()
		if !plan.NotificationPriorToExpiration.IsNull() && !plan.NotificationPriorToExpiration.IsUnknown() && plan.NotificationPriorToExpiration.ValueString() != "" {
			if d, err := time.ParseDuration(plan.NotificationPriorToExpiration.ValueString()); err == nil {
				v := int64(d / time.Millisecond)
				profile.NotificationPriorToExpiration = &v
			} else {
				return nil, err
			}
		}

		if !plan.ExtensionDuration.IsNull() && !plan.ExtensionDuration.IsUnknown() && plan.ExtensionDuration.ValueString() != "" {
			if d, err := time.ParseDuration(plan.ExtensionDuration.ValueString()); err == nil {
				v := int64(d / time.Millisecond)
				profile.ExtensionDuration = &v
			} else {
				return nil, err
			}
		}
		profile.ExtensionLimit = plan.ExtensionLimit.ValueInt64()
	}

	return &profile, nil
}
