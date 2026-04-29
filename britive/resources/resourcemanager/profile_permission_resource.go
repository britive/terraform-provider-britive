package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)


type ProfilePermissionResource struct {
	client *britive.Client
}

type ProfilePermissionResourceModel struct {
	ID               types.String               `tfsdk:"id"`
	ProfileID        types.String               `tfsdk:"profile_id"`
	PermissionID     types.String               `tfsdk:"permission_id"`
	Name             types.String               `tfsdk:"name"`
	Description      types.String               `tfsdk:"description"`
	Version          validators.CaseInsensitiveStringValue `tfsdk:"version"`
	ResourceTypeID   types.String               `tfsdk:"resource_type_id"`
	ResourceTypeName types.String               `tfsdk:"resource_type_name"`
	Variables        []PermissionVariableModel  `tfsdk:"variables"`
}

type PermissionVariableModel struct {
	Name            types.String `tfsdk:"name"`
	Value           types.String `tfsdk:"value"`
	IsSystemDefined types.Bool   `tfsdk:"is_system_defined"`
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
							Required:    true,
							Description: "Value of variable",
						},
						"is_system_defined": schema.BoolAttribute{
							Required:    true,
							Description: "State value is system defined or not",
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


func (r *ProfilePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfilePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Mapping resource to permission model")

	resourceManagerProfilePermission, err := r.mapResourceToModel(ctx, &plan)
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

	err = r.mapModelToResource(ctx, permissions, created.PermissionID, &plan)
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

	err = r.mapModelToResource(ctx, resourceManagerPermissions, permissionID, &state)
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

	resourceManagerProfilePermission, err := r.mapResourceToModel(ctx, &plan)
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

	err = r.mapModelToResource(ctx, permissions, permissionID, &plan)
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

	err = r.mapModelToResource(ctx, permission, permissionID, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Model to Resource", err.Error())
		return
	}

	log.Printf("[INFO] Imported resource manager profile permission: %s/%s", profileID, permissionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ProfilePermissionResource) mapResourceToModel(ctx context.Context, plan *ProfilePermissionResourceModel) (britive.ResourceManagerProfilePermission, error) {
	permission := britive.ResourceManagerProfilePermission{}

	// Extract profile ID from potential composite ID
	rawProfileID := plan.ProfileID.ValueString()
	profArr := strings.Split(rawProfileID, "/")
	profileID := profArr[len(profArr)-1]
	permission.ProfilID = profileID

	// Get available permissions to find permission ID by name
	rawPermissions, err := r.client.GetAvailablePermissions(profileID)
	if err != nil {
		return permission, fmt.Errorf("error getting available permissions: %v", err)
	}

	permissionName := plan.Name.ValueString()
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

		// Build map of valid permission variables
		permissionVariableMap := make(map[string]bool)
		for _, v := range resourceTypePermission.Variables {
			if varName, ok := v.(string); ok {
				permissionVariableMap[varName] = true
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
				"value":           v.Value.ValueString(),
				"isSystemDefined": v.IsSystemDefined.ValueBool(),
			}
			permission.Variables = append(permission.Variables, varMap)
		}
	} else if len(resourceTypePermission.Variables) > 0 {
		return permission, fmt.Errorf("missing required variables: all variables defined in the '%s' permission are mandatory and must be provided", permissionName)
	}

	return permission, nil
}

func (r *ProfilePermissionResource) mapModelToResource(ctx context.Context, resourceManagerPermissions *britive.ResourceManagerPermissions, permissionID string, state *ProfilePermissionResourceModel) error {
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

	// Map variables directly as a slice (SetNestedBlock)
	var stateVariables []PermissionVariableModel
	if variables, ok := permission["variables"].([]interface{}); ok {
		for _, v := range variables {
			if permMap, ok := v.(map[string]interface{}); ok {
				varModel := PermissionVariableModel{}
				if name, ok := permMap["name"].(string); ok {
					varModel.Name = types.StringValue(name)
				}
				if value, ok := permMap["value"].(string); ok {
					varModel.Value = types.StringValue(value)
				}
				if isSystemDefined, ok := permMap["isSystemDefined"].(bool); ok {
					varModel.IsSystemDefined = types.BoolValue(isSystemDefined)
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
