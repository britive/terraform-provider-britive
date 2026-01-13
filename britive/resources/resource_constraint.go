package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceConstraint{}
	_ resource.ResourceWithConfigure   = &ResourceConstraint{}
	_ resource.ResourceWithImportState = &ResourceConstraint{}
)

type ResourceConstraint struct {
	client       *britive_client.Client
	helper       *ResourceConstraintHelper
	importHelper *imports.ImportHelper
}

type ResourceConstraintHelper struct{}

func NewResourceConstraint() resource.Resource {
	return &ResourceConstraint{}
}

func NewResourceConstraintHelper() *ResourceConstraintHelper {
	return &ResourceConstraintHelper{}
}

func (rc *ResourceConstraint) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_constraint"
}

func (rc *ResourceConstraint) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Constraint resource")

	if req.ProviderData == nil {
		return
	}

	rc.client = req.ProviderData.(*britive_client.Client)
	if rc.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured for Resource Constraint")
	rc.helper = NewResourceConstraintHelper()
}

func (rc *ResourceConstraint) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		validate.ConstraintExclusiveFieldsValidator{},
	}
}

func (rc *ResourceConstraint) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for constraint resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"profileID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"permission_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the permission associated with the profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"permissionName",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"permission_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The type of permission",
				Default:     stringdefault.StaticString("role"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"permissionType",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"constraint_type": schema.StringAttribute{
				Required:    true,
				Description: "The constraint type for a given profile permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.StringFunc(
						"constraintType",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the constraint",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Description: "Title of the condition constraint",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expression": schema.StringAttribute{
				Optional:    true,
				Description: "Expression of the condition constraint",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the condition constraint",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (rc *ResourceConstraint) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Create called for britive_constraint")

	var plan britive_client.ConstraintPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during constraint creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID := plan.ProfileID.ValueString()
	permissionName := plan.PermissionName.ValueString()
	permissionType := plan.PermissionType.ValueString()
	constraintType := plan.ConstraintType.ValueString()
	if strings.EqualFold(constraintType, "condition") {
		constraint := britive_client.ConditionConstraint{}
		err := rc.helper.mapConditionResourceToModel(plan, &constraint)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create contraint", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to map constrain resource model, %#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Creating new condition constraint: %#v", constraint))

		co, err := rc.client.CreateConditionConstraint(ctx, profileID, permissionName, permissionType, constraintType, constraint)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create constraint", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to create constraint, %#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted new condition constraint: %#v", co))
		plan.ID = types.StringValue(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, co.Title))
	} else {
		constraint := britive_client.Constraint{}
		err := rc.helper.mapResourceToModel(plan, &constraint)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create constraint", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to create constraint, %#v", err.Error()))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Creating new constraint: %#v", constraint))
		co, err := rc.client.CreateConstraint(ctx, profileID, permissionName, permissionType, constraintType, constraint)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create constraint", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to create constraint, %#v", err))
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Submitted new constraint: %#v", constraint))
		plan.ID = types.StringValue(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, co.Name))
	}

	planPtr, err := rc.helper.getAndMapModelToPlan(ctx, plan, *rc.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get constraint",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map constraint model to plan", map[string]interface{}{
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
		"constraint": planPtr,
	})
}

func (rc *ResourceConstraint) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_constraint")

	if rc.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ConstraintPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get constraint state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	planPtr, err := rc.helper.getAndMapModelToPlan(ctx, state, *rc.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get constraint",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map constraint model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Read constraint:  %#v", planPtr))
}

func (rc *ResourceConstraint) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (rc *ResourceConstraint) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_constraint")

	var state britive_client.ConstraintPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileID, permissionName, permissionType, constraintType, constraintName, err := rc.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete constraint", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse ID, %#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting constraint: %s for permission %s of profile %s", constraintName, permissionName, profileID))
	err = rc.client.DeleteConstraint(ctx, profileID, permissionName, permissionType, constraintType, constraintName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete constraint", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete constraint, %#v", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Deleted constraint %s from profile %s for permission %s", constraintName, profileID, permissionName))
	resp.State.RemoveResource(ctx)
}

func (rc *ResourceConstraint) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	var plan britive_client.ConstraintPlan

	importConstraintType, err := rc.importHelper.FetchImportFieldValue([]string{"paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)"}, importData, "constraint_type")
	if err != nil {
		resp.Diagnostics.AddError("Invalid importID", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, %#v", err))
		return
	}

	if strings.EqualFold(importConstraintType, "condition") {
		if err := rc.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<title>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<title>[^/]+)"}, importData); err != nil {
			resp.Diagnostics.AddError("Invalid importID", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, %#v", err))
			return
		}

		profileID := importData.Fields["profile_id"]
		permissionName := importData.Fields["permission_name"]
		permissionType := importData.Fields["permission_type"]
		constraintType := importData.Fields["constraint_type"]
		constraintTitle := importData.Fields["title"]
		if strings.TrimSpace(profileID) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid profileID")
			tflog.Error(ctx, "Failed to import constraint, Invalid profileID")
			return
		}
		if strings.TrimSpace(permissionName) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid permissionName")
			tflog.Error(ctx, "Failed to import constraint, Invalid permissionName")
			return
		}
		if strings.TrimSpace(permissionType) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid permissionType")
			tflog.Error(ctx, "Failed to import constraint, Invalid permissionType")
			return
		}
		if strings.TrimSpace(constraintType) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid constraintType")
			tflog.Error(ctx, "Failed to import constraint, Invalid constraintType")
			return
		}
		if strings.TrimSpace(constraintTitle) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid Title")
			tflog.Error(ctx, "Failed to import constraint, Invalid Title")
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Importing Constraint: %s/%s/%s/%s/%s", profileID, permissionName, permissionType, constraintType, constraintTitle))
		constraintResult, err := rc.client.GetConditionConstraint(ctx, profileID, permissionName, permissionType, constraintType)
		if err != nil {
			resp.Diagnostics.AddError("Failed to import constraint", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to import constraint, %#v", err))
			return
		}

		if !britive_client.ConditionConstraintEqual(constraintTitle, constraintResult.Result[0].Expression, constraintResult.Result[0].Description, constraintResult) {
			resp.Diagnostics.AddError("Failed to import constraint", fmt.Sprintf("Constraint %s of type %s for profile %s of permission %s", constraintTitle, constraintType, profileID, permissionName))
			tflog.Error(ctx, fmt.Sprintf("Constraint %s of type %s for profile %s of permission %s, err:=%#v", constraintTitle, constraintType, profileID, permissionName, err))
			return
		}

		plan.ID = types.StringValue(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, constraintTitle))
	} else {
		if err := rc.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)"}, importData); err != nil {
			resp.Diagnostics.AddError("Invalid importID", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to parse importID, %#v", err))
			return
		}

		profileID := importData.Fields["profile_id"]
		permissionName := importData.Fields["permission_name"]
		permissionType := importData.Fields["permission_type"]
		constraintType := importData.Fields["constraint_type"]
		constraintName := importData.Fields["name"]

		if strings.TrimSpace(profileID) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid profileID")
			tflog.Error(ctx, "Failed to import constraint, Invalid profileID")
			return
		}
		if strings.TrimSpace(permissionName) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid permissionName")
			tflog.Error(ctx, "Failed to import constraint, Invalid permissionName")
			return
		}
		if strings.TrimSpace(permissionType) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid permissionType")
			tflog.Error(ctx, "Failed to import constraint, Invalid permissionType")
			return
		}
		if strings.TrimSpace(constraintType) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid constraintType")
			tflog.Error(ctx, "Failed to import constraint, Invalid constraintType")
			return
		}
		if strings.TrimSpace(constraintName) == "" {
			resp.Diagnostics.AddError("Failed to import constraint", "Invalid name")
			tflog.Error(ctx, "Failed to import constraint, Invalid name")
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Importing Constraint: %s/%s/%s/%s/%s", profileID, permissionName, permissionType, constraintType, constraintName))
		constraintResult, err := rc.client.GetConstraint(ctx, profileID, permissionName, permissionType, constraintType)
		if err != nil {
			resp.Diagnostics.AddError("Failed to import constraint", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to import constraint, %#v", err))
			return
		}

		if !britive_client.ConstraintEqual(constraintName, constraintResult) {
			resp.Diagnostics.AddError("Failed to import constraint", fmt.Sprintf("Constraint %s of type %s for profile %s of permission %s", constraintName, constraintType, profileID, permissionName))
			tflog.Error(ctx, fmt.Sprintf("Constraint %s of type %s for profile %s of permission %s, error:%#v", constraintName, constraintType, profileID, permissionName, err))
			return
		}

		plan.ID = types.StringValue(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, constraintName))
	}

	planPtr, err := rc.helper.getAndMapModelToPlan(ctx, plan, *rc.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import constraint",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed import constraint model to plan", map[string]interface{}{
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

	tflog.Info(ctx, fmt.Sprintf("Imported constraint : %#v", planPtr))
}

func (rch *ResourceConstraintHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ConstraintPlan, c britive_client.Client) (*britive_client.ConstraintPlan, error) {
	profileID, permissionName, permissionType, constraintType, constraintName, err := rch.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading constraint %s for the permission %s", constraintName, permissionName))

	if strings.EqualFold(constraintType, "condition") {
		constraintResult, err := c.GetConditionConstraint(ctx, profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive_client.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("Constraint %s in permission %s for profile id %s", constraintName, permissionName, profileID)
		}
		if err != nil {
			return nil, err
		}

		newTitle := plan.Title.ValueString()
		newExpression := plan.Expression.ValueString()
		newDescription := plan.Description.ValueString()
		if britive_client.ConditionConstraintEqual(newTitle, newExpression, newDescription, constraintResult) {
			plan.Title = types.StringValue(newTitle)
			plan.Expression = types.StringValue(newExpression)
			plan.Description = types.StringValue(newDescription)
		} else {
			for _, rule := range constraintResult.Result {
				plan.Title = types.StringValue(rule.Title)
				plan.Expression = types.StringValue(rule.Expression)
				plan.Description = types.StringValue(rule.Description)
			}
		}
	} else {
		constraintResult, err := c.GetConstraint(ctx, profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive_client.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("Constraint %s in permission %s for profile id %s", constraintName, permissionName, profileID)
		}
		if err != nil {
			return nil, err
		}

		newName := plan.Name.ValueString()

		if britive_client.ConstraintEqual(newName, constraintResult) {
			plan.Name = types.StringValue(newName)
		} else {
			for _, rule := range constraintResult.Result {
				plan.Name = types.StringValue(rule.Name)
			}
		}
	}
	return &plan, nil
}

func (rch *ResourceConstraintHelper) mapConditionResourceToModel(plan britive_client.ConstraintPlan, constraint *britive_client.ConditionConstraint) error {

	constraint.Title = plan.Title.ValueString()
	constraint.Expression = plan.Expression.ValueString()
	constraint.Description = plan.Description.ValueString()

	return nil
}

func (rch *ResourceConstraintHelper) mapResourceToModel(plan britive_client.ConstraintPlan, constraint *britive_client.Constraint) error {

	constraint.Name = plan.Name.ValueString()

	return nil
}

func (resourceConstraintHelper *ResourceConstraintHelper) generateUniqueID(profileId, permissionName, permissionType, constraintType, constraintName string) string {
	return fmt.Sprintf("paps/%s/permissions/%s/%s/constraints/%s/%s", profileId, permissionName, permissionType, constraintType, constraintName)
}

func (resourceConstraintHelper *ResourceConstraintHelper) parseUniqueID(ID string) (profileId, permissionName, permissionType, constraintType, constraintName string, err error) {
	constraintParts := strings.Split(ID, "/")
	if len(constraintParts) < 8 {
		err = errs.NewInvalidResourceIDError("Constraint", ID)
		return
	}

	profileId = constraintParts[1]
	permissionName = constraintParts[3]
	permissionType = constraintParts[4]
	constraintType = constraintParts[6]
	constraintName = constraintParts[7]
	return
}
