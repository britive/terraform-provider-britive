package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TagOwnerResource{}
	_ resource.ResourceWithConfigure   = &TagOwnerResource{}
	_ resource.ResourceWithImportState = &TagOwnerResource{}
)

func NewTagOwnerResource() resource.Resource {
	return &TagOwnerResource{}
}

type TagOwnerResource struct {
	client *britive.Client
}

type TagOwnerResourceModel struct {
	ID    types.String      `tfsdk:"id"`
	TagID types.String      `tfsdk:"tag_id"`
	Users []TagOwnerEntityModel  `tfsdk:"user"`
	Tags  []TagOwnerEntityModel  `tfsdk:"tag"`
}

// TagOwnerEntity maps to each user {} or tag {} block.
// Exactly one of id or name must be set.
type TagOwnerEntityModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *TagOwnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_owner"
}

func (r *TagOwnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	ownerBlock := schema.SetNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The identifier of the owner entity.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"name": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The name of the owner entity.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(1),
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages owner relationships for a Britive tag.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this resource (tags/{tag_id}/owners).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the Britive tag.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"user": ownerBlock,
			"tag":  ownerBlock,
		},
	}
}

func (r *TagOwnerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *TagOwnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TagOwnerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := plan.TagID.ValueString()
	log.Printf("[INFO] Creating tag owners for tag: %s", tagID)

	if err := r.applyOwners(tagID, plan.Users, plan.Tags); err != nil {
		resp.Diagnostics.AddError("Error Creating Tag Owners", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("tags/%s/owners", tagID))

	// Read back to populate computed fields
	if err := r.readIntoModel(tagID, &plan); err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Tag Owners After Create", err.Error())
		return
	}

	log.Printf("[INFO] Created tag owners for tag: %s", tagID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagOwnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TagOwnerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := parseTagOwnerID(state.ID.ValueString())

	if err := r.readIntoModel(tagID, &state); err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Tag Owners", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagOwnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TagOwnerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := parseTagOwnerID(state.ID.ValueString())
	log.Printf("[INFO] Updating tag owners for tag: %s", tagID)

	if err := r.applyOwners(tagID, plan.Users, plan.Tags); err != nil {
		resp.Diagnostics.AddError("Error Updating Tag Owners", err.Error())
		return
	}

	plan.ID = state.ID

	if err := r.readIntoModel(tagID, &plan); err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Tag Owners After Update", err.Error())
		return
	}

	log.Printf("[INFO] Updated tag owners for tag: %s", tagID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagOwnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TagOwnerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := parseTagOwnerID(state.ID.ValueString())
	log.Printf("[INFO] Deleting tag owners for tag: %s", tagID)

	tag, err := r.client.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Tag", err.Error())
		return
	}

	request := britive.TagWithOwners{
		TagID:       tagID,
		Name:        tag.Name,
		Description: tag.Description,
		Relationships: britive.TagOwnerRelationships{
			Owners: []britive.TagOwnerEntity{},
		},
	}
	if _, err = r.client.UpdateTagOwners(request); err != nil {
		resp.Diagnostics.AddError("Error Deleting Tag Owners", err.Error())
		return
	}

	log.Printf("[INFO] Deleted tag owners for tag: %s", tagID)
}

func (r *TagOwnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	tagID := importID
	if strings.HasPrefix(importID, "tags/") && strings.HasSuffix(importID, "/owners") {
		tagID = strings.TrimSuffix(strings.TrimPrefix(importID, "tags/"), "/owners")
	}

	_, err := r.client.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Tag Not Found", fmt.Sprintf("Tag %s not found", tagID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Tag", err.Error())
		return
	}

	var state TagOwnerResourceModel
	state.ID = types.StringValue(fmt.Sprintf("tags/%s/owners", tagID))
	state.TagID = types.StringValue(tagID)

	if err := r.readIntoModel(tagID, &state); err != nil {
		resp.Diagnostics.AddError("Error Reading Tag Owners", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyOwners calls UpdateTagOwners with the given user and tag owner lists.
func (r *TagOwnerResource) applyOwners(tagID string, users, tags []TagOwnerEntityModel) error {
	tag, err := r.client.GetTag(tagID)
	if err != nil {
		return fmt.Errorf("getting tag %s: %w", tagID, err)
	}

	owners := make([]britive.TagOwnerEntity, 0)
	for _, u := range users {
		owner := britive.TagOwnerEntity{RelatedEntityType: "User"}
		if !u.ID.IsNull() && !u.ID.IsUnknown() && u.ID.ValueString() != "" {
			owner.RelatedEntityID = u.ID.ValueString()
		} else {
			owner.RelatedEntityName = u.Name.ValueString()
		}
		owners = append(owners, owner)
	}
	for _, t := range tags {
		owner := britive.TagOwnerEntity{RelatedEntityType: "Tag"}
		if !t.ID.IsNull() && !t.ID.IsUnknown() && t.ID.ValueString() != "" {
			owner.RelatedEntityID = t.ID.ValueString()
		} else {
			owner.RelatedEntityName = t.Name.ValueString()
		}
		owners = append(owners, owner)
	}

	request := britive.TagWithOwners{
		TagID:       tagID,
		Name:        tag.Name,
		Description: tag.Description,
		Relationships: britive.TagOwnerRelationships{Owners: owners},
	}
	_, err = r.client.UpdateTagOwners(request)
	return err
}

// readIntoModel fetches the current owner state from the API and populates the model.
// It preserves how the user configured each owner (by id or by name) using a lookup map.
func (r *TagOwnerResource) readIntoModel(tagID string, model *TagOwnerResourceModel) error {
	tagWithOwners, err := r.client.GetTagWithOwners(tagID)
	if err != nil {
		return err
	}

	// Build lookup maps keyed by the identifier the caller used (id or name)
	stateUserByKey := buildTagOwnerKeyMap(model.Users)
	stateTagByKey := buildTagOwnerKeyMap(model.Tags)

	var users, tags []TagOwnerEntityModel

	for _, owner := range tagWithOwners.Relationships.Owners {
		if owner.RelatedEntityType == "User" {
			entry := resolveTagOwnerEntry(owner, stateUserByKey)
			users = append(users, entry)
		} else if owner.RelatedEntityType == "Tag" {
			entry := resolveTagOwnerEntry(owner, stateTagByKey)
			tags = append(tags, entry)
		}
	}

	model.TagID = types.StringValue(tagID)
	if users == nil {
		users = []TagOwnerEntityModel{}
	}
	if tags == nil {
		tags = []TagOwnerEntityModel{}
	}
	model.Users = users
	model.Tags = tags
	return nil
}

func buildTagOwnerKeyMap(entities []TagOwnerEntityModel) map[string]TagOwnerEntityModel {
	m := make(map[string]TagOwnerEntityModel, len(entities))
	for _, e := range entities {
		if !e.ID.IsNull() && !e.ID.IsUnknown() && e.ID.ValueString() != "" {
			m[e.ID.ValueString()] = e
		} else if !e.Name.IsNull() && !e.Name.IsUnknown() && e.Name.ValueString() != "" {
			m[e.Name.ValueString()] = e
		}
	}
	return m
}

func resolveTagOwnerEntry(owner britive.TagOwnerEntity, stateByKey map[string]TagOwnerEntityModel) TagOwnerEntityModel {
	if si, ok := stateByKey[owner.RelatedEntityID]; ok {
		// Configured by id — return id only (preserve user intent)
		return TagOwnerEntityModel{
			ID:   si.ID,
			Name: types.StringNull(),
		}
	}
	if si, ok := stateByKey[owner.RelatedEntityName]; ok {
		// Configured by name — return name only
		return TagOwnerEntityModel{
			ID:   types.StringNull(),
			Name: si.Name,
		}
	}
	// External addition not in state — store by id
	return TagOwnerEntityModel{
		ID:   types.StringValue(owner.RelatedEntityID),
		Name: types.StringNull(),
	}
}

func parseTagOwnerID(id string) string {
	// "tags/{tagID}/owners" → tagID
	parts := strings.Split(id, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return id
}
