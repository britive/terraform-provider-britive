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
	_ resource.Resource                = &TagMemberResource{}
	_ resource.ResourceWithConfigure   = &TagMemberResource{}
	_ resource.ResourceWithImportState = &TagMemberResource{}
)

func NewTagMemberResource() resource.Resource {
	return &TagMemberResource{}
}

type TagMemberResource struct {
	client *britive.Client
}

type TagMemberResourceModel struct {
	ID       types.String `tfsdk:"id"`
	TagID    types.String `tfsdk:"tag_id"`
	TagName  types.String `tfsdk:"tag_name"`
	Username types.String `tfsdk:"username"`
}

func (r *TagMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_member"
}

func (r *TagMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive tag member.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the tag member.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the Britive tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tag_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the Britive tag.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username of the user added to the Britive tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *TagMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TagMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := plan.TagID.ValueString()
	username := plan.Username.ValueString()

	// Get user by username to retrieve user ID
	user, err := r.client.GetUserByName(username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting User",
			fmt.Sprintf("Could not get user '%s': %s", username, err.Error()),
		)
		return
	}

	// Create tag member
	err = r.client.CreateTagMember(tagID, user.UserID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Tag Member",
			fmt.Sprintf("Could not create tag member: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateTagMemberID(tagID, user.UserID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan, tagID, user.UserID); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Tag Member",
			fmt.Sprintf("Could not read tag member after creation: %s", err.Error()),
		)
		return
	}

	// tag_name is Optional+Computed but not returned by the API.
	// Set to null if still unknown to avoid "provider returned unknown value" errors.
	if plan.TagName.IsUnknown() {
		plan.TagName = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TagMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID, userID, err := parseTagMemberID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Tag Member ID",
			fmt.Sprintf("Could not parse tag member ID: %s", err.Error()),
		)
		return
	}

	// Get tag member
	user, err := r.client.GetTagMember(tagID, userID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Tag Member",
			fmt.Sprintf("Could not read tag member %s/%s: %s", tagID, userID, err.Error()),
		)
		return
	}

	state.TagID = types.StringValue(tagID)
	state.Username = types.StringValue(user.Username)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields have RequiresReplace, so Update should never be called
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All tag member fields require replacement. This should not happen.",
	)
}

func (r *TagMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TagMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID, userID, err := parseTagMemberID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Tag Member ID",
			fmt.Sprintf("Could not parse tag member ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeleteTagMember(tagID, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Tag Member",
			fmt.Sprintf("Could not delete tag member %s/%s: %s", tagID, userID, err.Error()),
		)
		return
	}
}

func (r *TagMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. tags/{tag_name}/users/{username}
	// 2. {tag_name}/{username}
	idRegexes := []string{
		`^tags/(?P<tag_name>[^/]+)/users/(?P<username>[^/]+)$`,
		`^(?P<tag_name>[^/]+)/(?P<username>[^/]+)$`,
	}

	var tagName, username string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "tag_name" && i < len(matches) {
					tagName = matches[i]
				}
				if matchName == "username" && i < len(matches) {
					username = matches[i]
				}
			}
			if tagName != "" && username != "" {
				break
			}
		}
	}

	if tagName == "" || username == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'tags/{tag_name}/users/{username}' or '{tag_name}/{username}'", req.ID),
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

	if strings.TrimSpace(username) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Username cannot be empty or whitespace.",
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
			"Error Importing Tag Member",
			fmt.Sprintf("Could not get tag '%s': %s", tagName, err.Error()),
		)
		return
	}

	// Get user by username
	user, err := r.client.GetUserByName(username)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("User '%s' not found.", username),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Tag Member",
			fmt.Sprintf("Could not get user '%s': %s", username, err.Error()),
		)
		return
	}

	// Verify tag member exists
	_, err = r.client.GetTagMember(tag.ID, user.UserID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Tag Member Not Found",
			fmt.Sprintf("User '%s' is not a member of tag '%s'.", username, tagName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Tag Member",
			fmt.Sprintf("Could not verify tag member: %s", err.Error()),
		)
		return
	}

	// Set the ID and attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateTagMemberID(tag.ID, user.UserID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_id"), tag.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
	// Note: tag_name is not set in import (cleared like in SDKv2 version)
}

// populateStateFromAPI fetches tag member data from API and populates the state model
func (r *TagMemberResource) populateStateFromAPI(ctx context.Context, state *TagMemberResourceModel, tagID, userID string) error {
	user, err := r.client.GetTagMember(tagID, userID)
	if err != nil {
		return err
	}

	state.TagID = types.StringValue(tagID)
	state.Username = types.StringValue(user.Username)

	return nil
}

// Helper functions
func generateTagMemberID(tagID, userID string) string {
	return fmt.Sprintf("tags/%s/users/%s", tagID, userID)
}

func parseTagMemberID(id string) (tagID, userID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 4 {
		err = fmt.Errorf("invalid tag member ID format: %s", id)
		return
	}

	tagID = parts[1]
	userID = parts[3]
	return
}
