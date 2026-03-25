package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/planmodifiers"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProfilePermissionResource struct {
	client *britive.Client
}

type ProfilePermissionResourceModel struct {
	ID               types.String                          `tfsdk:"id"`
	ProfileID        types.String                          `tfsdk:"profile_id"`
	PermissionID     types.String                          `tfsdk:"permission_id"`
	Name             types.String                          `tfsdk:"name"`
	Description      types.String                          `tfsdk:"description"`
	Version          validators.CaseInsensitiveStringValue `tfsdk:"version"`
	ResourceTypeID   types.String                          `tfsdk:"resource_type_id"`
	ResourceTypeName types.String                          `tfsdk:"resource_type_name"`
	Variables        []PermissionVariableModel             `tfsdk:"variables"`
}

type PermissionVariableModel struct {
	Name             types.String `tfsdk:"name"`
	Value            types.String `tfsdk:"value"`
	IsSystemDefined  types.Bool   `tfsdk:"is_system_defined"`
	PromptAtCheckout types.Bool   `tfsdk:"prompt_at_checkout"`
	RegexPattern     types.String `tfsdk:"regex_pattern"`
	Description      types.String `tfsdk:"description"`
	Type             types.String `tfsdk:"type"`
}

func NewProfilePermissionResource() resource.Resource {
	return &ProfilePermissionResource{}
}

func (r *ProfilePermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_profile_permission"
}

func (r *ProfilePermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager profile permission",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "Profile Id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission_id": schema.StringAttribute{
				Computed:    true,
				Description: "Profile permission Id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Required:    true,
				CustomType:  validators.CaseInsensitiveStringType{},
				Description: "Version of the permission (case-insensitive: latest, local, or specific version)",
				PlanModifiers: []planmodifier.String{
					planmodifiers.CaseInsensitivePreserveState(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of ResourceType associated with this permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of ResourceType associated with this permission",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"variables": schema.SetNestedBlock{
				Description: "Variables of permission",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of variable associated with permission",
						},
						"value": schema.StringAttribute{
							Optional:    true,
							Description: "Value of variable. The Britive platform does not allow a value to be set when prompt_at_checkout is true, since the value is supplied by the user at checkout time instead.",
						},
						"is_system_defined": schema.BoolAttribute{
							Required:    true,
							Description: "State value is system defined or not",
						},
						"prompt_at_checkout": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Whether the variable's value is supplied by the user at checkout time instead of being configured here. Defaults to false when not set. The Britive platform does not allow a value to be configured when this is true, and does not allow regex_pattern or description to be set when this is false.",
						},
						"regex_pattern": schema.StringAttribute{
							Optional:    true,
							Description: "Regex pattern used to validate the value supplied at checkout. The Britive platform only allows this to be set when prompt_at_checkout is true.",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Description shown to the user at checkout. The Britive platform only allows this to be set when prompt_at_checkout is true.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of the variable",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ProfilePermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

// ModifyPlan normalizes the computed "type" attribute of variables, and
// re-asserts "prompt_at_checkout" directly from config by name.
//
// "type" is purely server-derived, never user-settable, and has no
// schema-level Default: the Terraform plugin framework plans new
// SetNestedBlock element Computed attributes as null rather than unknown
// (even with a UseStateForUnknown plan modifier attached), which then either
// fails the post-apply consistency check once the API returns a concrete
// value (for genuinely new variables) or shows a spurious diff back to
// null/state on every plan (for variables already in state). Worse, when two
// variables in the same Set share most of their other known attribute values,
// the framework's own per-element proposed-value computation can misattribute
// which STATE element's "type" belongs to which plan element, producing a
// non-null but WRONG type (e.g. a String variable planned with a sibling's
// "password" type) - so it is not enough to fix only the null case. The plan
// value is therefore always replaced here by the matching state variable's
// type if one exists, else left unknown for the API's response to resolve -
// purely via Go-level name matching against prior STATE, bypassing whatever
// the framework's own per-element handling already produced, whether that
// was null, correct, or (as above) incorrect.
//
// "prompt_at_checkout" declares a plain schema-level Default (false), which
// in isolation only ever applies when a variable's raw config omits the
// attribute. In practice, though, when two variables in the same Set share
// most of their other known attribute values (e.g. identical regex_pattern
// and description, as happens for two password-style variables), the
// framework's own per-element proposed-value computation can misattribute
// which config block's value belongs to which plan element - so an explicit
// true in config can still surface as a planned false. This is corrected the
// same way as "type": by re-reading the raw CONFIG and reassigning each
// variable's prompt_at_checkout purely via Go-level name matching, which
// bypasses whatever the framework's own per-element handling produced,
// whether that was correct or not.
func (r *ProfilePermissionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ProfilePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config ProfilePermissionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	configPromptAtCheckoutByName := make(map[string]types.Bool)
	for _, cv := range config.Variables {
		if !cv.Name.IsNull() && !cv.Name.IsUnknown() {
			configPromptAtCheckoutByName[cv.Name.ValueString()] = cv.PromptAtCheckout
		}
	}

	stateVarByName := make(map[string]PermissionVariableModel)
	if !req.State.Raw.IsNull() {
		var state ProfilePermissionResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, sv := range state.Variables {
			if !sv.Name.IsNull() && !sv.Name.IsUnknown() {
				stateVarByName[sv.Name.ValueString()] = sv
			}
		}
	}

	modified := false
	for i, pv := range plan.Variables {
		if pv.Name.IsNull() || pv.Name.IsUnknown() {
			continue
		}
		name := pv.Name.ValueString()
		sv, inState := stateVarByName[name]

		wantType := types.StringUnknown()
		if inState {
			wantType = sv.Type
		}
		if !pv.Type.Equal(wantType) {
			plan.Variables[i].Type = wantType
			modified = true
		}

		if cv, ok := configPromptAtCheckoutByName[name]; ok {
			want := cv
			if cv.IsNull() {
				want = types.BoolValue(false)
			}
			if !pv.PromptAtCheckout.Equal(want) {
				plan.Variables[i].PromptAtCheckout = want
				modified = true
			}
		}
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *ProfilePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfilePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Mapping resource to permission model")

	resourceManagerProfilePermission, err := r.mapResourceToModel(ctx, &plan, "")
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}

	log.Printf("[INFO] Creating profile permission %#v", resourceManagerProfilePermission)

	created, err := r.client.CreateUpdateResourceManagerProfilePermission(resourceManagerProfilePermission, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Manager Profile Permission", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/profile/%s/permission/%s", created.ProfilID, created.PermissionID))
	plan.PermissionID = types.StringValue(created.PermissionID)

	log.Printf("[INFO] Created profile permission %#v", created)

	// Read back to get computed values
	permissions, err := r.client.GetResourceManagerProfilePermission(created.ProfilID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile Permission", err.Error())
		return
	}

	err = r.mapModelToResource(ctx, permissions, created.PermissionID, &plan, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Model to Resource", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfilePermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, permissionID := r.parseUniqueID(state.ID.ValueString())

	log.Printf("[INFO] Reading profile permission with profile: %s and permission: %s", profileID, permissionID)

	resourceManagerPermissions, err := r.client.GetResourceManagerProfilePermission(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile Permission", err.Error())
		return
	}

	log.Printf("[INFO] Finding permission from list of permissions: %#v", resourceManagerPermissions)

	err = r.mapModelToResource(ctx, resourceManagerPermissions, permissionID, &state, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Model to Resource", err.Error())
		return
	}

	// Check if permission was found (if not, mapModelToResource sets empty ID)
	if state.PermissionID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfilePermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfilePermissionResourceModel
	var state ProfilePermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check for version change (case-insensitive)
	if !strings.EqualFold(state.Version.ValueString(), plan.Version.ValueString()) {
		resp.Diagnostics.AddError(
			"Immutable Field Changed",
			fmt.Sprintf("field 'version' is immutable and cannot be changed (from '%s' to '%s')", state.Version.ValueString(), plan.Version.ValueString()),
		)
		return
	}

	profileID, permissionID := r.parseUniqueID(state.ID.ValueString())

	resourceManagerProfilePermission, err := r.mapResourceToModel(ctx, &plan, permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}
	resourceManagerProfilePermission.ProfilID = profileID
	resourceManagerProfilePermission.PermissionID = permissionID

	log.Printf("[INFO] Updating resource manager profile permission: %#v", resourceManagerProfilePermission)

	_, err = r.client.CreateUpdateResourceManagerProfilePermission(resourceManagerProfilePermission, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource Manager Profile Permission", err.Error())
		return
	}

	log.Printf("[INFO] Updated resource manager profile permission: %s", state.ID.ValueString())

	// Read back to get updated values
	permissions, err := r.client.GetResourceManagerProfilePermission(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile Permission", err.Error())
		return
	}

	err = r.mapModelToResource(ctx, permissions, permissionID, &plan, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Model to Resource", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfilePermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, permissionID := r.parseUniqueID(state.ID.ValueString())

	log.Printf("[INFO] Deleting resource manager profile permission with profile: %s, permission: %s", profileID, permissionID)

	err := r.client.DeleteResourceManagerProfilePermission(profileID, permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Manager Profile Permission", err.Error())
		return
	}

	log.Printf("[INFO] Deleted resource manager profile permission with profile: %s, permission: %s", profileID, permissionID)
}

func (r *ProfilePermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var profileID, permissionID string

	// Support two formats: "resource-manager/profile/{profile_id}/permission/{permission_id}" or "{profile_id}/{permission_id}"
	if strings.HasPrefix(importID, "resource-manager/profile/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 5 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/profile/{profile_id}/permission/{permission_id}' or '{profile_id}/{permission_id}', got: %s", importID))
			return
		}
		profileID = parts[2]
		permissionID = parts[4]
	} else {
		parts := strings.Split(importID, "/")
		if len(parts) != 2 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/profile/{profile_id}/permission/{permission_id}' or '{profile_id}/{permission_id}', got: %s", importID))
			return
		}
		profileID = parts[0]
		permissionID = parts[1]
	}

	if strings.TrimSpace(profileID) == "" || strings.TrimSpace(permissionID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Profile ID and Permission ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing resource manager profile permission: %s/%s", profileID, permissionID)

	permission, err := r.client.GetResourceManagerProfilePermission(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Manager Profile Permission Not Found", fmt.Sprintf("Permission %s for profile %s not found", permissionID, profileID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Manager Profile Permission", err.Error())
		return
	}

	// Check if permission exists in the list
	isFoundPermission := false
	for _, perm := range permission.Permissions {
		if permID, ok := perm["permissionId"].(string); ok && permID == permissionID {
			isFoundPermission = true
			break
		}
	}

	if !isFoundPermission {
		resp.Diagnostics.AddError("Resource Manager Profile Permission Not Found", fmt.Sprintf("Permission with id: %s not found", permissionID))
		return
	}

	var state ProfilePermissionResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/profile/%s/permission/%s", profileID, permissionID))
	state.ProfileID = types.StringValue(profileID)

	err = r.mapModelToResource(ctx, permission, permissionID, &state, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Model to Resource", err.Error())
		return
	}

	log.Printf("[INFO] Imported resource manager profile permission: %s/%s", profileID, permissionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ProfilePermissionResource) mapResourceToModel(ctx context.Context, plan *ProfilePermissionResourceModel, existingPermissionID string) (britive.ResourceManagerProfilePermission, error) {
	permission := britive.ResourceManagerProfilePermission{}

	// Extract profile ID from potential composite ID
	rawProfileID := plan.ProfileID.ValueString()
	profArr := strings.Split(rawProfileID, "/")
	profileID := profArr[len(profArr)-1]
	permission.ProfilID = profileID

	permissionName := plan.Name.ValueString()

	if existingPermissionID != "" {
		// Updating a permission already associated with the profile: it will
		// no longer show up in the "available" (unassigned) permissions list,
		// so use the permission ID already known from state instead of
		// looking it up.
		permission.PermissionID = existingPermissionID
	} else {
		// Get available permissions to find permission ID by name
		rawPermissions, err := r.client.GetAvailablePermissions(profileID)
		if err != nil {
			return permission, fmt.Errorf("error getting available permissions: %v", err)
		}

		for _, perm := range rawPermissions.Permissions {
			if name, ok := perm["name"].(string); ok && name == permissionName {
				if permID, ok := perm["permissionId"].(string); ok {
					permission.PermissionID = permID
					break
				}
			}
		}

		if permission.PermissionID == "" {
			return permission, fmt.Errorf("permission '%s' is invalid or already associated with the profile", permissionName)
		}
	}

	// Normalize version
	version := plan.Version.ValueString()
	if strings.EqualFold(version, "latest") || strings.EqualFold(version, "local") {
		version = strings.ToLower(version)
	}

	// Get specified version permission to validate version and variables
	resourceTypePermission, err := r.client.GetSpecifiedVersionPermission(permission.PermissionID, version)
	if err != nil {
		return permission, fmt.Errorf("permission with version: %s not found: %v", version, err)
	}

	permission.Version = version
	permission.ResourceTypeId = resourceTypePermission.ResourceTypeID
	permission.ResourceTypeName = resourceTypePermission.ResourceTypeName

	// Validate variables (use slice directly, no ElementsAs needed)
	if len(plan.Variables) > 0 {
		userVariables := plan.Variables

		// Build map of valid permission variables. Variable names defined on the
		// resource type permission may carry a "<name>:<type>" suffix (e.g. "test3:password")
		// to declare the variable's type, so match on the base name only.
		permissionVariableMap := make(map[string]bool)
		for _, v := range resourceTypePermission.Variables {
			if varName, ok := v.(string); ok {
				baseName, _, _ := strings.Cut(varName, ":")
				permissionVariableMap[baseName] = true
			}
		}

		// Validate user variables
		for _, v := range userVariables {
			varName := v.Name.ValueString()
			if !permissionVariableMap[varName] {
				return permission, fmt.Errorf("the variable '%s' is not valid for the '%s' permission", varName, permissionName)
			}
		}

		// Check if all required variables are provided
		if len(userVariables) < len(resourceTypePermission.Variables) {
			return permission, fmt.Errorf("missing required variables: all variables defined in the '%s' permission are mandatory and must be provided", permissionName)
		}

		// Convert variables to map format for API
		for _, v := range userVariables {
			varMap := map[string]interface{}{
				"name":            v.Name.ValueString(),
				"isSystemDefined": v.IsSystemDefined.ValueBool(),
			}
			// Always send "value" explicitly, even as null. Omitting the key
			// entirely (when the user clears/removes it from config) appears to
			// be treated by the API as "leave the existing stored value
			// unchanged" rather than "clear it" - which left a stale value in
			// place after a variable's value was removed from config, causing
			// a post-apply consistency mismatch (planned null vs. actual stale
			// value). Note that the Britive platform itself does not allow a
			// value to be set at all when prompt_at_checkout is true - this
			// provider does not validate that combination locally and instead
			// lets the platform accept, clear, or reject it.
			if v.Value.IsNull() {
				varMap["value"] = nil
			} else if !v.Value.IsUnknown() {
				varMap["value"] = v.Value.ValueString()
			}
			if !v.RegexPattern.IsNull() && !v.RegexPattern.IsUnknown() {
				varMap["regexPattern"] = v.RegexPattern.ValueString()
			}
			if !v.Description.IsNull() && !v.Description.IsUnknown() {
				varMap["description"] = v.Description.ValueString()
			}
			if !v.PromptAtCheckout.IsNull() && !v.PromptAtCheckout.IsUnknown() {
				varMap["promptAtCheckout"] = v.PromptAtCheckout.ValueBool()
			}
			permission.Variables = append(permission.Variables, varMap)
		}
	} else if len(resourceTypePermission.Variables) > 0 {
		return permission, fmt.Errorf("missing required variables: all variables defined in the '%s' permission are mandatory and must be provided", permissionName)
	}

	return permission, nil
}

// mapModelToResource maps the API's permission representation onto state.
// trustConfiguredValues must be true when called from Create/Update, right
// after submitting `state`'s own variables to the API in the same operation:
// in that case, any variable attribute the API doesn't echo back is filled in
// from the just-submitted values so the resulting state always matches what
// was planned (required for Terraform's post-apply consistency check). It
// must be false when called from Read/ImportState, an independent refresh
// against previously-known state: there, every gap is left null instead, so
// that genuine backend-side state (e.g. the backend never persists "value"
// for a variable with prompt_at_checkout = true, which is always the case for
// password-type variables) is reflected in state and surfaces as a diff on
// the next plan rather than being silently masked.
func (r *ProfilePermissionResource) mapModelToResource(ctx context.Context, resourceManagerPermissions *britive.ResourceManagerPermissions, permissionID string, state *ProfilePermissionResourceModel, trustConfiguredValues bool) error {
	var permission map[string]interface{}
	for _, perm := range resourceManagerPermissions.Permissions {
		if permID, ok := perm["permissionId"].(string); ok && permID == permissionID {
			permission = perm
			break
		}
	}

	if permission == nil {
		// Permission not found, clear the state
		state.PermissionID = types.StringNull()
		return nil
	}

	log.Printf("[INFO] Setting resource manager profile permission %#v", permission)

	if permID, ok := permission["permissionId"].(string); ok {
		state.PermissionID = types.StringValue(permID)
	}
	if permName, ok := permission["permissionName"].(string); ok {
		state.Name = types.StringValue(permName)
	}
	if desc, ok := permission["description"].(string); ok {
		state.Description = types.StringValue(desc)
	}
	if version, ok := permission["version"].(string); ok {
		state.Version = validators.NewCaseInsensitiveStringValue(version)
	}
	if rtID, ok := permission["resourceTypeId"].(string); ok {
		state.ResourceTypeID = types.StringValue(rtID)
	}
	if rtName, ok := permission["resourceTypeName"].(string); ok {
		state.ResourceTypeName = types.StringValue(rtName)
	}

	existingVariablesByName := make(map[string]PermissionVariableModel)
	for _, v := range state.Variables {
		if !v.Name.IsNull() && !v.Name.IsUnknown() {
			existingVariablesByName[v.Name.ValueString()] = v
		}
	}

	// Map variables directly as a slice (SetNestedBlock)
	var stateVariables []PermissionVariableModel
	if variables, ok := permission["variables"].([]interface{}); ok {
		for _, v := range variables {
			if permMap, ok := v.(map[string]interface{}); ok {
				varModel := PermissionVariableModel{}
				if name, ok := permMap["name"].(string); ok {
					varModel.Name = types.StringValue(name)
				}
				existing, hasExisting := existingVariablesByName[varModel.Name.ValueString()]

				if t, ok := permMap["type"].(string); ok {
					varModel.Type = types.StringValue(t)
				}

				if value, ok := permMap["value"].(string); ok {
					varModel.Value = types.StringValue(value)
				} else if trustConfiguredValues && hasExisting {
					// Create/Update: the value was just submitted in this same
					// operation, so reflect it back exactly (required for
					// Terraform's post-apply consistency check). On a later,
					// independent Read/Import, no fallback is applied here:
					// the backend never persists a value for a variable with
					// prompt_at_checkout = true (which is always the case for
					// password-type variables), so a null value in that case
					// is the correct, expected state rather than a masked one.
					varModel.Value = existing.Value
				}
				if isSystemDefined, ok := permMap["isSystemDefined"].(bool); ok {
					varModel.IsSystemDefined = types.BoolValue(isSystemDefined)
				}
				if promptAtCheckout, ok := permMap["promptAtCheckout"].(bool); ok {
					varModel.PromptAtCheckout = types.BoolValue(promptAtCheckout)
				} else if trustConfiguredValues && hasExisting {
					varModel.PromptAtCheckout = existing.PromptAtCheckout
				}
				if regexPattern, ok := permMap["regexPattern"].(string); ok {
					varModel.RegexPattern = types.StringValue(regexPattern)
				} else if trustConfiguredValues && hasExisting {
					varModel.RegexPattern = existing.RegexPattern
				}
				if description, ok := permMap["description"].(string); ok {
					varModel.Description = types.StringValue(description)
				} else if trustConfiguredValues && hasExisting {
					varModel.Description = existing.Description
				}
				stateVariables = append(stateVariables, varModel)
			}
		}
	}

	state.Variables = stateVariables

	log.Printf("[INFO] Read resource manager profile permission: %#v", permission)
	return nil
}

func (r *ProfilePermissionResource) parseUniqueID(id string) (string, string) {
	idArr := strings.Split(id, "/")
	profileID := idArr[len(idArr)-3]
	permissionID := idArr[len(idArr)-1]
	return profileID, permissionID
}
