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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ProfileSessionAttributeResource{}
	_ resource.ResourceWithConfigure      = &ProfileSessionAttributeResource{}
	_ resource.ResourceWithImportState    = &ProfileSessionAttributeResource{}
	_ resource.ResourceWithValidateConfig = &ProfileSessionAttributeResource{}
	_ resource.ResourceWithUpgradeState   = &ProfileSessionAttributeResource{}
)

func NewProfileSessionAttributeResource() resource.Resource {
	return &ProfileSessionAttributeResource{}
}

type ProfileSessionAttributeResource struct {
	client *britive.Client
}

type ProfileSessionAttributeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	AppName        types.String `tfsdk:"app_name"`
	ProfileID      types.String `tfsdk:"profile_id"`
	ProfileName    types.String `tfsdk:"profile_name"`
	AttributeName  types.String `tfsdk:"attribute_name"`
	AttributeType  types.String `tfsdk:"attribute_type"`
	AttributeValue types.String `tfsdk:"attribute_value"`
	MappingName    types.String `tfsdk:"mapping_name"`
	Transitive     types.Bool   `tfsdk:"transitive"`
}

func (r *ProfileSessionAttributeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile_session_attribute"
}

func (r *ProfileSessionAttributeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a Britive profile session attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the profile session attribute.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The application name of the application the profile is associated with.",
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
			"profile_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the profile.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"attribute_name": schema.StringAttribute{
				Optional:    true,
				Description: "The attribute name associated with the profile. Required when attribute_type is 'Identity'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"attribute_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Identity"),
				Description: "The type of attribute associated with the profile. Must be 'Static' or 'Identity'.",
				Validators: []validator.String{
					stringvalidator.OneOf("Static", "Identity"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"attribute_value": schema.StringAttribute{
				Optional:    true,
				Description: "The attribute value associated with the profile. Required when attribute_type is 'Static'.",
			},
			"mapping_name": schema.StringAttribute{
				Required:    true,
				Description: "The attribute mapping name associated with the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"transitive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "The attribute transitive associated with the profile.",
			},
		},
	}
}

// UpgradeState normalizes states from prior schema versions.
// Version 0 (v2.x SDKv2): unset Optional-only string fields (attribute_value,
// attribute_name) were stored as "" in the flatmap; normalize to null.
func (r *ProfileSessionAttributeResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true},
			"app_name":        schema.StringAttribute{Optional: true, Computed: true},
			"profile_id":      schema.StringAttribute{Required: true},
			"profile_name":    schema.StringAttribute{Optional: true, Computed: true},
			"attribute_name":  schema.StringAttribute{Optional: true},
			"attribute_type":  schema.StringAttribute{Optional: true, Computed: true},
			"attribute_value": schema.StringAttribute{Optional: true},
			"mapping_name":    schema.StringAttribute{Required: true},
			"transitive":      schema.BoolAttribute{Optional: true, Computed: true},
		},
	}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorState ProfileSessionAttributeResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}
				if !priorState.AttributeValue.IsNull() && priorState.AttributeValue.ValueString() == "" {
					priorState.AttributeValue = types.StringNull()
				}
				if !priorState.AttributeName.IsNull() && priorState.AttributeName.ValueString() == "" {
					priorState.AttributeName = types.StringNull()
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &priorState)...)
			},
		},
	}
}

func (r *ProfileSessionAttributeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ProfileSessionAttributeResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attributeType := data.AttributeType.ValueString()
	if attributeType == "" {
		attributeType = "Identity" // default
	}

	// Validate conditional requirements based on attribute_type
	if strings.EqualFold(attributeType, "Identity") {
		// When Identity: attribute_name is required, attribute_value must be empty
		if data.AttributeName.IsNull() || data.AttributeName.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"attribute_name is required when attribute_type is 'Identity'",
			)
		}
		if !data.AttributeValue.IsNull() && data.AttributeValue.ValueString() != "" {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"attribute_value must be empty when attribute_type is 'Identity'",
			)
		}
	} else if strings.EqualFold(attributeType, "Static") {
		// When Static: attribute_value is required, attribute_name must be empty
		if data.AttributeValue.IsNull() || data.AttributeValue.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"attribute_value is required when attribute_type is 'Static'",
			)
		}
		if !data.AttributeName.IsNull() && data.AttributeName.ValueString() != "" {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"attribute_name must be empty when attribute_type is 'Static'",
			)
		}
	}
}

func (r *ProfileSessionAttributeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProfileSessionAttributeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfileSessionAttributeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := plan.ProfileID.ValueString()

	// Build session attribute based on type
	sessionAttribute, err := r.buildSessionAttribute(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Session Attribute",
			fmt.Sprintf("Could not build session attribute: %s", err.Error()),
		)
		return
	}

	created, err := r.client.CreateProfileSessionAttribute(profileID, *sessionAttribute)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Profile Session Attribute",
			fmt.Sprintf("Could not create profile session attribute: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateProfileSessionAttributeID(profileID, created.ID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Session Attribute",
			fmt.Sprintf("Could not read profile session attribute after creation: %s", err.Error()),
		)
		return
	}

	// app_name and profile_name are Optional+Computed but not returned by the API.
	// Set them to null if still unknown to avoid "provider returned unknown value" errors.
	if plan.AppName.IsUnknown() {
		plan.AppName = types.StringNull()
	}
	if plan.ProfileName.IsUnknown() {
		plan.ProfileName = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfileSessionAttributeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfileSessionAttributeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.populateStateFromAPI(ctx, &state); err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Profile Session Attribute",
			fmt.Sprintf("Could not read profile session attribute: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfileSessionAttributeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfileSessionAttributeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, sessionAttributeID, err := parseProfileSessionAttributeID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Session Attribute ID",
			fmt.Sprintf("Could not parse profile session attribute ID: %s", err.Error()),
		)
		return
	}

	// Build session attribute
	sessionAttribute, err := r.buildSessionAttribute(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Building Session Attribute",
			fmt.Sprintf("Could not build session attribute: %s", err.Error()),
		)
		return
	}
	sessionAttribute.ID = sessionAttributeID

	_, err = r.client.UpdateProfileSessionAttribute(profileID, *sessionAttribute)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Profile Session Attribute",
			fmt.Sprintf("Could not update profile session attribute: %s", err.Error()),
		)
		return
	}

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Session Attribute",
			fmt.Sprintf("Could not read profile session attribute after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfileSessionAttributeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfileSessionAttributeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, sessionAttributeID, err := parseProfileSessionAttributeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Session Attribute ID",
			fmt.Sprintf("Could not parse profile session attribute ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeleteProfileSessionAttribute(profileID, sessionAttributeID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Profile Session Attribute",
			fmt.Sprintf("Could not delete profile session attribute: %s", err.Error()),
		)
		return
	}
}

func (r *ProfileSessionAttributeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. apps/{app_name}/paps/{profile_name}/session-attributes/type/{attribute_type}/mapping-name/{mapping_name}
	// 2. {app_name}/{profile_name}/{attribute_type}/{mapping_name}
	idRegexes := []string{
		`^apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/session-attributes/type/(?P<attribute_type>[^/]+)/mapping-name/(?P<mapping_name>[^/]+)$`,
		`^(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<attribute_type>[^/]+)/(?P<mapping_name>[^/]+)$`,
	}

	var appName, profileName, attributeType, mappingName string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if i == 0 {
					continue
				}
				switch matchName {
				case "app_name":
					appName = matches[i]
				case "profile_name":
					profileName = matches[i]
				case "attribute_type":
					attributeType = matches[i]
				case "mapping_name":
					mappingName = matches[i]
				}
			}
			break
		}
	}

	if appName == "" || profileName == "" || attributeType == "" || mappingName == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats", req.ID),
		)
		return
	}

	// Validate fields
	if strings.TrimSpace(appName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "app_name cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(profileName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "profile_name cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(attributeType) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "attribute_type cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(mappingName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "mapping_name cannot be empty or whitespace")
		return
	}

	// Get application by name
	app, err := r.client.GetApplicationByName(appName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Application Not Found",
			fmt.Sprintf("Application '%s' not found.", appName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Session Attribute",
			fmt.Sprintf("Could not get application '%s': %s", appName, err.Error()),
		)
		return
	}

	// Get profile by name
	profile, err := r.client.GetProfileByName(app.AppContainerID, profileName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Profile Not Found",
			fmt.Sprintf("Profile '%s' not found.", profileName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Session Attribute",
			fmt.Sprintf("Could not get profile '%s': %s", profileName, err.Error()),
		)
		return
	}

	// Get session attribute
	sessionAttribute, err := r.client.GetProfileSessionAttributeByTypeAndMappingName(profile.ProfileID, attributeType, mappingName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Session Attribute Not Found",
			fmt.Sprintf("Session attribute with type '%s' and mapping name '%s' not found.", attributeType, mappingName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Session Attribute",
			fmt.Sprintf("Could not get session attribute: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateProfileSessionAttributeID(profile.ProfileID, sessionAttribute.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profile.ProfileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mapping_name"), mappingName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("attribute_type"), attributeType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("transitive"), sessionAttribute.Transitive)...)

	// Handle attribute_name and attribute_value based on type
	if strings.EqualFold(attributeType, "Identity") && sessionAttribute.AttributeSchemaID != "" {
		attribute, err := r.client.GetAttribute(sessionAttribute.AttributeSchemaID)
		if err == nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("attribute_name"), attribute.Name)...)
		}
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("attribute_value"), sessionAttribute.AttributeValue)...)
	}
	// Note: app_name and profile_name are not set (cleared like in SDKv2 version)
}

// Helper methods
func (r *ProfileSessionAttributeResource) buildSessionAttribute(plan ProfileSessionAttributeResourceModel) (*britive.SessionAttribute, error) {
	attributeName := plan.AttributeName.ValueString()
	attributeType := plan.AttributeType.ValueString()
	attributeValue := plan.AttributeValue.ValueString()
	mappingName := plan.MappingName.ValueString()
	transitive := plan.Transitive.ValueBool()

	if attributeType == "" {
		attributeType = "Identity"
	}

	var attributeSchemaID string
	var attrValue string

	if strings.EqualFold(attributeType, "Identity") {
		if attributeName == "" {
			return nil, fmt.Errorf("attribute_name is required when attribute_type is Identity")
		}
		if attributeValue != "" {
			return nil, fmt.Errorf("attribute_value must be empty when attribute_type is Identity")
		}

		attribute, err := r.client.GetAttributeByName(attributeName)
		if err != nil {
			return nil, fmt.Errorf("could not get attribute '%s': %w", attributeName, err)
		}
		attributeSchemaID = attribute.ID
	} else {
		if attributeValue == "" {
			return nil, fmt.Errorf("attribute_value is required when attribute_type is Static")
		}
		if attributeName != "" {
			return nil, fmt.Errorf("attribute_name must be empty when attribute_type is Static")
		}
		attrValue = attributeValue
	}

	return &britive.SessionAttribute{
		AttributeSchemaID:    attributeSchemaID,
		MappingName:          mappingName,
		Transitive:           transitive,
		SessionAttributeType: attributeType,
		AttributeValue:       attrValue,
	}, nil
}

func (r *ProfileSessionAttributeResource) populateStateFromAPI(ctx context.Context, state *ProfileSessionAttributeResourceModel) error {
	profileID, sessionAttributeID, err := parseProfileSessionAttributeID(state.ID.ValueString())
	if err != nil {
		return err
	}

	sessionAttr, err := r.client.GetProfileSessionAttribute(profileID, sessionAttributeID)
	if err != nil {
		return err
	}

	state.ProfileID = types.StringValue(profileID)
	state.AttributeType = types.StringValue(sessionAttr.SessionAttributeType)
	state.MappingName = types.StringValue(sessionAttr.MappingName)
	state.Transitive = types.BoolValue(sessionAttr.Transitive)

	// Handle attribute_name and attribute_value based on type
	if strings.EqualFold(sessionAttr.SessionAttributeType, "Identity") || sessionAttr.AttributeSchemaID != "" {
		attribute, err := r.client.GetAttribute(sessionAttr.AttributeSchemaID)
		if err != nil {
			return err
		}
		state.AttributeName = types.StringValue(attribute.Name)
		state.AttributeValue = types.StringNull() // Clear value for Identity type
	} else {
		state.AttributeValue = types.StringValue(sessionAttr.AttributeValue)
		state.AttributeName = types.StringNull() // Clear name for Static type
	}

	return nil
}

// Helper functions
func generateProfileSessionAttributeID(profileID, attributeID string) string {
	return fmt.Sprintf("paps/%s/session-attributes/%s", profileID, attributeID)
}

func parseProfileSessionAttributeID(id string) (profileID, attributeID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 4 {
		err = fmt.Errorf("invalid profile session attribute ID format: %s", id)
		return
	}
	profileID = parts[1]
	attributeID = parts[3]
	return
}
