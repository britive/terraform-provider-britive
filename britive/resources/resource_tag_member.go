package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	_ resource.Resource                = &ResourceTagMember{}
	_ resource.ResourceWithConfigure   = &ResourceTagMember{}
	_ resource.ResourceWithImportState = &ResourceTagMember{}
)

type ResourceTagMember struct {
	client       *britive_client.Client
	helper       *ResourceTagMemberHelper
	importHelper *imports.ImportHelper
}

type ResourceTagMemberHelper struct{}

func NewResourceTagMember() resource.Resource {
	return &ResourceTagMember{}
}

func NewResourceTagMemberHelper() *ResourceTagMemberHelper {
	return &ResourceTagMemberHelper{}
}

func (rtm *ResourceTagMember) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_tag_member"
}

func (rtm *ResourceTagMember) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Tag Member resource")

	if req.ProviderData == nil {
		return
	}

	rtm.client = req.ProviderData.(*britive_client.Client)
	if rtm.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for ResourceTagMember")
	rtm.helper = NewResourceTagMemberHelper()
}

func (rtm *ResourceTagMember) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Tag Member resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the Britive tag",
				Validators: []validator.String{
					validate.StringFunc(
						"tagId",
						validate.StringIsNotWhiteSpace(),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tag_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of britive tag",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "TThe username of the user added to the Britive tag",
				Validators: []validator.String{
					validate.StringFunc(
						"tagId",
						validate.StringIsNotWhiteSpace(),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (rtm *ResourceTagMember) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_tag_member")

	var plan britive_client.TagMemberPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during tag member creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})

		return
	}

	tagID := plan.TagID.ValueString()
	username := plan.Username.ValueString()

	user, err := rtm.client.GetUserByName(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch user", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch user, eror:%#v", err))
		return
	}

	log.Printf("[INFO] Creating new tag member: %s/%s", tagID, user.UserID)
	tflog.Info(ctx, fmt.Sprintf("Creating new tag member: %s/%s", tagID, user.UserID))
	err = rtm.client.CreateTagMember(ctx, tagID, user.UserID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create tag member", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create tag member, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new tag member: %s/%s", tagID, user.UserID))
	plan.ID = types.StringValue(rtm.helper.generateUniqueID(tagID, user.UserID))

	planPtr, err := rtm.helper.getAndMapModelToPlan(ctx, plan, *rtm.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get tag",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map tag member model to plan", map[string]interface{}{
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

func (rtm *ResourceTagMember) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_tag_member")

	if rtm.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.TagMemberPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get tag member state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rtm.helper.getAndMapModelToPlan(ctx, state, *rtm.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get britive tag",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map britive tag member model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read britive tag member:  %#v", planPtr))
}

func (rtm *ResourceTagMember) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (rtm *ResourceTagMember) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_tag_member")

	var state britive_client.TagMemberPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tagID, userID, err := rtm.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete tag member", err.Error())
		tflog.Error(ctx, "Failed to parse ID")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting tag member %s/%s", tagID, userID))

	err = rtm.client.DeleteTagMember(ctx, tagID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete tag member", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete tag member, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleted tag member %s/%s", tagID, userID))
	resp.State.RemoveResource(ctx)
}

func (rtm *ResourceTagMember) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rtm.importHelper.ParseImportID([]string{"tags/(?P<tag_name>[^/]+)/users/(?P<username>[^/]+)", "(?P<tag_name>[^/]+)/(?P<username>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import tag member", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	tagName := importData.Fields["tag_name"]
	username := importData.Fields["username"]

	if strings.TrimSpace(tagName) == "" {
		resp.Diagnostics.AddError("Failed to import tag member", "Invalid tag_name")
		tflog.Error(ctx, "Failed to import tag member, Invalid tag_name")
		return
	}

	if strings.TrimSpace(username) == "" {
		resp.Diagnostics.AddError("Failed to import tag member", "Invalid username")
		tflog.Error(ctx, "Failed to import tag member, Invalid username")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing tag member %s/%s", tagName, username))

	tag, err := rtm.client.GetTagByName(ctx, tagName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import tag member", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import tag member, error:%#v", err))
		return
	}
	user, err := rtm.client.GetUserByName(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import tag member", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import tag member, error:%#v", err))
		return
	}

	plan := britive_client.TagMemberPlan{
		ID:      types.StringValue(rtm.helper.generateUniqueID(tag.ID, user.UserID)),
		TagName: types.StringValue(tagName),
	}

	planPtr, err := rtm.helper.getAndMapModelToPlan(ctx, plan, *rtm.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map tag member model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after import", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Imported tag member %s/%s", tagName, username))
}

func (rtmh *ResourceTagMemberHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.TagMemberPlan, c britive_client.Client) (*britive_client.TagMemberPlan, error) {
	tagID, userID, err := rtmh.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tag, err := c.GetTag(ctx, tagID)
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading tag member %s/%s", tagID, userID))
	u, err := c.GetTagMember(ctx, tagID, userID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("member %s in tag %s", userID, tagID)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received tag member %#v", u))

	plan.TagID = types.StringValue(tagID)
	plan.Username = types.StringValue(u.Username)
	plan.TagName = types.StringValue(tag.Name)

	return &plan, nil
}

func (rtmh *ResourceTagMemberHelper) generateUniqueID(tagID string, userID string) string {
	return fmt.Sprintf("tags/%s/users/%s", tagID, userID)
}

func (rtmh *ResourceTagMemberHelper) parseUniqueID(ID string) (tagID string, userID string, err error) {
	tagMemberParts := strings.Split(ID, "/")
	if len(tagMemberParts) < 4 {
		err = errs.NewInvalidResourceIDError("tag member", ID)
		return
	}
	tagID = tagMemberParts[1]
	userID = tagMemberParts[3]
	return
}
