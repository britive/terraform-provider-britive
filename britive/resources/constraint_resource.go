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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ConstraintResource{}
	_ resource.ResourceWithConfigure   = &ConstraintResource{}
	_ resource.ResourceWithImportState = &ConstraintResource{}
	_ resource.ResourceWithValidateConfig = &ConstraintResource{}
)

func NewConstraintResource() resource.Resource {
	return &ConstraintResource{}
}

type ConstraintResource struct {
	client *britive.Client
}

type ConstraintResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProfileID      types.String `tfsdk:"profile_id"`
	PermissionName types.String `tfsdk:"permission_name"`
	PermissionType types.String `tfsdk:"permission_type"`
	ConstraintType types.String `tfsdk:"constraint_type"`
	Name           types.String `tfsdk:"name"`
	Title          types.String `tfsdk:"title"`
	Expression     types.String `tfsdk:"expression"`
	Description    types.String `tfsdk:"description"`
}

func (r *ConstraintResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_constraint"
}

func (r *ConstraintResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive profile permission constraint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the constraint.",
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
			"permission_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the permission associated with the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("role"),
				Description: "The type of permission.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"constraint_type": schema.StringAttribute{
				Required:    true,
				Description: "The constraint type for a given profile permission.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the constraint. Cannot be set if title, expression, or description are set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Description: "Title of the condition constraint. Cannot be set if name is set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expression": schema.StringAttribute{
				Optional:    true,
				Description: "Expression of the condition constraint. Cannot be set if name is set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the condition constraint. Cannot be set if name is set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ConstraintResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig implements custom validation logic
func (r *ConstraintResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ConstraintResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check mutual exclusivity: name vs condition fields (title, expression, description)
	nameSet := !data.Name.IsNull() && data.Name.ValueString() != ""
	titleSet := !data.Title.IsNull() && data.Title.ValueString() != ""
	expressionSet := !data.Expression.IsNull() && data.Expression.ValueString() != ""
	descriptionSet := !data.Description.IsNull() && data.Description.ValueString() != ""

	if nameSet && (titleSet || expressionSet || descriptionSet) {
		resp.Diagnostics.AddError(
			"Invalid Constraint Configuration",
			"If 'name' is set, then 'title', 'expression', and 'description' cannot be set, and vice versa.",
		)
	}
}

func (r *ConstraintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConstraintResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := plan.ProfileID.ValueString()
	permissionName := plan.PermissionName.ValueString()
	permissionType := plan.PermissionType.ValueString()
	constraintType := plan.ConstraintType.ValueString()

	// Check if this is a condition constraint or regular constraint
	if strings.EqualFold(constraintType, "condition") {
		// Create condition constraint
		constraint := britive.ConditionConstraint{
			Title:       plan.Title.ValueString(),
			Expression:  plan.Expression.ValueString(),
			Description: plan.Description.ValueString(),
		}

		created, err := r.client.CreateConditionConstraint(profileID, permissionName, permissionType, constraintType, constraint)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Condition Constraint",
				fmt.Sprintf("Could not create condition constraint: %s", err.Error()),
			)
			return
		}

		plan.ID = types.StringValue(generateConstraintID(profileID, permissionName, permissionType, constraintType, created.Title))
	} else {
		// Create regular constraint
		constraint := britive.Constraint{
			Name: plan.Name.ValueString(),
		}

		created, err := r.client.CreateConstraint(profileID, permissionName, permissionType, constraintType, constraint)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Constraint",
				fmt.Sprintf("Could not create constraint: %s", err.Error()),
			)
			return
		}

		plan.ID = types.StringValue(generateConstraintID(profileID, permissionName, permissionType, constraintType, created.Name))
	}

	// Read back to populate any computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Constraint",
			fmt.Sprintf("Could not read constraint after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConstraintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConstraintResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, permissionName, permissionType, constraintType, _, err := parseConstraintID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Constraint ID",
			fmt.Sprintf("Could not parse constraint ID: %s", err.Error()),
		)
		return
	}

	if strings.EqualFold(constraintType, "condition") {
		// Read condition constraint
		result, err := r.client.GetConditionConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Condition Constraint",
				fmt.Sprintf("Could not read condition constraint: %s", err.Error()),
			)
			return
		}

		// Find matching constraint by comparing with current state
		newTitle := state.Title.ValueString()
		newExpression := state.Expression.ValueString()
		newDescription := state.Description.ValueString()

		if britive.ConditionConstraintEqual(newTitle, newExpression, newDescription, result) {
			state.Title = types.StringValue(newTitle)
			state.Expression = types.StringValue(newExpression)
			state.Description = types.StringValue(newDescription)
		} else {
			// Update with first matching result
			if len(result.Result) > 0 {
				state.Title = types.StringValue(result.Result[0].Title)
				state.Expression = types.StringValue(result.Result[0].Expression)
				state.Description = types.StringValue(result.Result[0].Description)
			}
		}
	} else {
		// Read regular constraint
		result, err := r.client.GetConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Constraint",
				fmt.Sprintf("Could not read constraint: %s", err.Error()),
			)
			return
		}

		// Find matching constraint
		newName := state.Name.ValueString()

		if britive.ConstraintEqual(newName, result) {
			state.Name = types.StringValue(newName)
		} else {
			// Update with first matching result
			if len(result.Result) > 0 {
				state.Name = types.StringValue(result.Result[0].Name)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConstraintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields have RequiresReplace, so Update should never be called
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All constraint fields require replacement. This should not happen.",
	)
}

func (r *ConstraintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConstraintResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, permissionName, permissionType, constraintType, constraintName, err := parseConstraintID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Constraint ID",
			fmt.Sprintf("Could not parse constraint ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeleteConstraint(profileID, permissionName, permissionType, constraintType, constraintName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Constraint",
			fmt.Sprintf("Could not delete constraint %s: %s", constraintName, err.Error()),
		)
		return
	}
}

func (r *ConstraintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. paps/{profile_id}/permissions/{permission_name}/{permission_type}/constraints/{constraint_type}/{name_or_title}
	// 2. {profile_id}/{permission_name}/{permission_type}/{constraint_type}/{name_or_title}
	idRegexes := []string{
		`^paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)$`,
		`^(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)$`,
	}

	var profileID, permissionName, permissionType, constraintType, nameOrTitle string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if i == 0 {
					continue
				}
				switch matchName {
				case "profile_id":
					profileID = matches[i]
				case "permission_name":
					permissionName = matches[i]
				case "permission_type":
					permissionType = matches[i]
				case "constraint_type":
					constraintType = matches[i]
				case "name":
					nameOrTitle = matches[i]
				}
			}
			break
		}
	}

	if profileID == "" || permissionName == "" || permissionType == "" || constraintType == "" || nameOrTitle == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats", req.ID),
		)
		return
	}

	// Validate fields
	if strings.TrimSpace(profileID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "profile_id cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(permissionName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "permission_name cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(permissionType) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "permission_type cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(constraintType) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "constraint_type cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(nameOrTitle) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "constraint name/title cannot be empty or whitespace")
		return
	}

	// Determine if it's a condition constraint
	if strings.EqualFold(constraintType, "condition") {
		// Get condition constraint
		result, err := r.client.GetConditionConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Constraint Not Found",
				fmt.Sprintf("Constraint type '%s' not found for profile '%s', permission '%s'", constraintType, profileID, permissionName),
			)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Importing Constraint",
				fmt.Sprintf("Could not import constraint: %s", err.Error()),
			)
			return
		}

		// Find matching condition constraint
		found := false
		for _, rule := range result.Result {
			if rule.Title == nameOrTitle {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateConstraintID(profileID, permissionName, permissionType, constraintType, nameOrTitle))...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_name"), permissionName)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_type"), permissionType)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("constraint_type"), constraintType)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("title"), rule.Title)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("expression"), rule.Expression)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), rule.Description)...)
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Constraint Not Found",
				fmt.Sprintf("Condition constraint '%s' not found for profile '%s', permission '%s'", nameOrTitle, profileID, permissionName),
			)
		}
	} else {
		// Get regular constraint
		result, err := r.client.GetConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Constraint Not Found",
				fmt.Sprintf("Constraint type '%s' not found for profile '%s', permission '%s'", constraintType, profileID, permissionName),
			)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Importing Constraint",
				fmt.Sprintf("Could not import constraint: %s", err.Error()),
			)
			return
		}

		// Find matching constraint
		if !britive.ConstraintEqual(nameOrTitle, result) {
			resp.Diagnostics.AddError(
				"Constraint Not Found",
				fmt.Sprintf("Constraint '%s' not found for profile '%s', permission '%s'", nameOrTitle, profileID, permissionName),
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateConstraintID(profileID, permissionName, permissionType, constraintType, nameOrTitle))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_name"), permissionName)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_type"), permissionType)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("constraint_type"), constraintType)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), nameOrTitle)...)
	}
}

// populateStateFromAPI fetches constraint data from API and populates the state model
func (r *ConstraintResource) populateStateFromAPI(ctx context.Context, state *ConstraintResourceModel) error {
	profileID, permissionName, permissionType, constraintType, _, err := parseConstraintID(state.ID.ValueString())
	if err != nil {
		return err
	}

	if strings.EqualFold(constraintType, "condition") {
		result, err := r.client.GetConditionConstraint(profileID, permissionName, permissionType, constraintType)
		if err != nil {
			return err
		}

		// Find matching constraint
		newTitle := state.Title.ValueString()
		newExpression := state.Expression.ValueString()
		newDescription := state.Description.ValueString()

		if britive.ConditionConstraintEqual(newTitle, newExpression, newDescription, result) {
			state.Title = types.StringValue(newTitle)
			state.Expression = types.StringValue(newExpression)
			state.Description = types.StringValue(newDescription)
		} else if len(result.Result) > 0 {
			state.Title = types.StringValue(result.Result[0].Title)
			state.Expression = types.StringValue(result.Result[0].Expression)
			state.Description = types.StringValue(result.Result[0].Description)
		}
	} else {
		result, err := r.client.GetConstraint(profileID, permissionName, permissionType, constraintType)
		if err != nil {
			return err
		}

		newName := state.Name.ValueString()

		if britive.ConstraintEqual(newName, result) {
			state.Name = types.StringValue(newName)
		} else if len(result.Result) > 0 {
			state.Name = types.StringValue(result.Result[0].Name)
		}
	}

	return nil
}

// Helper functions
func generateConstraintID(profileID, permissionName, permissionType, constraintType, constraintName string) string {
	return fmt.Sprintf("paps/%s/permissions/%s/%s/constraints/%s/%s", profileID, permissionName, permissionType, constraintType, constraintName)
}

func parseConstraintID(id string) (profileID, permissionName, permissionType, constraintType, constraintName string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 8 {
		err = fmt.Errorf("invalid constraint ID format: %s", id)
		return
	}

	profileID = parts[1]
	permissionName = parts[3]
	permissionType = parts[4]
	constraintType = parts[6]
	constraintName = parts[7]
	return
}
