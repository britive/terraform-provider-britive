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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TagResource{}
	_ resource.ResourceWithConfigure   = &TagResource{}
	_ resource.ResourceWithImportState = &TagResource{}
)

func NewTagResource() resource.Resource {
	return &TagResource{}
}

type TagResource struct {
	client *britive.Client
}

type TagResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Disabled           types.Bool   `tfsdk:"disabled"`
	IdentityProviderID types.String `tfsdk:"identity_provider_id"`
	External           types.Bool   `tfsdk:"external"`
}

func (r *TagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive tag.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the tag.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of Britive tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive tag.",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "To disable the Britive tag.",
			},
			"identity_provider_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identity of the identity provider associated with the Britive tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"external": schema.BoolAttribute{
				Computed:    true,
				Description: "The boolean attribute that indicates whether the tag is external or not.",
			},
		},
	}
}

func (r *TagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate for external tag
	if err := r.validateForExternalTag(plan.IdentityProviderID.ValueString(), plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Tag Configuration",
			fmt.Sprintf("Cannot create tag: %s", err.Error()),
		)
		return
	}

	// Create tag
	tag := britive.Tag{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if plan.Disabled.ValueBool() {
		tag.Status = "Inactive"
	} else {
		tag.Status = "Active"
	}

	tag.UserTagIdentityProviders = []britive.UserTagIdentityProvider{
		{
			IdentityProvider: britive.IdentityProvider{
				ID: plan.IdentityProviderID.ValueString(),
			},
		},
	}

	createdTag, err := r.client.CreateTag(tag)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Tag",
			fmt.Sprintf("Could not create tag: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(createdTag.ID)

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Tag",
			fmt.Sprintf("Could not read tag after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate for external tag
	if err := r.validateForExternalTag(state.IdentityProviderID.ValueString(), state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Tag Configuration",
			fmt.Sprintf("Cannot read tag: %s", err.Error()),
		)
		return
	}

	tagID := state.ID.ValueString()
	tag, err := r.client.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Tag",
			fmt.Sprintf("Could not read tag %s: %s", tagID, err.Error()),
		)
		return
	}

	state.Name = types.StringValue(tag.Name)
	state.Description = types.StringValue(tag.Description)
	state.Disabled = types.BoolValue(strings.EqualFold(tag.Status, "Inactive"))
	state.External = types.BoolValue(tag.External.(bool))

	if len(tag.UserTagIdentityProviders) > 0 {
		state.IdentityProviderID = types.StringValue(tag.UserTagIdentityProviders[0].IdentityProvider.ID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// populateStateFromAPI fetches tag data from API and populates the state model
func (r *TagResource) populateStateFromAPI(ctx context.Context, state *TagResourceModel) error {
	// Validate for external tag
	if err := r.validateForExternalTag(state.IdentityProviderID.ValueString(), state.Name.ValueString()); err != nil {
		return err
	}

	tagID := state.ID.ValueString()
	tag, err := r.client.GetTag(tagID)
	if err != nil {
		return err
	}

	state.Name = types.StringValue(tag.Name)
	state.Description = types.StringValue(tag.Description)
	state.Disabled = types.BoolValue(strings.EqualFold(tag.Status, "Inactive"))
	state.External = types.BoolValue(tag.External.(bool))

	if len(tag.UserTagIdentityProviders) > 0 {
		state.IdentityProviderID = types.StringValue(tag.UserTagIdentityProviders[0].IdentityProvider.ID)
	}

	return nil
}

func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate for external tag
	if err := r.validateForExternalTag(plan.IdentityProviderID.ValueString(), plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Tag Configuration",
			fmt.Sprintf("Cannot update tag: %s", err.Error()),
		)
		return
	}

	tagID := plan.ID.ValueString()

	// Update name and description
	tag := britive.Tag{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	updatedTag, err := r.client.UpdateTag(tagID, tag)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Tag",
			fmt.Sprintf("Could not update tag %s: %s", tagID, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updatedTag.ID)

	// Update disabled status separately
	disabled := plan.Disabled.ValueBool()
	_, err = r.client.EnableOrDisableTag(tagID, disabled)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Tag Status",
			fmt.Sprintf("Could not update status for tag %s: %s", tagID, err.Error()),
		)
		return
	}

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Tag",
			fmt.Sprintf("Could not read tag after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate for external tag
	if err := r.validateForExternalTag(state.IdentityProviderID.ValueString(), state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Tag Configuration",
			fmt.Sprintf("Cannot delete tag: %s", err.Error()),
		)
		return
	}

	tagID := state.ID.ValueString()
	err := r.client.DeleteTag(tagID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Tag",
			fmt.Sprintf("Could not delete tag %s: %s", tagID, err.Error()),
		)
		return
	}
}

func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. tags/{name}
	// 2. {name}
	idRegexes := []string{`^tags/(?P<name>[^/]+)$`, `^(?P<name>[^/]+)$`}

	var tagName string
	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, name := range re.SubexpNames() {
				if name == "name" && i < len(matches) {
					tagName = matches[i]
					break
				}
			}
			if tagName != "" {
				break
			}
		}
	}

	if tagName == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'tags/{name}' or '{name}'", req.ID),
		)
		return
	}

	if strings.TrimSpace(tagName) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Tag name cannot be empty or whitespace.",
		)
		return
	}

	// Get tag by name
	tag, err := r.client.GetTagByName(tagName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Tag Not Found",
			fmt.Sprintf("Tag '%s' not found.", tagName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Tag",
			fmt.Sprintf("Could not import tag '%s': %s", tagName, err.Error()),
		)
		return
	}

	// Check if tag is external
	if external, ok := tag.External.(bool); ok && external {
		resp.Diagnostics.AddError(
			"Cannot Import External Tag",
			fmt.Sprintf("Importing external tags is not supported. Attempted to import tag '%s'.", tagName),
		)
		return
	}

	// Set the ID and name so Read can populate the rest
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), tag.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), tag.Name)...)

	if len(tag.UserTagIdentityProviders) > 0 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_provider_id"), tag.UserTagIdentityProviders[0].IdentityProvider.ID)...)
	}
}

func (r *TagResource) validateForExternalTag(identityProviderID, tagName string) error {
	if identityProviderID == "" {
		return nil
	}

	identityProvider, err := r.client.GetIdentityProvider(identityProviderID)
	if err != nil {
		return err
	}

	if !strings.EqualFold(identityProvider.Type, "DEFAULT") {
		return fmt.Errorf("managing external tags is not supported. Attempted to manage tag '%s'", tagName)
	}

	return nil
}
