package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ApplicationApproversGroupResource{}
	_ resource.ResourceWithConfigure      = &ApplicationApproversGroupResource{}
	_ resource.ResourceWithImportState    = &ApplicationApproversGroupResource{}
	_ resource.ResourceWithValidateConfig = &ApplicationApproversGroupResource{}
)

// ApplicationApproversGroupResource manages an approvers group under an application. Approvers
// groups are referenced by britive_application_access_builder_settings' association_approver
// blocks.
type ApplicationApproversGroupResource struct {
	client *britive.Client
}

// ApplicationApproversGroupResourceModel describes the resource data model.
type ApplicationApproversGroupResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	GroupID       types.String `tfsdk:"group_id"`
	Name          types.String `tfsdk:"name"`
	Condition     types.String `tfsdk:"condition"`
	// A types.Set, not a plain Go slice: a "dynamic" member block inside a resource that uses
	// for_each is wholly unknown during validate/plan, which a Go slice can't represent. Convert
	// with accessRequestMembersFromSet/accessRequestMembersToSet. Mirrors
	// ApplicationScanScheduleResourceModel.Associations.
	Members types.Set `tfsdk:"member"`
}

// accessRequestMemberModel is a user or tag, referenced by either id or name. Used by
// approvers group member blocks.
type accessRequestMemberModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// accessRequestMemberObjectType mirrors accessRequestMemberModel's tfsdk tags.
var accessRequestMemberObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type": types.StringType,
		"id":   types.StringType,
		"name": types.StringType,
	},
}

// accessRequestMembersFromSet decodes a member set. Returns nil without error for a
// null/unknown set.
func accessRequestMembersFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []accessRequestMemberModel {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var members []accessRequestMemberModel
	diags.Append(set.ElementsAs(ctx, &members, false)...)
	return members
}

// accessRequestMembersToSet is accessRequestMembersFromSet's inverse.
func accessRequestMembersToSet(ctx context.Context, members []accessRequestMemberModel, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, accessRequestMemberObjectType, members)
	diags.Append(d...)
	return set
}

// NewApplicationApproversGroupResource is a helper function to simplify the provider implementation.
func NewApplicationApproversGroupResource() resource.Resource {
	return &ApplicationApproversGroupResource{}
}

// Metadata returns the resource type name.
func (r *ApplicationApproversGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_approvers_group"
}

// Schema defines the schema for the resource.
func (r *ApplicationApproversGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an approvers group for a Britive application's access requests.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the approvers group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the application.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The unique identifier of the approvers group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the approvers group.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"condition": schema.StringAttribute{
				Description: "Whether All or Any of the members must approve. One of All, Any.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("All", "Any"),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"member": schema.SetNestedBlock{
				Description: "A user or tag in the approvers group. At least one is required.",
				NestedObject: schema.NestedBlockObject{
					Attributes: accessRequestMemberAttributes(),
				},
			},
		},
	}
}

// accessRequestMemberAttributes returns the type/id/name attributes shared by member blocks.
func accessRequestMemberAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"type": schema.StringAttribute{
			Description: "The member type. One of User, Tag.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("User", "Tag"),
			},
		},
		"id": schema.StringAttribute{
			Description: "The ID of the user or tag. Exactly one of id or name must be set.",
			Optional:    true,
		},
		"name": schema.StringAttribute{
			Description: "The username of the user, or the name of the tag. Exactly one of id or name must be set.",
			Optional:    true,
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ApplicationApproversGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig requires at least one member block, and rejects any member block that sets
// both or neither of id and name.
func (r *ApplicationApproversGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ApplicationApproversGroupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateBlockRequired(data.Members, path.Root("member"), "member", &resp.Diagnostics)
	for _, m := range accessRequestMembersFromSet(ctx, data.Members, &resp.Diagnostics) {
		validateIDOrName(m.ID, m.Name, path.Root("member"), "member", &resp.Diagnostics)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ApplicationApproversGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationApproversGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueString()

	members, err := r.resolveMembers(ctx, plan.Members, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Approvers Group Members", err.Error())
		return
	}

	log.Printf("[INFO] Creating approvers group %s for application: %s", plan.Name.ValueString(), applicationID)

	created, err := r.client.CreateApproversGroup(applicationID, britive.ApproversGroup{
		Name:      plan.Name.ValueString(),
		Condition: plan.Condition.ValueString(),
		Members:   members,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Approvers Group", err.Error())
		return
	}

	plan.GroupID = types.StringValue(created.ID)
	plan.ID = types.StringValue(approversGroupCompositeID(applicationID, created.ID))

	log.Printf("[INFO] Created approvers group: %s", created.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ApplicationApproversGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationApproversGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	groupID := state.GroupID.ValueString()

	log.Printf("[INFO] Reading approvers group: %s", groupID)

	group, err := r.client.GetApproversGroup(applicationID, groupID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Approvers Group", err.Error())
		return
	}

	prior := accessRequestMembersFromSet(ctx, state.Members, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.mapModelToResource(ctx, group, &state, prior, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ApplicationApproversGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ApplicationApproversGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	groupID := state.GroupID.ValueString()

	patch := britive.ApproversGroupPatch{}
	if !plan.Name.Equal(state.Name) {
		patch.Name = plan.Name.ValueString()
	}
	if !plan.Condition.Equal(state.Condition) {
		patch.Condition = plan.Condition.ValueString()
	}
	if !plan.Members.Equal(state.Members) {
		members, err := r.resolveMembers(ctx, plan.Members, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if err != nil {
			resp.Diagnostics.AddError("Error Resolving Approvers Group Members", err.Error())
			return
		}
		patch.Members = &members
	}

	log.Printf("[INFO] Updating approvers group: %s", groupID)

	if err := r.client.PatchApproversGroup(applicationID, groupID, patch); err != nil {
		resp.Diagnostics.AddError("Error Updating Approvers Group", err.Error())
		return
	}

	plan.ID = state.ID
	plan.GroupID = state.GroupID

	log.Printf("[INFO] Updated approvers group: %s", groupID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ApplicationApproversGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationApproversGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.GroupID.ValueString()

	log.Printf("[INFO] Deleting approvers group: %s", groupID)

	if err := r.client.DeleteApproversGroup(state.ApplicationID.ValueString(), groupID); err != nil {
		detail := err.Error()
		if strings.Contains(detail, "mapped") {
			detail += "\n\nRemove this approvers group from every association_approver block of britive_application_access_builder_settings first."
		}
		resp.Diagnostics.AddError("Error Deleting Approvers Group", detail)
		return
	}

	log.Printf("[INFO] Deleted approvers group: %s", groupID)
}

// ImportState imports the resource state.
func (r *ApplicationApproversGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	applicationID, groupID, err := parseApproversGroupCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	log.Printf("[INFO] Importing approvers group: %s", groupID)

	group, err := r.client.GetApproversGroup(applicationID, groupID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Approvers Group Not Found", fmt.Sprintf("Approvers group %s not found under application %s", groupID, applicationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Approvers Group", err.Error())
		return
	}

	var state ApplicationApproversGroupResourceModel
	state.ID = types.StringValue(approversGroupCompositeID(applicationID, groupID))
	state.ApplicationID = types.StringValue(applicationID)
	state.GroupID = types.StringValue(groupID)
	r.mapModelToResource(ctx, group, &state, nil, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// mapModelToResource maps the API's approvers group onto state. Each member keeps the id or
// name form used in prior; members not in prior (import, or added outside Terraform) use id.
func (r *ApplicationApproversGroupResource) mapModelToResource(ctx context.Context, group *britive.ApproversGroup, state *ApplicationApproversGroupResourceModel, prior []accessRequestMemberModel, diags *diag.Diagnostics) {
	state.Name = types.StringValue(group.Name)
	state.Condition = types.StringValue(group.Condition)

	priorRefs := make([]accessRequestRef, 0, len(prior))
	for _, p := range prior {
		priorRefs = append(priorRefs, accessRequestRef{Type: p.Type.ValueString(), ID: p.ID, Name: p.Name})
	}

	resolve := cachedUserOrTagResolver(r.client)
	members := make([]accessRequestMemberModel, 0, len(group.Members))
	for _, m := range group.Members {
		id, name := accessRequestRefForm(m.MemberType, m.ID, m.Name, priorRefs, resolve)
		members = append(members, accessRequestMemberModel{
			Type: types.StringValue(m.MemberType),
			ID:   id,
			Name: name,
		})
	}
	state.Members = accessRequestMembersToSet(ctx, members, diags)
}

// resolveMembers converts member blocks into the API's member shape, resolving names to IDs.
func (r *ApplicationApproversGroupResource) resolveMembers(ctx context.Context, set types.Set, diags *diag.Diagnostics) ([]britive.ApproversGroupMember, error) {
	members := accessRequestMembersFromSet(ctx, set, diags)
	resolved := make([]britive.ApproversGroupMember, 0, len(members))
	for _, m := range members {
		id, err := resolveUserOrTagID(r.client, m.Type.ValueString(), m.ID, m.Name)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, britive.ApproversGroupMember{ID: id, MemberType: m.Type.ValueString()})
	}
	return resolved, nil
}

// accessRequestRef is a user, tag, or notification medium reference from prior state, used to
// keep the id or name form the practitioner configured.
type accessRequestRef struct {
	Type string
	ID   types.String
	Name types.String
}

// accessRequestNameResolver resolves a name of the given entity type to its ID.
type accessRequestNameResolver func(entityType, name string) (string, error)

// accessRequestRefForm returns the id/name pair to store for an API entity. If prior has an
// entry of the same type configured by this ID, the ID form is kept; else if prior has one
// configured by a name that matches - either the API's returned name, or, when resolve is set,
// a name that resolves to this ID - that name is kept; otherwise the ID form is used. Resolving
// covers responses whose name is empty or differs from the configured one (e.g. a display
// name instead of a username).
func accessRequestRefForm(entityType, apiID, apiName string, prior []accessRequestRef, resolve accessRequestNameResolver) (types.String, types.String) {
	for _, p := range prior {
		if p.Type == entityType && isSet(p.ID) && p.ID.ValueString() == apiID {
			return types.StringValue(apiID), types.StringNull()
		}
	}
	for _, p := range prior {
		if p.Type == entityType && isSet(p.Name) && apiName != "" && p.Name.ValueString() == apiName {
			return types.StringNull(), p.Name
		}
	}
	if resolve != nil {
		for _, p := range prior {
			if p.Type != entityType || !isSet(p.Name) {
				continue
			}
			if id, err := resolve(entityType, p.Name.ValueString()); err == nil && id == apiID {
				return types.StringNull(), p.Name
			}
		}
	}
	return types.StringValue(apiID), types.StringNull()
}

// cachedUserOrTagResolver returns a resolver that looks up users and tags by name, caching
// each result for the duration of one Read.
func cachedUserOrTagResolver(client *britive.Client) accessRequestNameResolver {
	cache := make(map[string]string)
	return func(entityType, name string) (string, error) {
		key := entityType + "/" + name
		if id, ok := cache[key]; ok {
			return id, nil
		}
		id, err := resolveUserOrTagID(client, entityType, types.StringNull(), types.StringValue(name))
		if err != nil {
			return "", err
		}
		cache[key] = id
		return id, nil
	}
}

// resolveUserOrTagName returns name when set, otherwise looks up the user's username or the
// tag's name by ID.
func resolveUserOrTagName(client *britive.Client, memberType string, id, name types.String) (string, error) {
	if isSet(name) {
		return name.ValueString(), nil
	}
	switch memberType {
	case "User":
		user, err := client.GetUser(id.ValueString())
		if err != nil {
			return "", fmt.Errorf("looking up user %q: %w", id.ValueString(), err)
		}
		return user.Username, nil
	case "Tag":
		tag, err := client.GetTag(id.ValueString())
		if err != nil {
			return "", fmt.Errorf("looking up tag %q: %w", id.ValueString(), err)
		}
		return tag.Name, nil
	default:
		return "", fmt.Errorf("unsupported member type %q", memberType)
	}
}

// resolveUserOrTagID returns id when set, otherwise looks up the user (by username) or tag
// (by name) and returns its ID.
func resolveUserOrTagID(client *britive.Client, memberType string, id, name types.String) (string, error) {
	if isSet(id) {
		return id.ValueString(), nil
	}
	switch memberType {
	case "User":
		user, err := client.GetUserByName(name.ValueString())
		if err != nil {
			return "", fmt.Errorf("looking up user %q: %w", name.ValueString(), err)
		}
		return user.UserID, nil
	case "Tag":
		tag, err := client.GetTagByName(name.ValueString())
		if err != nil {
			return "", fmt.Errorf("looking up tag %q: %w", name.ValueString(), err)
		}
		return tag.ID, nil
	default:
		return "", fmt.Errorf("unsupported member type %q", memberType)
	}
}

// validateIDOrName adds an error when both or neither of id and name are set. Unknown values
// are skipped, since they're only resolved at apply time.
func validateIDOrName(id, name types.String, p path.Path, block string, diags *diag.Diagnostics) {
	if id.IsUnknown() || name.IsUnknown() {
		return
	}
	if isSet(id) == isSet(name) {
		diags.AddAttributeError(
			p,
			"Invalid Configuration",
			fmt.Sprintf("Each %s block must set exactly one of id or name.", block),
		)
	}
}

// validateBlockRequired adds an error when a required set block has no elements. An unknown
// set is skipped, since its size is only known at apply time.
func validateBlockRequired(set types.Set, p path.Path, block string, diags *diag.Diagnostics) {
	if set.IsUnknown() {
		return
	}
	if set.IsNull() || len(set.Elements()) == 0 {
		diags.AddAttributeError(
			p,
			"Missing Required Block",
			fmt.Sprintf("At least one %s block is required.", block),
		)
	}
}

// isSet reports whether v is known, non-null, and non-empty.
func isSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

// approversGroupCompositeID builds the resource's `id` value.
func approversGroupCompositeID(applicationID, groupID string) string {
	return fmt.Sprintf("apps/%s/approvers-groups/%s", applicationID, groupID)
}

// parseApproversGroupCompositeID extracts applicationID and groupID from either the composite
// ID ("apps/{application_id}/approvers-groups/{group_id}") or "{application_id}/{group_id}".
func parseApproversGroupCompositeID(id string) (applicationID string, groupID string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 4:
		if parts[0] != "apps" || parts[2] != "approvers-groups" {
			return "", "", errs.NewInvalidResourceIDError("approvers group", id)
		}
		return parts[1], parts[3], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", errs.NewInvalidResourceIDError("approvers group", id)
	}
}
