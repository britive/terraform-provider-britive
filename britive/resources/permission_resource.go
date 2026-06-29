package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &PermissionResource{}
	_ resource.ResourceWithConfigure   = &PermissionResource{}
	_ resource.ResourceWithImportState = &PermissionResource{}
)

func NewPermissionResource() resource.Resource {
	return &PermissionResource{}
}

type PermissionResource struct {
	client *britive.Client
}

type PermissionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Consumer    types.String `tfsdk:"consumer"`
	Resources   types.Set    `tfsdk:"resources"`
	Actions     types.Set    `tfsdk:"actions"`
}

func (r *PermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (r *PermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the permission.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of Britive permission.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the Britive permission.",
			},
			"consumer": schema.StringAttribute{
				Required:    true,
				Description: "The consumer service.",
			},
			"resources": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Comma separated list of resources.",
			},
			"actions": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Actions to be performed on the resource.",
			},
		},
	}
}

func (r *PermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert resources set to string slice, then to interface slice for API
	var resourceStrs []string
	resp.Diagnostics.Append(plan.Resources.ElementsAs(ctx, &resourceStrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resources := make([]interface{}, len(resourceStrs))
	for i, v := range resourceStrs {
		resources[i] = v
	}

	// Convert actions set to string slice, then to interface slice for API
	var actionStrs []string
	resp.Diagnostics.Append(plan.Actions.ElementsAs(ctx, &actionStrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	actions := make([]interface{}, len(actionStrs))
	for i, v := range actionStrs {
		actions[i] = v
	}

	var createDesc *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		d := plan.Description.ValueString()
		createDesc = &d
	}

	permission := britive.Permission{
		Name:        plan.Name.ValueString(),
		Description: createDesc,
		Consumer:    plan.Consumer.ValueString(),
		Resources:   resources,
		Actions:     actions,
	}

	created, err := r.client.AddPermission(permission)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Permission",
			fmt.Sprintf("Could not create permission: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generatePermissionID(created.PermissionID))

	// Read back to populate computed fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Permission",
			fmt.Sprintf("Could not read permission after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID, err := parsePermissionID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Permission ID",
			fmt.Sprintf("Could not parse permission ID: %s", err.Error()),
		)
		return
	}

	// Get permission by ID to get the name
	permissionInfo, err := r.client.GetPermission(permissionID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Permission",
			fmt.Sprintf("Could not read permission %s: %s", permissionID, err.Error()),
		)
		return
	}

	// Get full permission details by name
	permission, err := r.client.GetPermissionByName(permissionInfo.Name)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Permission",
			fmt.Sprintf("Could not read permission '%s': %s", permissionInfo.Name, err.Error()),
		)
		return
	}

	state.Name = types.StringValue(permission.Name)
	if permission.Description == nil || *permission.Description == "" {
		state.Description = types.StringNull()
	} else {
		state.Description = types.StringValue(*permission.Description)
	}
	state.Consumer = types.StringValue(permission.Consumer)

	// Save prior-state resource strings for normalization matching below.
	var priorResourceStrs []string
	state.Resources.ElementsAs(ctx, &priorResourceStrs, false)

	// Convert []interface{} resources to []string for Framework
	resourceStrs := make([]string, 0, len(permission.Resources))
	for _, r := range permission.Resources {
		if s, ok := r.(string); ok {
			resourceStrs = append(resourceStrs, s)
		}
	}
	// The API may normalize values (e.g. "apps.*" → "apps"). Map them back to the
	// prior-state originals so subsequent plans don't show spurious drift.
	resourceStrs = preserveKnownResourceStrings(resourceStrs, priorResourceStrs)
	resourcesSet, diags := types.SetValueFrom(ctx, types.StringType, resourceStrs)
	resp.Diagnostics.Append(diags...)
	state.Resources = resourcesSet

	// Convert []interface{} actions to []string for Framework
	actionStrs := make([]string, 0, len(permission.Actions))
	for _, a := range permission.Actions {
		if s, ok := a.(string); ok {
			actionStrs = append(actionStrs, s)
		}
	}
	actionsSet, diags := types.SetValueFrom(ctx, types.StringType, actionStrs)
	resp.Diagnostics.Append(diags...)
	state.Actions = actionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PermissionResourceModel
	var state PermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID, err := parsePermissionID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Permission ID",
			fmt.Sprintf("Could not parse permission ID: %s", err.Error()),
		)
		return
	}

	// Convert resources set to string slice, then to interface slice for API
	var resourceStrs []string
	resp.Diagnostics.Append(plan.Resources.ElementsAs(ctx, &resourceStrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resources := make([]interface{}, len(resourceStrs))
	for i, v := range resourceStrs {
		resources[i] = v
	}

	// Convert actions set to string slice, then to interface slice for API
	var actionStrs []string
	resp.Diagnostics.Append(plan.Actions.ElementsAs(ctx, &actionStrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	actions := make([]interface{}, len(actionStrs))
	for i, v := range actionStrs {
		actions[i] = v
	}

	var updateDesc *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		d := plan.Description.ValueString()
		updateDesc = &d
	}

	permission := britive.Permission{
		Name:        plan.Name.ValueString(),
		Description: updateDesc,
		Consumer:    plan.Consumer.ValueString(),
		Resources:   resources,
		Actions:     actions,
	}

	// Get old name for update
	oldName := state.Name.ValueString()

	_, err = r.client.UpdatePermission(permission, oldName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Permission",
			fmt.Sprintf("Could not update permission: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generatePermissionID(permissionID))

	// Save planned values before populateStateFromAPI overwrites them.
	// The API may normalize or drop resource strings (e.g. "apps.*" → "apps")
	// which would cause a "planned set element does not correlate with any
	// element in actual" error from Terraform core without this preservation step.
	plannedDescription := plan.Description
	plannedResources := plan.Resources

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Permission",
			fmt.Sprintf("Could not read permission after update: %s", err.Error()),
		)
		return
	}

	// Restore planned values so that state exactly matches the plan.
	// Any real drift introduced by the API will surface on the next refresh
	// rather than as an apply-time inconsistency error.
	plan.Resources = plannedResources
	plan.Description = plannedDescription

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID, err := parsePermissionID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Permission ID",
			fmt.Sprintf("Could not parse permission ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeletePermission(permissionID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Permission",
			fmt.Sprintf("Could not delete permission %s: %s", permissionID, err.Error()),
		)
		return
	}
}

func (r *PermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. permissions/{name}
	// 2. {name}
	idRegexes := []string{
		`^permissions/(?P<name>[^/]+)$`,
		`^(?P<name>[^/]+)$`,
	}

	var permissionName string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "name" && i < len(matches) {
					permissionName = matches[i]
					break
				}
			}
			if permissionName != "" {
				break
			}
		}
	}

	if permissionName == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats: 'permissions/{name}' or '{name}'", req.ID),
		)
		return
	}

	if strings.TrimSpace(permissionName) == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Permission name cannot be empty or whitespace.",
		)
		return
	}

	// Get permission by name
	permission, err := r.client.GetPermissionByName(permissionName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Permission Not Found",
			fmt.Sprintf("Permission '%s' not found.", permissionName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Permission",
			fmt.Sprintf("Could not import permission '%s': %s", permissionName, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generatePermissionID(permission.PermissionID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), permission.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), permission.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("consumer"), permission.Consumer)...)

	// Convert []interface{} resources to []string for Framework
	importResourceStrs := make([]string, 0, len(permission.Resources))
	for _, r := range permission.Resources {
		if s, ok := r.(string); ok {
			importResourceStrs = append(importResourceStrs, s)
		}
	}
	resourcesSet, diags := types.SetValueFrom(ctx, types.StringType, importResourceStrs)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resources"), resourcesSet)...)

	// Convert []interface{} actions to []string for Framework
	importActionStrs := make([]string, 0, len(permission.Actions))
	for _, a := range permission.Actions {
		if s, ok := a.(string); ok {
			importActionStrs = append(importActionStrs, s)
		}
	}
	actionsSet, diags := types.SetValueFrom(ctx, types.StringType, importActionStrs)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("actions"), actionsSet)...)
}

// populateStateFromAPI fetches permission data from API and populates the state model
func (r *PermissionResource) populateStateFromAPI(ctx context.Context, state *PermissionResourceModel) error {
	permissionID, err := parsePermissionID(state.ID.ValueString())
	if err != nil {
		return err
	}

	// Get permission by ID to get the name
	permissionInfo, err := r.client.GetPermission(permissionID)
	if err != nil {
		return err
	}

	// Get full permission details by name
	permission, err := r.client.GetPermissionByName(permissionInfo.Name)
	if err != nil {
		return err
	}

	state.Name = types.StringValue(permission.Name)
	if permission.Description == nil || *permission.Description == "" {
		state.Description = types.StringNull()
	} else {
		state.Description = types.StringValue(*permission.Description)
	}
	state.Consumer = types.StringValue(permission.Consumer)

	// Convert []interface{} resources to []string for Framework
	resourceStrs := make([]string, 0, len(permission.Resources))
	for _, r := range permission.Resources {
		if s, ok := r.(string); ok {
			resourceStrs = append(resourceStrs, s)
		}
	}
	resourcesSet, diags := types.SetValueFrom(ctx, types.StringType, resourceStrs)
	if diags.HasError() {
		return fmt.Errorf("error converting resources to set")
	}
	state.Resources = resourcesSet

	// Convert []interface{} actions to []string for Framework
	actionStrs := make([]string, 0, len(permission.Actions))
	for _, a := range permission.Actions {
		if s, ok := a.(string); ok {
			actionStrs = append(actionStrs, s)
		}
	}
	actionsSet, diags := types.SetValueFrom(ctx, types.StringType, actionStrs)
	if diags.HasError() {
		return fmt.Errorf("error converting actions to set")
	}
	state.Actions = actionsSet

	return nil
}

// normalizeResource strips a trailing ".*" wildcard suffix.
// The Britive API accepts "foo.*" but stores and returns it as "foo", so
// normalizing both sides lets us treat them as equivalent.
func normalizeResource(r string) string {
	return strings.TrimSuffix(r, ".*")
}

// preserveKnownResourceStrings maps each API-returned resource string back to
// its original form from the known (planned or prior-state) list when the two
// normalize to the same base string.  This prevents "inconsistent result after
// apply" errors and spurious plan drift caused by the API stripping ".*"
// wildcards (e.g. "apps.*" → "apps").
// Resources present in apiResources but absent in knownResources are kept as-is.
func preserveKnownResourceStrings(apiResources, knownResources []string) []string {
	knownByNorm := make(map[string]string, len(knownResources))
	for _, k := range knownResources {
		knownByNorm[normalizeResource(k)] = k
	}
	result := make([]string, 0, len(apiResources))
	for _, a := range apiResources {
		if k, ok := knownByNorm[normalizeResource(a)]; ok {
			result = append(result, k)
		} else {
			result = append(result, a)
		}
	}
	return result
}

// Helper functions
func generatePermissionID(permissionID string) string {
	return fmt.Sprintf("permissions/%s", permissionID)
}

func parsePermissionID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid permission ID format: %s", id)
	}
	return parts[1], nil
}
