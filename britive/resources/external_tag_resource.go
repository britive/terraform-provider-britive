package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ExternalTagResource{}
	_ resource.ResourceWithConfigure   = &ExternalTagResource{}
	_ resource.ResourceWithImportState = &ExternalTagResource{}
	_ resource.ResourceWithModifyPlan  = &ExternalTagResource{}
)

// Error codes the Britive API returns for operations that are not allowed on a
// connected external tag. They are matched by prefix because the client renders
// API errors as "<code>: <message>".
const (
	errCodeTagAlreadyExists   = "UT-0001"
	errCodeExternalTagAddUser = "UT-0004"
	errCodeExternalTagDelete  = "UT-0009"
	errCodeExternalTagUpdate  = "UT-0010"
)

func NewExternalTagResource() resource.Resource {
	return &ExternalTagResource{}
}

// ExternalTagResource manages a tag that belongs to a non-default identity
// provider. Terraform creates the tag as a placeholder; once the identity
// provider connects a same-name group through SCIM the tag becomes "connected"
// (external = true) and the Britive API reserves name, status, membership and
// deletion for the identity provider.
type ExternalTagResource struct {
	client *britive.Client
}

type ExternalTagResourceModel struct {
	ID                 types.String        `tfsdk:"id"`
	Name               types.String        `tfsdk:"name"`
	Description        types.String        `tfsdk:"description"`
	IdentityProviderID types.String        `tfsdk:"identity_provider_id"`
	Requestable        types.Bool          `tfsdk:"requestable"`
	External           types.Bool          `tfsdk:"external"`
	ExternalID         types.String        `tfsdk:"external_id"`
	Attributes         []TagAttributeModel `tfsdk:"attributes"`
}

func (r *ExternalTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_tag"
}

func (r *ExternalTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an external tag: a tag on a non-default identity provider that the identity provider " +
			"connects through SCIM. Terraform creates the placeholder and keeps managing requestable, attributes " +
			"and description; once connected, name, status, membership and deletion belong to the identity provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the tag.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "The name of the tag. Must match the group name the identity provider will provision. " +
					"Cannot be changed once the tag is connected.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Description: "The description of the tag. Once the tag is connected, it can only be changed when the " +
					"identity provider's \"Allow tag name and description to be editable\" setting is on.",
			},
			"identity_provider_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identity of the non-default identity provider that owns the tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"requestable": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the tag is requestable. Writable in both the placeholder and the connected state.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external": schema.BoolAttribute{
				Computed: true,
				Description: "True once the identity provider has connected the tag through SCIM. " +
					"Until then the tag is a placeholder with the full lifecycle.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_id": schema.StringAttribute{
				Computed:    true,
				Description: "The identity provider's identifier for the connected group, when the provider supplies one.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"attributes": schema.SetNestedBlock{
				Description: "The attributes of the tag. Writable in both the placeholder and the connected state.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"attribute_name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the attribute.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"attribute_value": schema.StringAttribute{
							Required:    true,
							Description: "The value of the attribute.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ExternalTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan enforces the rules the Britive API applies to connected external
// tags before apply, so the user sees the reason instead of a 400:
//   - identity_provider_id must not be the default provider (use britive_tag)
//   - name cannot change once connected
//   - description can change once connected only when the identity provider
//     allows editing external tags
//
// Only scalar attributes are read here; the attributes block may be an unknown
// collection during plan and must not be decoded into the slice model.
func (r *ExternalTagResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		// Destroy, or provider not configured (terraform validate).
		return
	}

	var planIdP types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("identity_provider_id"), &planIdP)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !planIdP.IsNull() && !planIdP.IsUnknown() {
		idp, err := r.client.GetIdentityProvider(planIdP.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Identity Provider",
				fmt.Sprintf("Could not read identity provider %s: %s", planIdP.ValueString(), err.Error()),
			)
			return
		}
		if strings.EqualFold(idp.Type, "DEFAULT") {
			resp.Diagnostics.AddAttributeError(
				path.Root("identity_provider_id"),
				"Not An External Tag",
				fmt.Sprintf("Identity provider '%s' is Britive's default provider. Tags on it are regular tags; use the britive_tag resource instead.", idp.Name),
			)
			return
		}
	}

	if req.State.Raw.IsNull() {
		// Create. The common order is identity provider first: SCIM has already
		// created the tag and Create would fail with UT-0001. Catch it here and
		// tell the user to import instead.
		var planName types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &planName)...)
		if resp.Diagnostics.HasError() || planName.IsUnknown() || planIdP.IsNull() || planIdP.IsUnknown() {
			return
		}
		if existing := r.findExistingTag(planName.ValueString(), planIdP.ValueString()); existing != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"External Tag Already Exists",
				existingTagGuidance(existing),
			)
		}
		return
	}

	var stateExternal types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("external"), &stateExternal)...)
	if resp.Diagnostics.HasError() || stateExternal.IsNull() || !stateExternal.ValueBool() {
		return
	}

	var stateName, planName, stateDesc, planDesc types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("name"), &stateName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &planName)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("description"), &stateDesc)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("description"), &planDesc)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !planName.IsUnknown() && planName.ValueString() != stateName.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Cannot Rename Connected External Tag",
			fmt.Sprintf("Tag '%s' is connected to its identity provider through SCIM; the identity provider owns its name. "+
				"Rename the group in the identity provider instead, or keep name = %q.", stateName.ValueString(), stateName.ValueString()),
		)
		return
	}

	if planDesc.IsUnknown() || descriptionsEqual(planDesc, stateDesc) {
		return
	}

	if planIdP.IsNull() || planIdP.IsUnknown() {
		return
	}
	idp, err := r.client.GetIdentityProvider(planIdP.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Identity Provider",
			fmt.Sprintf("Could not read identity provider %s: %s", planIdP.ValueString(), err.Error()),
		)
		return
	}
	if !idp.AllowEditingExternalTags {
		resp.Diagnostics.AddAttributeError(
			path.Root("description"),
			"Cannot Edit Connected External Tag",
			fmt.Sprintf("Tag '%s' is connected to identity provider '%s', which does not allow editing external tags. "+
				"Turn on \"Allow tag name and description to be editable\" on the identity provider's SCIM tab, or keep the current description.",
				stateName.ValueString(), idp.Name),
		)
	}
}

// apiDefaultDescription is what the Britive API stores and returns when a tag
// is created without a description.
const apiDefaultDescription = "No Description"

// descriptionsEqual treats null, "" and the API's default placeholder as the
// same value, so an omitted description never counts as a change.
func descriptionsEqual(a, b types.String) bool {
	return normalizeDescription(a) == normalizeDescription(b)
}

func normalizeDescription(v types.String) string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == apiDefaultDescription {
		return ""
	}
	return v.ValueString()
}

func (r *ExternalTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExternalTagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.requireNonDefaultIdentityProvider(plan.IdentityProviderID.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("identity_provider_id"), "Not An External Tag", err.Error())
		return
	}

	tag := britive.Tag{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Status:      "Active",
		UserTagIdentityProviders: []britive.UserTagIdentityProvider{
			{IdentityProvider: britive.IdentityProvider{ID: plan.IdentityProviderID.ValueString()}},
		},
	}

	createdTag, err := r.client.CreateTag(tag)
	if err != nil && strings.HasPrefix(err.Error(), errCodeTagAlreadyExists) {
		// Created by SCIM between plan and apply.
		if existing := r.findExistingTag(plan.Name.ValueString(), plan.IdentityProviderID.ValueString()); existing != nil {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "External Tag Already Exists", existingTagGuidance(existing))
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating External Tag",
			fmt.Sprintf("Could not create tag '%s': %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(createdTag.ID)

	if !plan.Requestable.IsNull() && !plan.Requestable.IsUnknown() || len(plan.Attributes) > 0 {
		if err := r.applyOverlay(createdTag.ID, &plan); err != nil {
			// Roll back the placeholder so the failed create leaves nothing behind
			// and the next apply creates the tag again. Keeping it as a tainted
			// resource does not work: Terraform plans the replacement as a fresh
			// create, and ModifyPlan would refuse it with "External Tag Already Exists".
			if delErr := r.client.DeleteTag(createdTag.ID); delErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating External Tag",
					fmt.Sprintf("Tag '%s' was created but its requestable flag or attributes could not be set: %s\n\n"+
						"Removing the partially created tag (%s) also failed: %s. Delete it in Britive, or import it, before applying again.",
						plan.Name.ValueString(), r.explainExternalError(err), createdTag.ID, delErr.Error()),
				)
				return
			}
			resp.Diagnostics.AddError(
				"Error Creating External Tag",
				fmt.Sprintf("Tag '%s' could not be created because its requestable flag or attributes could not be set: %s\n\n"+
					"The partially created tag was removed, so the next apply starts clean.",
					plan.Name.ValueString(), r.explainExternalError(err)),
			)
			return
		}
	}

	if err := r.populateFromAPI(&plan, false); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading External Tag",
			fmt.Sprintf("Could not read tag after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExternalTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ExternalTagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := state.ID.ValueString()
	tag, err := r.client.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		if !state.External.IsNull() && state.External.ValueBool() {
			resp.Diagnostics.AddWarning(
				"Connected External Tag Removed By Identity Provider",
				fmt.Sprintf("Tag '%s' (%s) no longer exists in Britive. It was connected to its identity provider, so the "+
					"identity provider most likely deleted the group. The resource has been removed from state; the next plan "+
					"will recreate it as a placeholder unless you remove it from the configuration.", state.Name.ValueString(), tagID),
			)
		}
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading External Tag",
			fmt.Sprintf("Could not read tag %s: %s", tagID, err.Error()),
		)
		return
	}

	mapExternalTagToModel(tag, &state, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ExternalTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ExternalTagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := state.ID.ValueString()
	plan.ID = state.ID

	// Name and description live on PATCH /user-tags/{id}. Only call it when one of
	// them changed: on a connected tag that endpoint is gated by the identity
	// provider's edit flag (ModifyPlan has already checked it), and it also clears
	// attributes as a side effect, which applyOverlay restores right after.
	if plan.Name.ValueString() != state.Name.ValueString() || !descriptionsEqual(plan.Description, state.Description) {
		_, err := r.client.UpdateTag(tagID, britive.Tag{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating External Tag",
				fmt.Sprintf("Could not update tag %s: %s", tagID, r.explainExternalError(err)),
			)
			return
		}
	}

	if err := r.applyOverlay(tagID, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating External Tag Attributes",
			fmt.Sprintf("Could not update attributes of tag %s: %s", tagID, r.explainExternalError(err)),
		)
		return
	}

	if err := r.populateFromAPI(&plan, false); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading External Tag",
			fmt.Sprintf("Could not read tag after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a placeholder. A connected tag belongs to the identity
// provider and the API refuses to delete it (UT-0009); in that case the tag is
// left in place and only removed from state, with a warning, the same way
// aws_default_vpc and aws_vpc_peering_connection_accepter behave.
func (r *ExternalTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ExternalTagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := state.ID.ValueString()

	tag, err := r.client.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading External Tag",
			fmt.Sprintf("Could not read tag %s before deletion: %s", tagID, err.Error()),
		)
		return
	}

	if tag.External {
		r.warnConnectedNotDeleted(resp, tag)
		return
	}

	err = r.client.DeleteTag(tagID)
	if err != nil && strings.HasPrefix(err.Error(), errCodeExternalTagDelete) {
		// Connected between our read and the delete call.
		r.warnConnectedNotDeleted(resp, tag)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting External Tag",
			fmt.Sprintf("Could not delete tag %s: %s", tagID, err.Error()),
		)
		return
	}
}

func (r *ExternalTagResource) warnConnectedNotDeleted(resp *resource.DeleteResponse, tag *britive.Tag) {
	idpName := ""
	if len(tag.UserTagIdentityProviders) > 0 {
		idpName = tag.UserTagIdentityProviders[0].IdentityProvider.Name
	}
	resp.Diagnostics.AddWarning(
		"Connected External Tag Not Deleted",
		fmt.Sprintf("Tag '%s' (%s) is connected to identity provider '%s' through SCIM and cannot be deleted from Britive. "+
			"It has been removed from Terraform state but still exists. Delete the group in the identity provider to remove it. "+
			"If the resource is still in the configuration, remove it too; otherwise the next plan will report that the tag "+
			"already exists and ask you to import it.",
			tag.Name, tag.ID, idpName),
	)
}

// ImportState accepts the tag id, or the tag name when it is unique across
// identity providers.
func (r *ExternalTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := strings.TrimSpace(req.ID)
	if importID == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be the tag id or the tag name.")
		return
	}

	tag, err := r.client.GetTag(importID)
	if errors.Is(err, britive.ErrNotFound) {
		tags, nameErr := r.client.GetTagsByName(importID)
		if errors.Is(nameErr, britive.ErrNotFound) {
			resp.Diagnostics.AddError("Tag Not Found", fmt.Sprintf("No tag with id or name '%s' was found.", importID))
			return
		}
		if nameErr != nil {
			resp.Diagnostics.AddError("Error Importing External Tag", nameErr.Error())
			return
		}
		if len(tags) > 1 {
			resp.Diagnostics.AddError(
				"Ambiguous Tag Name",
				fmt.Sprintf("%d tags are named '%s'. Import by tag id instead:\n%s", len(tags), importID, describeTagCandidates(tags)),
			)
			return
		}
		tag = &tags[0]
	} else if err != nil {
		resp.Diagnostics.AddError("Error Importing External Tag", err.Error())
		return
	}

	if isDefaultIdentityProviderTag(tag) {
		resp.Diagnostics.AddError(
			"Not An External Tag",
			fmt.Sprintf("Tag '%s' is on Britive's default identity provider. Import it with the britive_tag resource instead.", tag.Name),
		)
		return
	}

	var state ExternalTagResourceModel
	state.ID = types.StringValue(tag.ID)
	mapExternalTagToModel(tag, &state, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyOverlay writes the fields Britive lets callers own in both states:
// requestable and attributes. The id-in-body PATCH must also carry name and
// description; they are copied from the tag as it currently is so that the
// call never counts as a name/description edit, which a connected tag whose
// identity provider forbids editing would reject with UT-0010 (an empty
// description differs from the API's stored "No Description" default).
func (r *ExternalTagResource) applyOverlay(tagID string, plan *ExternalTagResourceModel) error {
	current, err := r.client.GetTag(tagID)
	if err != nil {
		return fmt.Errorf("could not read tag before updating attributes: %w", err)
	}
	reqBody := britive.TagAttributesUpdateRequest{
		Name:        current.Name,
		Description: current.Description,
		Attributes:  buildTagAttributes(plan.Attributes),
	}
	// Always send requestable: the PATCH overwrites it, so leaving it out would
	// reset it. Without a configured value, keep the tag's current one.
	requestable := current.Requestable
	if !plan.Requestable.IsNull() && !plan.Requestable.IsUnknown() {
		requestable = plan.Requestable.ValueBool()
	}
	reqBody.Requestable = &requestable
	_, err = r.client.UpdateTagAttributes(tagID, reqBody)
	return err
}

func (r *ExternalTagResource) populateFromAPI(state *ExternalTagResourceModel, fromRead bool) error {
	tag, err := r.client.GetTag(state.ID.ValueString())
	if err != nil {
		return err
	}
	mapExternalTagToModel(tag, state, fromRead)
	return nil
}

// mapExternalTagToModel copies API values into the model. After Create/Update
// (fromRead = false) a null description in the plan stays null even though the
// API reports its default, to keep the apply result consistent with the plan.
func mapExternalTagToModel(tag *britive.Tag, state *ExternalTagResourceModel, fromRead bool) {
	state.ID = types.StringValue(tag.ID)
	state.Name = types.StringValue(tag.Name)
	switch {
	case tag.Description == "" || tag.Description == apiDefaultDescription:
		// Absent in the API: keep whatever the config said (null or ""), so the
		// API's default placeholder never shows up as drift.
		if !state.Description.IsNull() && state.Description.ValueString() != apiDefaultDescription {
			state.Description = types.StringValue("")
		} else {
			state.Description = types.StringNull()
		}
	case fromRead || !state.Description.IsNull():
		state.Description = types.StringValue(tag.Description)
	}
	state.Requestable = types.BoolValue(tag.Requestable)
	state.External = types.BoolValue(tag.External)
	state.Attributes = mapTagAttributesToModel(tag.Attributes)

	state.ExternalID = types.StringValue("")
	if len(tag.UserTagIdentityProviders) > 0 {
		state.IdentityProviderID = types.StringValue(tag.UserTagIdentityProviders[0].IdentityProvider.ID)
		if ext, ok := tag.UserTagIdentityProviders[0].ExternalID.(string); ok {
			state.ExternalID = types.StringValue(ext)
		}
	}
}

// findExistingTag returns the tag with this name on this identity provider, or
// nil when there is none (or the lookup fails; Create will then surface the
// real error).
func (r *ExternalTagResource) findExistingTag(name, identityProviderID string) *britive.Tag {
	tags, err := r.client.GetTagsByName(name)
	if err != nil {
		return nil
	}
	for i := range tags {
		if len(tags[i].UserTagIdentityProviders) > 0 && tags[i].UserTagIdentityProviders[0].IdentityProvider.ID == identityProviderID {
			return &tags[i]
		}
	}
	return nil
}

// existingTagGuidance explains that the tag already exists and how to adopt it.
func existingTagGuidance(tag *britive.Tag) string {
	idpName := ""
	if len(tag.UserTagIdentityProviders) > 0 {
		idpName = tag.UserTagIdentityProviders[0].IdentityProvider.Name
	}
	state := "a placeholder that has not been connected yet"
	if tag.External {
		state = "connected to the identity provider through SCIM"
	}
	return fmt.Sprintf("A tag named '%s' already exists on identity provider '%s' (id %s) and is %s. "+
		"Terraform does not take over existing tags on create. To manage it, import it, either with\n\n"+
		"  terraform import <resource address> %s\n\n"+
		"or with an import block in the configuration:\n\n"+
		"  import {\n    to = <resource address>\n    id = %q\n  }\n\n"+
		"If Terraform should no longer manage this tag (for example after a destroy that left a connected tag in "+
		"Britive), remove the resource from the configuration instead.",
		tag.Name, idpName, tag.ID, state, tag.ID, tag.ID)
}

func (r *ExternalTagResource) requireNonDefaultIdentityProvider(identityProviderID string) error {
	idp, err := r.client.GetIdentityProvider(identityProviderID)
	if err != nil {
		return fmt.Errorf("could not read identity provider %s: %w", identityProviderID, err)
	}
	if strings.EqualFold(idp.Type, "DEFAULT") {
		return fmt.Errorf("identity provider '%s' is Britive's default provider. Tags on it are regular tags; use the britive_tag resource instead", idp.Name)
	}
	return nil
}

// explainExternalError appends guidance to the API's external-tag error codes.
func (r *ExternalTagResource) explainExternalError(err error) string {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, errCodeExternalTagUpdate) && strings.Contains(strings.ToLower(msg), "requestable"):
		// Same code, different rule: a tenant setting, not the SCIM edit flag.
		return msg + " (requestable can only be changed when tag access requests are enabled for the tenant in Britive; " +
			"enable it, or remove requestable from the configuration to keep the tag's current value)"
	case strings.HasPrefix(msg, errCodeExternalTagUpdate):
		return msg + " (the tag is connected through SCIM; turn on \"Allow tag name and description to be editable\" on the identity provider to change name or description)"
	case strings.HasPrefix(msg, errCodeExternalTagAddUser):
		return msg + " (membership of a connected external tag is managed by the identity provider)"
	default:
		return msg
	}
}

// describeTagCandidates lists tags as "id (identity provider)" lines for
// ambiguity errors.
func describeTagCandidates(tags []britive.Tag) string {
	lines := make([]string, 0, len(tags))
	for _, t := range tags {
		idp := "unknown identity provider"
		if len(t.UserTagIdentityProviders) > 0 {
			idp = fmt.Sprintf("%s, %s", t.UserTagIdentityProviders[0].IdentityProvider.Name, t.UserTagIdentityProviders[0].IdentityProvider.ID)
		}
		kind := "placeholder"
		if t.External {
			kind = "connected"
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s; %s)", t.ID, idp, kind))
	}
	return strings.Join(lines, "\n")
}
