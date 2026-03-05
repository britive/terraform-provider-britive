package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceTag{}
	_ resource.ResourceWithConfigure   = &ResourceTag{}
	_ resource.ResourceWithImportState = &ResourceTag{}
)

type ResourceTag struct {
	client       *britive_client.Client
	helper       *ResourceTagHelper
	importHelper *imports.ImportHelper
}

type ResourceTagHelper struct{}

func NewResourceTag() resource.Resource {
	return &ResourceTag{}
}

func NewResourceTagHelper() *ResourceTagHelper {
	return &ResourceTagHelper{}
}

func (rt *ResourceTag) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_tag"
}

func (rt *ResourceTag) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Tag resource")

	if req.ProviderData == nil {
		return
	}

	rt.client = req.ProviderData.(*britive_client.Client)
	if rt.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for ResourceTag")
	rt.helper = NewResourceTagHelper()
}

func (rt *ResourceTag) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Tag resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of Britive Tag",
				Validators: []validator.String{
					validate.StringFunc(
						"name",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive tag",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "To disable britive tag",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"identity_provider_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identity of the identity provider associated with the Britive tag",
				Validators: []validator.String{
					validate.StringFunc(
						"name",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"external": schema.BoolAttribute{
				Computed:    true,
				Description: "The boolean attribute that indicates whether the tag is external or not",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (rt *ResourceTag) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_tag")

	var plan britive_client.TagPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during tag creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})

		return
	}

	err := rt.helper.validateForExternalTag(ctx, plan, rt.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to validate tag", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to validate tag, error:%#v", err))
		return
	}

	tag := britive_client.Tag{}
	tag.Name = plan.Name.ValueString()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		tag.Description = plan.Description.ValueString()
	}

	if plan.Disabled.ValueBool() {
		tag.Status = "Inactive"
	} else {
		tag.Status = "Active"
	}

	tag.UserTagIdentityProviders = []britive_client.UserTagIdentityProvider{
		{
			IdentityProvider: britive_client.IdentityProvider{
				ID: plan.IdentityProviderID.ValueString(),
			},
		},
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new tag: %#v", tag))
	ut, err := rt.client.CreateTag(ctx, tag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create tag", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create tag, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new tag: %#v", ut))

	plan.ID = types.StringValue(ut.ID)

	planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, rt.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get tag",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map tag model to plan", map[string]interface{}{
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

func (rt *ResourceTag) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_tag")

	if rt.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.TagPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get tag state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rt.helper.getAndMapModelToPlan(ctx, state, rt.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get britive tag",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map britive tag model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read britive tag:  %#v", planPtr))
}

func (rt *ResourceTag) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_tag")

	var plan, state britive_client.TagPlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	err := rt.helper.validateForExternalTag(ctx, plan, rt.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to validate Tag", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to validate external tag or not, error:%#v", err))
		return
	}

	tagID := plan.ID.ValueString()
	var hasChanges bool
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) {
		hasChanges = true
		tag := britive_client.Tag{}
		tag.Name = plan.Name.ValueString()
		tag.Description = plan.Description.ValueString()

		tflog.Info(ctx, fmt.Sprintf("Updating tag: %#v", tag))
		ut, err := rt.client.UpdateTag(ctx, tagID, tag)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update tag", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update tag, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated tag: %#v", ut))
		plan.ID = types.StringValue(ut.ID)
	}
	if !plan.Disabled.Equal(state.Disabled) {
		hasChanges = true
		disabled := plan.Disabled.ValueBool()

		tflog.Info(ctx, fmt.Sprintf("Updating status disabled: %t of tag: %s", disabled, tagID))
		ut, err := rt.client.EnableOrDisableTag(ctx, tagID, disabled)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update tag", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to update tag, error:%#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted updated status of tag: %#v", ut))
		plan.ID = types.StringValue(ut.ID)
	}
	if hasChanges {
		planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, rt.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to set state after update",
				fmt.Sprintf("Error: %v", err),
			)
			tflog.Error(ctx, "Failed get and map tag model to plan", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "Failed to set state after update", map[string]interface{}{
				"diagnostics": resp.Diagnostics,
			})
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Updated tag: %#v", planPtr))
	}
}

func (rt *ResourceTag) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_tag")

	var state britive_client.TagPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	err := rt.helper.validateForExternalTag(ctx, state, rt.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to validate Tag", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to validate external tag or not, error:%#v", err))
		return
	}

	tagID := state.ID.ValueString()

	tflog.Info(ctx, fmt.Sprintf("Deleting tag: %s", tagID))
	err = rt.client.DeleteTag(ctx, tagID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete tag", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete tag, error:%#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Tag %s deleted", tagID))
	resp.State.RemoveResource(ctx)
}

func (rt *ResourceTag) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rt.importHelper.ParseImportID([]string{"tags/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import tag", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	tagName := importData.Fields["name"]

	if strings.TrimSpace(tagName) == "" {
		resp.Diagnostics.AddError("Failed to import tag", "Invalid name")
		tflog.Error(ctx, "Failed to import tag, Invalid name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing tag: %s", tagName))

	tag, err := rt.client.GetTagByName(ctx, tagName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import tag", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to import tag, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Imported tag: %#v", tag))

	if tag.External.(bool) {
		resp.Diagnostics.AddError("Failed to import tag", fmt.Sprintf("importing external tags is not supported. attempted to import tag '%s'", tagName))
		tflog.Error(ctx, fmt.Sprintf("importing external tags is not supported. attempted to import tag '%s'", tagName))
		return
	}

	plan := britive_client.TagPlan{
		ID: types.StringValue(tag.ID),
	}

	planPtr, err := rt.helper.getAndMapModelToPlan(ctx, plan, rt.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map tag model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported tag: %#v", planPtr))
}

func (rth *ResourceTagHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.TagPlan, c *britive_client.Client) (*britive_client.TagPlan, error) {
	err := rth.validateForExternalTag(ctx, plan, c)
	if err != nil {
		return nil, err
	}

	tagID := plan.ID.ValueString()

	tflog.Info(ctx, fmt.Sprintf("Reading tag %s", tagID))
	tag, err := c.GetTag(ctx, tagID)
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received tag: %#v", tag))
	plan.Name = types.StringValue(tag.Name)
	if (plan.Description.IsNull() || plan.Description.IsUnknown()) && (strings.EqualFold(tag.Description, "No Description") || tag.Description == britive_client.EmptyString) {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(tag.Description)
	}

	if len(tag.UserTagIdentityProviders) > 0 {
		plan.IdentityProviderID = types.StringValue(tag.UserTagIdentityProviders[0].IdentityProvider.ID)
	}
	plan.Disabled = types.BoolValue(strings.EqualFold(tag.Status, "Inactive"))
	plan.External = types.BoolValue(tag.External.(bool))
	return &plan, nil
}

func (rth *ResourceTagHelper) validateForExternalTag(ctx context.Context, plan britive_client.TagPlan, c *britive_client.Client) error {
	identityProviderID := plan.IdentityProviderID.ValueString()
	if identityProviderID == "" {
		return nil
	}

	identityProvider, err := c.GetIdentityProvider(ctx, identityProviderID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(identityProvider.Type, "DEFAULT") {
		return fmt.Errorf("managing external tags is not supported. attempted to manage tag '%s'", plan.Name.ValueString())
	}
	return nil
}
