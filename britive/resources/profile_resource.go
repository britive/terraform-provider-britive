package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProfileResource is the resource implementation.
type ProfileResource struct {
	client *britive.Client
}

// ProfileResourceModel describes the resource data model.
type ProfileResourceModel struct {
	ID                             types.String            `tfsdk:"id"`
	AppContainerID                 types.String            `tfsdk:"app_container_id"`
	AppName                        types.String            `tfsdk:"app_name"`
	Name                           types.String            `tfsdk:"name"`
	Description                    types.String            `tfsdk:"description"`
	Disabled                       types.Bool              `tfsdk:"disabled"`
	Associations                   []ProfileAssociationModel `tfsdk:"associations"`
	ExpirationDuration             validators.DurationStringValue `tfsdk:"expiration_duration"`
	Extendable                     types.Bool                     `tfsdk:"extendable"`
	NotificationPriorToExpiration  validators.DurationStringValue `tfsdk:"notification_prior_to_expiration"`
	ExtensionDuration              validators.DurationStringValue `tfsdk:"extension_duration"`
	ExtensionLimit                 types.Int64             `tfsdk:"extension_limit"`
	DestinationURL                 types.String            `tfsdk:"destination_url"`
	AllowImpersonation             types.Bool              `tfsdk:"allow_impersonation"`
}

type ProfileAssociationModel struct {
	Type       types.String `tfsdk:"type"`
	Value      types.String `tfsdk:"value"`
	ParentName types.String `tfsdk:"parent_name"`
}

// NewProfileResource is a helper function to simplify the provider implementation.
func NewProfileResource() resource.Resource {
	return &ProfileResource{}
}

// Metadata returns the resource type name.
func (r *ProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile"
}

// Schema defines the schema for the resource.
func (r *ProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a Britive profile",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The profile ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_container_id": schema.StringAttribute{
				Description: "The identity of the Britive application",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_name": schema.StringAttribute{
				Description: "The name of the Britive application",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Britive profile",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the Britive profile",
				Optional:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "To disable the Britive profile",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"expiration_duration": schema.StringAttribute{
				Description: "The expiration time for the Britive profile",
				Required:    true,
				CustomType:  validators.DurationStringType{},
				Validators: []validator.String{
					validators.Duration(),
				},
			},
			"extendable": schema.BoolAttribute{
				Description: "The Boolean flag that indicates whether profile expiry is extendable or not",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"notification_prior_to_expiration": schema.StringAttribute{
				Description: "The profile expiry notification as a time value",
				Optional:    true,
				CustomType:  validators.DurationStringType{},
				Validators: []validator.String{
					validators.Duration(),
				},
			},
			"extension_duration": schema.StringAttribute{
				Description: "The profile expiry extension as a time value",
				Optional:    true,
				CustomType:  validators.DurationStringType{},
				Validators: []validator.String{
					validators.Duration(),
				},
			},
			"extension_limit": schema.Int64Attribute{
				Description: "The repetition limit for extending the profile expiry",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"destination_url": schema.StringAttribute{
				Description: "The destination url to redirect user after checkout",
				Optional:    true,
			},
			"allow_impersonation": schema.BoolAttribute{
				Description: "Enable or disable delegation",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				Description: "The list of associations for the Britive profile",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The type of association, should be one of [Environment, EnvironmentGroup, ApplicationResource]",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("Environment", "EnvironmentGroup", "ApplicationResource"),
							},
						},
						"value": schema.StringAttribute{
							Description: "The association value",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"parent_name": schema.StringAttribute{
							Description: "The parent name of the resource. Required only if the association type is ApplicationResource",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ProfileResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ProfileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When extendable is true, notification_prior_to_expiration and extension_duration are required
	if !data.Extendable.IsNull() && data.Extendable.ValueBool() {
		if data.NotificationPriorToExpiration.IsNull() || data.NotificationPriorToExpiration.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("notification_prior_to_expiration"),
				"Missing Required Field",
				"When extendable is true, notification_prior_to_expiration must be provided",
			)
		}
		if data.ExtensionDuration.IsNull() || data.ExtensionDuration.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("extension_duration"),
				"Missing Required Field",
				"When extendable is true, extension_duration must be provided",
			)
		}
	}

	// Validate that ApplicationResource associations have parent_name
	for i, assoc := range data.Associations {
		if assoc.Type.ValueString() == "ApplicationResource" {
			if assoc.ParentName.IsNull() || strings.TrimSpace(assoc.ParentName.ValueString()) == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("associations"),
					"Missing Required Field",
					fmt.Sprintf("parent_name is required for ApplicationResource associations (association %d)", i),
				)
			}
		}
	}
}

// UpgradeState normalizes states from prior schema versions.
// Version 0 (v2.x SDKv2): unset Optional-only string fields were stored as ""
// in the flatmap; normalize them to null so they match the config expectation.
func (r *ProfileResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// PriorSchema describes the v0 state shape (same structure as v1, no validators/modifiers
	// needed — the Framework uses it only for type-safe unmarshaling).
	priorSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                              schema.StringAttribute{Computed: true},
			"app_container_id":                schema.StringAttribute{Required: true},
			"app_name":                        schema.StringAttribute{Optional: true, Computed: true},
			"name":                            schema.StringAttribute{Required: true},
			"description":                     schema.StringAttribute{Optional: true},
			"disabled":                        schema.BoolAttribute{Optional: true, Computed: true},
			"expiration_duration":             schema.StringAttribute{Required: true, CustomType: validators.DurationStringType{}},
			"extendable":                      schema.BoolAttribute{Optional: true, Computed: true},
			"notification_prior_to_expiration": schema.StringAttribute{Optional: true, CustomType: validators.DurationStringType{}},
			"extension_duration":              schema.StringAttribute{Optional: true, CustomType: validators.DurationStringType{}},
			"extension_limit":                 schema.Int64Attribute{Optional: true},
			"destination_url":                 schema.StringAttribute{Optional: true},
			"allow_impersonation":             schema.BoolAttribute{Optional: true, Computed: true},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type":        schema.StringAttribute{Required: true},
						"value":       schema.StringAttribute{Required: true},
						"parent_name": schema.StringAttribute{Optional: true, Computed: true},
					},
				},
			},
		},
	}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorState ProfileResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}
				if !priorState.Description.IsNull() && priorState.Description.ValueString() == "" {
					priorState.Description = types.StringNull()
				}
				if !priorState.DestinationURL.IsNull() && priorState.DestinationURL.ValueString() == "" {
					priorState.DestinationURL = types.StringNull()
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &priorState)...)
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map resource to API model
	profile, err := r.mapResourceToModel(&plan, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Profile", err.Error())
		return
	}

	log.Printf("[INFO] Creating new profile: %#v", profile)

	// Create profile
	p, err := r.client.CreateProfile(profile.AppContainerID, *profile)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Profile", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new profile: %#v", p)

	// Save profile associations — if this fails, delete the profile we just
	// created so Terraform doesn't leave an orphaned resource in Britive.
	err = r.saveProfileAssociations(p.AppContainerID, p.ProfileID, &plan)
	if err != nil {
		if deleteErr := r.client.DeleteProfile(p.AppContainerID, p.ProfileID); deleteErr != nil {
			log.Printf("[WARN] Failed to delete profile %s after association error: %s", p.ProfileID, deleteErr)
		}
		resp.Diagnostics.AddError("Error Saving Profile Associations", err.Error())
		return
	}

	// Set ID in state
	plan.ID = types.StringValue(p.ProfileID)

	// Read back to get computed values
	profile, err = r.client.GetProfile(p.ProfileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Profile", err.Error())
		return
	}

	// Map model to resource
	err = r.mapModelToResource(profile, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Profile", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := state.ID.ValueString()

	log.Printf("[INFO] Reading profile %s", profileID)

	profile, err := r.client.GetProfile(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Profile", err.Error())
		return
	}

	log.Printf("[INFO] Received profile %#v", profile)

	// Map model to resource
	err = r.mapModelToResource(profile, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Profile", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfileResourceModel
	var state ProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := state.ID.ValueString()
	appContainerID := plan.AppContainerID.ValueString()

	// Check for changes in main profile fields
	if !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.ExpirationDuration.Equal(state.ExpirationDuration) ||
		!plan.Extendable.Equal(state.Extendable) ||
		!plan.NotificationPriorToExpiration.Equal(state.NotificationPriorToExpiration) ||
		!plan.ExtensionDuration.Equal(state.ExtensionDuration) ||
		!plan.ExtensionLimit.Equal(state.ExtensionLimit) ||
		!plan.DestinationURL.Equal(state.DestinationURL) ||
		!plan.AllowImpersonation.Equal(state.AllowImpersonation) ||
		!associationsEqual(plan.Associations, state.Associations) {

		profile, err := r.mapResourceToModel(&plan, true)
		if err != nil {
			resp.Diagnostics.AddError("Error Mapping Profile", err.Error())
			return
		}

		log.Printf("[INFO] Updating profile: %#v", profile)

		up, err := r.client.UpdateProfile(appContainerID, profileID, *profile)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Profile", err.Error())
			return
		}

		log.Printf("[INFO] Submitted updated profile: %#v", up)

		err = r.saveProfileAssociations(appContainerID, profileID, &plan)
		if err != nil {
			resp.Diagnostics.AddError("Error Saving Profile Associations", err.Error())
			return
		}
	}

	// Handle disabled status separately
	if !plan.Disabled.Equal(state.Disabled) {
		disabled := plan.Disabled.ValueBool()

		log.Printf("[INFO] Updating status disabled: %t of profile: %s", disabled, profileID)
		up, err := r.client.EnableOrDisableProfile(appContainerID, profileID, disabled)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Profile Status", err.Error())
			return
		}

		log.Printf("[INFO] Submitted updated status of profile: %#v", up)
	}

	// Read back to get updated values
	profile, err := r.client.GetProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Profile", err.Error())
		return
	}

	// Map model to resource
	err = r.mapModelToResource(profile, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Profile", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := state.ID.ValueString()
	appContainerID := state.AppContainerID.ValueString()

	log.Printf("[INFO] Deleting profile: %s/%s", appContainerID, profileID)

	err := r.client.DeleteProfile(appContainerID, profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Profile", err.Error())
		return
	}

	log.Printf("[INFO] Deleted profile: %s/%s", appContainerID, profileID)
}

// ImportState imports the resource state.
func (r *ProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. apps/{app_name}/paps/{profile_name}
	// 2. {app_name}/{profile_name}

	importID := req.ID
	var appName, profileName string

	if strings.Contains(importID, "/paps/") {
		// Format: apps/{app_name}/paps/{profile_name}
		parts := strings.Split(importID, "/")
		if len(parts) != 4 || parts[0] != "apps" || parts[2] != "paps" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Import ID must be in format 'apps/{app_name}/paps/{profile_name}' or '{app_name}/{profile_name}', got: %s", importID),
			)
			return
		}
		appName = parts[1]
		profileName = parts[3]
	} else {
		// Format: {app_name}/{profile_name}
		parts := strings.Split(importID, "/")
		if len(parts) != 2 {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Import ID must be in format 'apps/{app_name}/paps/{profile_name}' or '{app_name}/{profile_name}', got: %s", importID),
			)
			return
		}
		appName = parts[0]
		profileName = parts[1]
	}

	if strings.TrimSpace(appName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "app_name cannot be empty")
		return
	}
	if strings.TrimSpace(profileName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "profile_name cannot be empty")
		return
	}

	log.Printf("[INFO] Importing profile: %s/%s", appName, profileName)

	// Get application by name
	app, err := r.client.GetApplicationByName(appName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Application Not Found", fmt.Sprintf("Application %s not found", appName))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Application", err.Error())
		return
	}

	// Get profile by name
	profile, err := r.client.GetProfileByName(app.AppContainerID, profileName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Profile Not Found", fmt.Sprintf("Profile %s not found", profileName))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Profile", err.Error())
		return
	}

	// Set the state
	var state ProfileResourceModel
	state.ID = types.StringValue(profile.ProfileID)

	// Map model to resource
	err = r.mapModelToResource(profile, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Profile", err.Error())
		return
	}

	// Clear app_name (only used for import)
	state.AppName = types.StringValue("")

	log.Printf("[INFO] Imported profile: %s/%s", appName, profileName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ProfileResource) mapResourceToModel(plan *ProfileResourceModel, isUpdate bool) (*britive.Profile, error) {
	profile := &britive.Profile{}

	profile.AppContainerID = plan.AppContainerID.ValueString()
	profile.Name = plan.Name.ValueString()
	profile.Description = plan.Description.ValueString()
	profile.DelegationEnabled = plan.AllowImpersonation.ValueBool()

	if !isUpdate {
		if plan.Disabled.ValueBool() {
			profile.Status = "inactive"
		} else {
			profile.Status = "active"
		}
	}

	// Parse expiration duration
	expirationDuration, err := time.ParseDuration(plan.ExpirationDuration.ValueString())
	if err != nil {
		return nil, fmt.Errorf("invalid expiration_duration: %w", err)
	}
	profile.ExpirationDuration = int64(expirationDuration / time.Millisecond)

	profile.DestinationUrl = plan.DestinationURL.ValueString()

	extendable := plan.Extendable.ValueBool()
	profile.Extendable = extendable

	if extendable {
		notificationPriorToExpirationString := plan.NotificationPriorToExpiration.ValueString()
		if notificationPriorToExpirationString == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("notification_prior_to_expiration")
		}
		notificationPriorToExpiration, err := time.ParseDuration(notificationPriorToExpirationString)
		if err != nil {
			return nil, fmt.Errorf("invalid notification_prior_to_expiration: %w", err)
		}
		nullableNotificationPriorToExpiration := int64(notificationPriorToExpiration / time.Millisecond)
		profile.NotificationPriorToExpiration = &nullableNotificationPriorToExpiration

		extensionDurationString := plan.ExtensionDuration.ValueString()
		if extensionDurationString == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("extension_duration")
		}
		extensionDuration, err := time.ParseDuration(extensionDurationString)
		if err != nil {
			return nil, fmt.Errorf("invalid extension_duration: %w", err)
		}
		nullableExtensionDuration := int64(extensionDuration / time.Millisecond)
		profile.ExtensionDuration = &nullableExtensionDuration

		if !plan.ExtensionLimit.IsNull() {
			profile.ExtensionLimit = int(plan.ExtensionLimit.ValueInt64())
		}
	}

	return profile, nil
}

func (r *ProfileResource) mapModelToResource(profile *britive.Profile, state *ProfileResourceModel) error {
	state.AppContainerID = types.StringValue(profile.AppContainerID)
	// app_name is not returned by the API; preserve existing value.
	// Use "" (not null) so UseStateForUnknown can copy it on subsequent plans,
	// which is critical when migrating from v2.x where app_name was absent from state.
	if state.AppName.IsUnknown() || state.AppName.IsNull() {
		state.AppName = types.StringValue("")
	}
	state.Name = types.StringValue(profile.Name)
	// Optional-only field: map empty API response to null (to avoid null vs "" mismatch in plan)
	if profile.Description != "" {
		state.Description = types.StringValue(profile.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.Disabled = types.BoolValue(strings.EqualFold(profile.Status, "inactive"))
	state.ExpirationDuration = validators.NewDurationStringValue(time.Duration(profile.ExpirationDuration * int64(time.Millisecond)).String())
	state.Extendable = types.BoolValue(profile.Extendable)
	state.AllowImpersonation = types.BoolValue(profile.DelegationEnabled)

	if profile.Extendable {
		if profile.NotificationPriorToExpiration != nil {
			state.NotificationPriorToExpiration = validators.NewDurationStringValue(time.Duration(*profile.NotificationPriorToExpiration * int64(time.Millisecond)).String())
		}
		if profile.ExtensionDuration != nil {
			state.ExtensionDuration = validators.NewDurationStringValue(time.Duration(*profile.ExtensionDuration * int64(time.Millisecond)).String())
		}
		// Handle ExtensionLimit type conversion (interface{} could be int, float64, etc.)
		if profile.ExtensionLimit != nil {
			switch v := profile.ExtensionLimit.(type) {
			case int:
				state.ExtensionLimit = types.Int64Value(int64(v))
			case int64:
				state.ExtensionLimit = types.Int64Value(v)
			case float64:
				state.ExtensionLimit = types.Int64Value(int64(v))
			}
		}
	}

	// Optional-only field: map empty API response to null (to avoid null vs "" mismatch in plan)
	if profile.DestinationUrl != "" {
		state.DestinationURL = types.StringValue(profile.DestinationUrl)
	} else {
		state.DestinationURL = types.StringNull()
	}

	// Map associations
	associations, err := r.mapProfileAssociationsModelToResource(profile.AppContainerID, profile.ProfileID, profile.Associations, state)
	if err != nil {
		return err
	}
	state.Associations = associations

	return nil
}

func (r *ProfileResource) saveProfileAssociations(appContainerID string, profileID string, plan *ProfileResourceModel) error {
	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(appContainerID)
	if err != nil {
		return err
	}
	if appRootEnvironmentGroup == nil {
		return nil
	}

	applicationType, err := r.client.GetApplicationType(appContainerID)
	if err != nil {
		return err
	}
	appType := applicationType.ApplicationType

	associationScopes := make([]britive.ProfileAssociation, 0)
	associationResources := make([]britive.ProfileAssociation, 0)
	unmatchedAssociations := make([]ProfileAssociationModel, 0)

	for _, a := range plan.Associations {
		associationType := a.Type.ValueString()
		associationValue := a.Value.ValueString()
		isAssociationExists := false

		switch associationType {
		case "EnvironmentGroup", "Environment":
			var rootAssociations []britive.Association
			if associationType == "EnvironmentGroup" {
				rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
				// Handle AWS Root vs root
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
					associationScopes = append(associationScopes, britive.ProfileAssociation{
						Type:  associationType,
						Value: aeg.ID,
					})
					break
				} else if associationType == "Environment" && appType == "AWS Standalone" {
					newAssociationValue := r.client.GetEnvId(appContainerID, associationValue)
					if aeg.ID == newAssociationValue {
						isAssociationExists = true
						associationScopes = append(associationScopes, britive.ProfileAssociation{
							Type:  associationType,
							Value: aeg.ID,
						})
						break
					}
				}
			}

		case "ApplicationResource":
			associationParentName := a.ParentName.ValueString()
			if strings.TrimSpace(associationParentName) == "" {
				return errs.NewNotEmptyOrWhiteSpaceError("associations.parent_name")
			}

			resourceAssoc, err := r.client.GetProfileAssociationResource(profileID, associationValue, associationParentName)
			if errors.Is(err, britive.ErrNotFound) {
				isAssociationExists = false
			} else if err != nil {
				return err
			} else if resourceAssoc != nil {
				isAssociationExists = true
				associationResources = append(associationResources, britive.ProfileAssociation{
					Type:  associationType,
					Value: resourceAssoc.NativeID,
				})
			}
		}

		if !isAssociationExists {
			unmatchedAssociations = append(unmatchedAssociations, a)
		}
	}

	if len(unmatchedAssociations) > 0 {
		return errs.NewNotFoundErrorf("associations %v", unmatchedAssociations)
	}

	log.Printf("[INFO] Updating profile %s associations: %#v", profileID, associationScopes)
	err = r.client.SaveProfileAssociationScopes(profileID, associationScopes)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted Update profile %s associations: %#v", profileID, associationScopes)

	if len(associationResources) > 0 {
		log.Printf("[INFO] Updating profile %s association resources: %#v", profileID, associationResources)
		err = r.client.SaveProfileAssociationResourceScopes(profileID, associationResources)
		if err != nil {
			return err
		}
		log.Printf("[INFO] Submitted Update profile %s association resources: %#v", profileID, associationResources)
	}

	return nil
}

func (r *ProfileResource) mapProfileAssociationsModelToResource(appContainerID string, profileID string, associations []britive.ProfileAssociation, state *ProfileResourceModel) ([]ProfileAssociationModel, error) {
	appRootEnvironmentGroup, err := r.client.GetApplicationRootEnvironmentGroup(appContainerID)
	if err != nil {
		return nil, err
	}

	if len(associations) == 0 || appRootEnvironmentGroup == nil {
		return make([]ProfileAssociationModel, 0), nil
	}

	applicationType, err := r.client.GetApplicationType(appContainerID)
	if err != nil {
		return nil, err
	}
	appType := applicationType.ApplicationType

	profileAssociations := make([]ProfileAssociationModel, 0)

	for _, association := range associations {
		switch association.Type {
		case "EnvironmentGroup", "Environment":
			var rootAssociations []britive.Association
			if association.Type == "EnvironmentGroup" {
				rootAssociations = appRootEnvironmentGroup.EnvironmentGroups
			} else {
				rootAssociations = appRootEnvironmentGroup.Environments
			}

			var a *britive.Association
			for _, aeg := range rootAssociations {
				if aeg.ID == association.Value {
					a = &aeg
					break
				}
			}

			if a == nil {
				return nil, errs.NewNotFoundErrorf("association %s", association.Value)
			}

			associationValue := a.Name

			// Check if input used ID or name
			for _, inputAssoc := range state.Associations {
				if inputAssoc.Type.ValueString() == association.Type {
					inputValue := inputAssoc.Value.ValueString()

					if association.Type == "EnvironmentGroup" && (appType == "AWS" || appType == "AWS Standalone") && strings.EqualFold("root", a.Name) && strings.EqualFold("root", inputValue) {
						associationValue = inputValue
					}

					if a.ID == inputValue {
						associationValue = a.ID
						break
					} else if association.Type == "Environment" && appType == "AWS Standalone" {
						envID := r.client.GetEnvId(appContainerID, inputValue)
						if a.ID == envID {
							associationValue = inputValue
							break
						}
					}
				}
			}

			profileAssociations = append(profileAssociations, ProfileAssociationModel{
				Type:       types.StringValue(association.Type),
				Value:      types.StringValue(associationValue),
				ParentName: types.StringValue(""),
			})

		case "ApplicationResource":
			par, err := r.client.GetProfileAssociationResourceByNativeID(profileID, association.Value)
			if errors.Is(err, britive.ErrNotFound) {
				return nil, errs.NewNotFoundErrorf("application resource %s", association.Value)
			} else if err != nil {
				return nil, err
			} else if par == nil {
				return nil, errs.NewNotFoundErrorf("application resource %s", association.Value)
			}

			profileAssociations = append(profileAssociations, ProfileAssociationModel{
				Type:       types.StringValue(association.Type),
				Value:      types.StringValue(par.Name),
				ParentName: types.StringValue(par.ParentName),
			})
		}
	}

	return profileAssociations, nil
}

// associationsEqual compares two slices of ProfileAssociationModel for equality
func associationsEqual(a, b []ProfileAssociationModel) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for comparison (order doesn't matter for sets)
	aMap := make(map[string]ProfileAssociationModel)
	for _, assoc := range a {
		key := fmt.Sprintf("%s|%s|%s", assoc.Type.ValueString(), assoc.Value.ValueString(), assoc.ParentName.ValueString())
		aMap[key] = assoc
	}

	bMap := make(map[string]ProfileAssociationModel)
	for _, assoc := range b {
		key := fmt.Sprintf("%s|%s|%s", assoc.Type.ValueString(), assoc.Value.ValueString(), assoc.ParentName.ValueString())
		bMap[key] = assoc
	}

	if len(aMap) != len(bMap) {
		return false
	}

	for key := range aMap {
		if _, ok := bMap[key]; !ok {
			return false
		}
	}

	return true
}
