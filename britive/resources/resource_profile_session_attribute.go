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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceProfileSessionAttribute{}
	_ resource.ResourceWithConfigure   = &ResourceProfileSessionAttribute{}
	_ resource.ResourceWithImportState = &ResourceProfileSessionAttribute{}
)

// SessionAttributeType - godoc
type SessionAttributeType string

const (
	//SessionAttributeTypeStatic - godoc
	SessionAttributeTypeStatic SessionAttributeType = "Static"
	//SessionAttributeTypeIdentity - godoc
	SessionAttributeTypeIdentity SessionAttributeType = "Identity"
)

type ResourceProfileSessionAttribute struct {
	client       *britive_client.Client
	helper       *ResourceProfileSessionAttributeHelper
	importHelper *imports.ImportHelper
}

type ResourceProfileSessionAttributeHelper struct{}

func NewResourceProfileSessionAttribute() resource.Resource {
	return &ResourceProfileSessionAttribute{}
}

func NewResourceProfileSessionAttributeHelper() *ResourceProfileSessionAttributeHelper {
	return &ResourceProfileSessionAttributeHelper{}
}

func (rpsa *ResourceProfileSessionAttribute) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_profile_session_attribute"
}

func (rpsa *ResourceProfileSessionAttribute) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Profile Session Attribute resource")

	if req.ProviderData == nil {
		return
	}

	rpsa.client = req.ProviderData.(*britive_client.Client)
	if rpsa.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Entity Environment")
	rpsa.helper = NewResourceProfileSessionAttributeHelper()
}

func (rpsa *ResourceProfileSessionAttribute) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for profile session attribute resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Profile associated application name",
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"applicationId",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"profile_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The profile name",
			},
			"attribute_name": schema.StringAttribute{
				Optional:    true,
				Description: "profile associated attribute name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"attribute_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "profile associated attribute type",
				Default:     stringdefault.StaticString("Identity"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"Static",
						"Identity",
					),
				},
			},
			"attribute_value": schema.StringAttribute{
				Optional:    true,
				Description: "profile associated attribute value",
			},
			"mapping_name": schema.StringAttribute{
				Required:    true,
				Description: "profile associated attribute mapping name",
				Validators: []validator.String{
					validate.StringFunc(
						"mappingName",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"transitive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "profile associated attribute transitive",
			},
		},
	}
}

func (rpsa *ResourceProfileSessionAttribute) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_profile_session_attribute")

	var plan britive_client.ProfileSessionAttributePlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during profile session attribute creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID := plan.ProfileID.ValueString()
	sessionAttribute, err := rpsa.helper.mapResourceToModel(ctx, plan, rpsa.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to map profile session attribute to model, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new profile session attribute: %#v", sessionAttribute))

	pt, err := rpsa.client.CreateProfileSessionAttribute(ctx, profileID, *sessionAttribute)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create profile session attribute, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new profile session attribute: %#v", *pt))

	plan.ID = types.StringValue(rpsa.helper.generateUniqueID(profileID, pt.ID))

	planPtr, err := rpsa.helper.getAndMapModelToPlan(ctx, plan, *rpsa.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile session attribute",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile session attribute model to plan", map[string]interface{}{
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
	tflog.Info(ctx, "Create completed and state set", map[string]interface{}{
		"profile session attributr": planPtr,
	})

}

func (rpsa *ResourceProfileSessionAttribute) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_profile_session_attribute")

	if rpsa.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ProfileSessionAttributePlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get profile session attribute state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rpsa.helper.getAndMapModelToPlan(ctx, state, *rpsa.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get profile session attribute",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile session attribute model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read profile session attribute:  %#v", planPtr))
}

func (rpsa *ResourceProfileSessionAttribute) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Update called for britive_profile_session_attriute")

	var plan, state britive_client.ProfileSessionAttributePlan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Update failed to get plan/state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	if plan.MappingName.Equal(state.MappingName) && plan.AttributeValue.Equal(state.AttributeValue) && plan.Transitive.Equal(state.Transitive) {
		return
	}

	profileID, sessionAttributeID, err := rpsa.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("error:%#v", err))
		return
	}
	sessionAttribute, err := rpsa.helper.mapResourceToModel(ctx, plan, rpsa.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("error:%#v", err))
		return
	}
	sessionAttribute.ID = sessionAttributeID

	tflog.Info(ctx, fmt.Sprintf("Updating profile session attribute: %#v", *sessionAttribute))

	upt, err := rpsa.client.UpdateProfileSessionAttribute(ctx, profileID, *sessionAttribute)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted Updated profile session attribute: %#v", upt))

	plan.ID = types.StringValue(rpsa.helper.generateUniqueID(profileID, sessionAttributeID))

	planPtr, err := rpsa.helper.getAndMapModelToPlan(ctx, plan, *rpsa.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after update",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map profile session attribute model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Updated profile session attribute: %#v", planPtr))
}

func (rpsa *ResourceProfileSessionAttribute) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_profile_session_attribute")

	var state britive_client.ProfileSessionAttributePlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID, sessionAttributeID, err := rpsa.helper.parseUniqueID(state.ID.String())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete profile session attribute, err:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting profile session attribute: %s/%s", profileID, sessionAttributeID))

	err = rpsa.client.DeleteProfileSessionAttribute(ctx, profileID, sessionAttributeID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete profile session attribute, err:%#v", err))
		return
	}

	log.Printf("[INFO] Deleted profile session attribute: %s/%s", profileID, sessionAttributeID)
	tflog.Info(ctx, fmt.Sprintf("Deleted profile session attribute: %s/%s", profileID, sessionAttributeID))

	resp.State.RemoveResource(ctx)
}

func (rpsa *ResourceProfileSessionAttribute) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rpsa.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/session-attributes/type/(?P<attribute_type>[^/]+)/mapping-name/(?P<mapping_name>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<attribute_type>[^/]+)/(?P<mapping_name>[^/]+)"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import profile session attribute", "Invalid importID")
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID: %#v", err))
		return
	}

	appName := importData.Fields["app_name"]
	profileName := importData.Fields["profile_name"]
	attributeType := importData.Fields["attribute_type"]
	mappingName := importData.Fields["mapping_name"]
	if strings.TrimSpace(appName) == "" {
		resp.Diagnostics.AddError("Failed to import profile session attribute", "Invalid applicationName")
		tflog.Error(ctx, "Failed to import profile session attribute, Invalid applicationName")
		return
	}
	if strings.TrimSpace(profileName) == "" {
		resp.Diagnostics.AddError("Failed to import profile session attribute", "Invalid profileName")
		tflog.Error(ctx, "Failed to import profile session attribute, Invalid profileName")
		return
	}
	if strings.TrimSpace(attributeType) == "" {
		resp.Diagnostics.AddError("Failed to import profile session attribute", "Invalid attributeType")
		tflog.Error(ctx, "Failed to import profile session attribute, Invalid attributeType")
		return
	}
	if strings.TrimSpace(mappingName) == "" {
		resp.Diagnostics.AddError("Failed to import profile session attribute", "Invalid mappingName")
		tflog.Error(ctx, "Failed to import profile session attribute, Invalid mappingName")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing profile session attribute: %s/%s/%s/%s", appName, profileName, attributeType, mappingName))

	app, err := rpsa.client.GetApplicationByName(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile session attribute, %#v", err))
		return
	}
	profile, err := rpsa.client.GetProfileByName(ctx, app.AppContainerID, profileName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile session attribute, %#v", err))
		return
	}

	sessionAttribute, err := rpsa.client.GetProfileSessionAttributeByTypeAndMappingName(ctx, profile.ProfileID, attributeType, mappingName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import profile session attribute", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch profile session attribute, %#v", err))
		return
	}

	plan := britive_client.ProfileSessionAttributePlan{
		ID:          types.StringValue(rpsa.helper.generateUniqueID(profile.ProfileID, sessionAttribute.ID)),
		AppName:     types.StringValue(""),
		ProfileName: types.StringValue(""),
	}

	planPtr, err := rpsa.helper.getAndMapModelToPlan(ctx, plan, *rpsa.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import profile session attribute",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import profile session attribute model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported profile session attribute: %s/%s/%s/%s", appName, profileName, attributeType, mappingName))

}

func (rpsah *ResourceProfileSessionAttributeHelper) mapResourceToModel(ctx context.Context, plan britive_client.ProfileSessionAttributePlan, c *britive_client.Client) (*britive_client.SessionAttribute, error) {
	attributeName := plan.AttributeName.ValueString()
	mappingName := plan.MappingName.ValueString()
	attributeType := plan.AttributeType.ValueString()
	attributeValuePassed := plan.AttributeValue.ValueString()
	transitive := plan.Transitive.ValueBool()
	if attributeType == "" {
		attributeType = string(SessionAttributeTypeIdentity)
	}
	var attributeSchemaId string
	var attributeValue string
	if strings.EqualFold(attributeType, string(SessionAttributeTypeIdentity)) {
		if attributeName == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("attribute_name")
		}
		if attributeValuePassed != "" {
			return nil, fmt.Errorf("expected attribute_value should be empty when attribute_type is %s", attributeType)
		}
		attribute, err := c.GetAttributeByName(ctx, attributeName)
		if errors.Is(err, britive_client.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("session attribute %s", attributeName)
		}
		if err != nil {
			return nil, err
		}
		attributeSchemaId = attribute.ID
	} else {
		if attributeValuePassed == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("attribute_value")
		}
		if attributeName != "" {
			return nil, fmt.Errorf("expected attribute_name should be empty when attribute_type is %s", attributeType)
		}
		attributeValue = attributeValuePassed
	}
	profileSessionAttribute := britive_client.SessionAttribute{
		AttributeSchemaID:    attributeSchemaId,
		MappingName:          mappingName,
		Transitive:           transitive,
		SessionAttributeType: attributeType,
		AttributeValue:       attributeValue,
	}
	return &profileSessionAttribute, nil
}

func (rpsah *ResourceProfileSessionAttributeHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ProfileSessionAttributePlan, c britive_client.Client) (*britive_client.ProfileSessionAttributePlan, error) {
	profileID, sessionAttributeID, err := rpsah.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading profile session attribute: %s/%s", profileID, sessionAttributeID))

	pt, err := c.GetProfileSessionAttribute(ctx, profileID, sessionAttributeID)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("session attribute %s in profile %s", sessionAttributeID, profileID)
	}
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Received profile session attribute: %#v", pt))

	var attributeName string
	var attributeValue string
	if strings.EqualFold(pt.SessionAttributeType, string(SessionAttributeTypeIdentity)) || pt.AttributeSchemaID != "" {
		tflog.Info(ctx, fmt.Sprintf("Reading attribute: %s/%s", profileID, sessionAttributeID))

		attribute, err := c.GetAttribute(ctx, pt.AttributeSchemaID)
		if errors.Is(err, britive_client.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("attribute %s", pt.AttributeSchemaID)
		}
		if err != nil {
			return nil, err
		}

		tflog.Info(ctx, fmt.Sprintf("Received attribute: %#v", attribute))

		attributeName = attribute.Name
	} else {
		attributeValue = pt.AttributeValue
	}
	plan.ProfileID = types.StringValue(profileID)
	plan.AttributeName = types.StringValue(attributeName)
	plan.AttributeType = types.StringValue(pt.SessionAttributeType)
	plan.AttributeValue = types.StringValue(attributeValue)
	plan.MappingName = types.StringValue(pt.MappingName)
	plan.Transitive = types.BoolValue(pt.Transitive)

	return &plan, nil
}

func (rpsah *ResourceProfileSessionAttributeHelper) generateUniqueID(profileID string, attributeID string) string {
	return fmt.Sprintf("paps/%s/session-attributes/%s", profileID, attributeID)
}

func (rpsah *ResourceProfileSessionAttributeHelper) parseUniqueID(ID string) (profileID string, attributeID string, err error) {
	profileTagParts := strings.Split(ID, "/")
	if len(profileTagParts) < 4 {
		err = errs.NewInvalidResourceIDError("profile session attribute", ID)
		return
	}
	profileID = profileTagParts[1]
	attributeID = profileTagParts[3]
	return
}
