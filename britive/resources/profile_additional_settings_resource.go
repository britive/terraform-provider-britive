package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                    = &ProfileAdditionalSettingsResource{}
	_ resource.ResourceWithConfigure       = &ProfileAdditionalSettingsResource{}
	_ resource.ResourceWithImportState     = &ProfileAdditionalSettingsResource{}
	_ resource.ResourceWithUpgradeState    = &ProfileAdditionalSettingsResource{}
)

func NewProfileAdditionalSettingsResource() resource.Resource {
	return &ProfileAdditionalSettingsResource{}
}

type ProfileAdditionalSettingsResource struct {
	client *britive.Client
}

type ProfileAdditionalSettingsResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	ProfileID                    types.String `tfsdk:"profile_id"`
	UseAppCredentialType         types.Bool   `tfsdk:"use_app_credential_type"`
	ConsoleAccess                types.Bool   `tfsdk:"console_access"`
	ProgrammaticAccess           types.Bool   `tfsdk:"programmatic_access"`
	ProjectIDForServiceAccount   types.String `tfsdk:"project_id_for_service_account"`
}

func (r *ProfileAdditionalSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile_additional_settings"
}

func (r *ProfileAdditionalSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a Britive profile additional settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the profile additional settings.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"use_app_credential_type": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Inherit the credential type settings from the application.",
			},
			"console_access": schema.BoolAttribute{
				Optional:    true,
				Description: "Provide the console access for the profile, overridden if use_app_credential_type is set to true.",
			},
			"programmatic_access": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Provide the programmatic access for the profile, overridden if use_app_credential_type is set to true.",
			},
			"project_id_for_service_account": schema.StringAttribute{
				Optional:    true,
				Description: "The project id for creating service accounts.",
			},
		},
	}
}

func (r *ProfileAdditionalSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// UpgradeState normalizes states from prior schema versions.
// Version 0 (v2.x SDKv2): unset Optional-only string field project_id_for_service_account
// was stored as "" in the flatmap; normalize to null so it matches config expectation.
func (r *ProfileAdditionalSettingsResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                             schema.StringAttribute{Computed: true},
			"profile_id":                     schema.StringAttribute{Required: true},
			"use_app_credential_type":        schema.BoolAttribute{Optional: true, Computed: true},
			"console_access":                 schema.BoolAttribute{Optional: true},
			"programmatic_access":            schema.BoolAttribute{Optional: true, Computed: true},
			"project_id_for_service_account": schema.StringAttribute{Optional: true},
		},
	}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorState ProfileAdditionalSettingsResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}
				if !priorState.ProjectIDForServiceAccount.IsNull() && priorState.ProjectIDForServiceAccount.ValueString() == "" {
					priorState.ProjectIDForServiceAccount = types.StringNull()
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &priorState)...)
			},
		},
	}
}

func (r *ProfileAdditionalSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfileAdditionalSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := plan.ProfileID.ValueString()

	profileAdditionalSettings := britive.ProfileAdditionalSettings{
		ProfileID:                      profileID,
		UseApplicationCredentialType:   plan.UseAppCredentialType.ValueBool(),
		ConsoleAccess:                  plan.ConsoleAccess.ValueBool(),
		ProgrammaticAccess:             plan.ProgrammaticAccess.ValueBool(),
	}

	// Only set ProjectIdForServiceAccount if provided
	if !plan.ProjectIDForServiceAccount.IsNull() {
		profileAdditionalSettings.ProjectIdForServiceAccount = plan.ProjectIDForServiceAccount.ValueString()
	}

	_, err := r.client.UpdateProfileAdditionalSettings(profileAdditionalSettings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Profile Additional Settings",
			fmt.Sprintf("Could not create profile additional settings: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateProfileAdditionalSettingsID(profileID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Additional Settings",
			fmt.Sprintf("Could not read profile additional settings after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfileAdditionalSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfileAdditionalSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, err := parseProfileAdditionalSettingsID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Additional Settings ID",
			fmt.Sprintf("Could not parse profile additional settings ID: %s", err.Error()),
		)
		return
	}

	profileAdditionalSettings, err := r.client.GetProfileAdditionalSettings(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Additional Settings",
			fmt.Sprintf("Could not read profile additional settings: %s", err.Error()),
		)
		return
	}

	state.ProfileID = types.StringValue(profileID)
	state.UseAppCredentialType = types.BoolValue(profileAdditionalSettings.UseApplicationCredentialType)
	state.ConsoleAccess = types.BoolValue(profileAdditionalSettings.ConsoleAccess)
	state.ProgrammaticAccess = types.BoolValue(profileAdditionalSettings.ProgrammaticAccess)

	// Only set ProjectIDForServiceAccount if it was configured or during import
	if !state.ProjectIDForServiceAccount.IsNull() || profileAdditionalSettings.ProjectIdForServiceAccount != "" {
		state.ProjectIDForServiceAccount = types.StringValue(profileAdditionalSettings.ProjectIdForServiceAccount)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfileAdditionalSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfileAdditionalSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, err := parseProfileAdditionalSettingsID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Additional Settings ID",
			fmt.Sprintf("Could not parse profile additional settings ID: %s", err.Error()),
		)
		return
	}

	profileAdditionalSettings := britive.ProfileAdditionalSettings{
		ProfileID:                      profileID,
		UseApplicationCredentialType:   plan.UseAppCredentialType.ValueBool(),
		ConsoleAccess:                  plan.ConsoleAccess.ValueBool(),
		ProgrammaticAccess:             plan.ProgrammaticAccess.ValueBool(),
	}

	// Only set ProjectIdForServiceAccount if provided
	if !plan.ProjectIDForServiceAccount.IsNull() {
		profileAdditionalSettings.ProjectIdForServiceAccount = plan.ProjectIDForServiceAccount.ValueString()
	}

	_, err = r.client.UpdateProfileAdditionalSettings(profileAdditionalSettings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Profile Additional Settings",
			fmt.Sprintf("Could not update profile additional settings: %s", err.Error()),
		)
		return
	}

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Additional Settings",
			fmt.Sprintf("Could not read profile additional settings after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfileAdditionalSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfileAdditionalSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, err := parseProfileAdditionalSettingsID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Additional Settings ID",
			fmt.Sprintf("Could not parse profile additional settings ID: %s", err.Error()),
		)
		return
	}

	// Reset to defaults on delete
	profileAdditionalSettings := britive.ProfileAdditionalSettings{
		ProfileID:                      profileID,
		UseApplicationCredentialType:   true,
		ConsoleAccess:                  false,
		ProgrammaticAccess:             false,
		ProjectIdForServiceAccount:     "",
	}

	_, err = r.client.UpdateProfileAdditionalSettings(profileAdditionalSettings)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Profile Additional Settings",
			fmt.Sprintf("Could not delete profile additional settings: %s", err.Error()),
		)
		return
	}
}

func (r *ProfileAdditionalSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support import format: paps/{profile_id}/additional-settings
	idRegex := `^paps/(?P<profile_id>[^/]+)/additional-settings$`

	re := regexp.MustCompile(idRegex)
	matches := re.FindStringSubmatch(req.ID)

	if matches == nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected format: 'paps/{profile_id}/additional-settings'", req.ID),
		)
		return
	}

	var profileID string
	for i, matchName := range re.SubexpNames() {
		if matchName == "profile_id" && i < len(matches) {
			profileID = matches[i]
			break
		}
	}

	if strings.TrimSpace(profileID) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"profile_id cannot be empty or whitespace.",
		)
		return
	}

	profileAdditionalSettings, err := r.client.GetProfileAdditionalSettings(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Profile Additional Settings Not Found",
			fmt.Sprintf("Profile additional settings for profile '%s' not found.", profileID),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Additional Settings",
			fmt.Sprintf("Could not import profile additional settings: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateProfileAdditionalSettingsID(profileID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("use_app_credential_type"), profileAdditionalSettings.UseApplicationCredentialType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("console_access"), profileAdditionalSettings.ConsoleAccess)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("programmatic_access"), profileAdditionalSettings.ProgrammaticAccess)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id_for_service_account"), profileAdditionalSettings.ProjectIdForServiceAccount)...)
}

// populateStateFromAPI fetches profile additional settings data from API and populates the state model
func (r *ProfileAdditionalSettingsResource) populateStateFromAPI(ctx context.Context, state *ProfileAdditionalSettingsResourceModel) error {
	profileID, err := parseProfileAdditionalSettingsID(state.ID.ValueString())
	if err != nil {
		return err
	}

	profileAdditionalSettings, err := r.client.GetProfileAdditionalSettings(profileID)
	if err != nil {
		return err
	}

	state.UseAppCredentialType = types.BoolValue(profileAdditionalSettings.UseApplicationCredentialType)
	state.ConsoleAccess = types.BoolValue(profileAdditionalSettings.ConsoleAccess)
	state.ProgrammaticAccess = types.BoolValue(profileAdditionalSettings.ProgrammaticAccess)

	// Only set ProjectIDForServiceAccount if it was configured or has a value from API
	if !state.ProjectIDForServiceAccount.IsNull() || profileAdditionalSettings.ProjectIdForServiceAccount != "" {
		state.ProjectIDForServiceAccount = types.StringValue(profileAdditionalSettings.ProjectIdForServiceAccount)
	}

	return nil
}

// Helper functions
func generateProfileAdditionalSettingsID(profileID string) string {
	return fmt.Sprintf("paps/%s/additional-settings", profileID)
}

func parseProfileAdditionalSettingsID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid profile additional settings ID format: %s", id)
	}
	return parts[1], nil
}
