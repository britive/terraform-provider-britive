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
	_ resource.Resource                = &EntityGroupResource{}
	_ resource.ResourceWithConfigure   = &EntityGroupResource{}
	_ resource.ResourceWithImportState = &EntityGroupResource{}
)

func NewEntityGroupResource() resource.Resource {
	return &EntityGroupResource{}
}

type EntityGroupResource struct {
	client *britive.Client
}

type EntityGroupResourceModel struct {
	ID                types.String `tfsdk:"id"`
	EntityID          types.String `tfsdk:"entity_id"`
	ApplicationID     types.String `tfsdk:"application_id"`
	EntityName        types.String `tfsdk:"entity_name"`
	EntityDescription types.String `tfsdk:"entity_description"`
	ParentID          types.String `tfsdk:"parent_id"`
}

func (r *EntityGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity_group"
}

func (r *EntityGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive application entity group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the entity group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The identity of the application entity of type environment group.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The identity of the Britive application.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"entity_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the entity.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_description": schema.StringAttribute{
				Required:    true,
				Description: "The description of the entity.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent id under which the environment group will be created.",
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

func (r *EntityGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EntityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EntityGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueString()

	entityGroup := britive.ApplicationEntityGroup{
		Name:        plan.EntityName.ValueString(),
		Description: plan.EntityDescription.ValueString(),
		ParentID:    plan.ParentID.ValueString(),
		EntityID:    plan.EntityID.ValueString(),
	}

	created, err := r.client.CreateEntityGroup(entityGroup, applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Entity Group",
			fmt.Sprintf("Could not create entity group: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateEntityGroupID(applicationID, created.EntityID))
	plan.EntityID = types.StringValue(created.EntityID)

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Entity Group",
			fmt.Sprintf("Could not read entity group after creation: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EntityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EntityGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, entityID, err := parseEntityGroupID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Entity Group ID",
			fmt.Sprintf("Could not parse entity group ID: %s", err.Error()),
		)
		return
	}

	appRootEnvGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Entity Group",
			fmt.Sprintf("Could not read application root environment group: %s", err.Error()),
		)
		return
	}

	if appRootEnvGroup == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the entity group
	found := false
	for _, envGroup := range appRootEnvGroup.EnvironmentGroups {
		if envGroup.ID == entityID {
			// Prevent root environment group (parent_id must not be empty)
			if envGroup.ParentID == "" {
				resp.Diagnostics.AddError(
					"Invalid Entity Group",
					"Cannot manage root environment group (parent_id cannot be empty).",
				)
				return
			}

			state.EntityID = types.StringValue(entityID)
			state.EntityName = types.StringValue(envGroup.Name)
			if desc, ok := envGroup.Description.(string); ok {
				state.EntityDescription = types.StringValue(desc)
			} else {
				state.EntityDescription = types.StringValue("")
			}
			state.ParentID = types.StringValue(envGroup.ParentID)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EntityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EntityGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, _, err := parseEntityGroupID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Entity Group ID",
			fmt.Sprintf("Could not parse entity group ID: %s", err.Error()),
		)
		return
	}

	entityGroup := britive.ApplicationEntityGroup{
		EntityID:    plan.EntityID.ValueString(),
		Name:        plan.EntityName.ValueString(),
		Description: plan.EntityDescription.ValueString(),
		ParentID:    plan.ParentID.ValueString(),
	}

	updated, err := r.client.UpdateEntityGroup(entityGroup, applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Entity Group",
			fmt.Sprintf("Could not update entity group: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateEntityGroupID(applicationID, updated.EntityID))

	// Read back to populate all fields
	if err := r.populateStateFromAPI(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Entity Group",
			fmt.Sprintf("Could not read entity group after update: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EntityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EntityGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, entityID, err := parseEntityGroupID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Entity Group ID",
			fmt.Sprintf("Could not parse entity group ID: %s", err.Error()),
		)
		return
	}

	err = r.client.DeleteEntityGroup(applicationID, entityID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Entity Group",
			fmt.Sprintf("Could not delete entity group %s: %s", entityID, err.Error()),
		)
		return
	}
}

func (r *EntityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. apps/{application_id}/root-environment-group/groups/{entity_id}
	// 2. {application_id}/groups/{entity_id}
	idRegexes := []string{
		`^apps/(?P<application_id>[^/]+)/root-environment-group/groups/(?P<entity_id>[^/]+)$`,
		`^(?P<application_id>[^/]+)/groups/(?P<entity_id>[^/]+)$`,
	}

	var applicationID, entityID string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if matchName == "application_id" && i < len(matches) {
					applicationID = matches[i]
				}
				if matchName == "entity_id" && i < len(matches) {
					entityID = matches[i]
				}
			}
			if applicationID != "" && entityID != "" {
				break
			}
		}
	}

	if applicationID == "" || entityID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats", req.ID),
		)
		return
	}

	if strings.TrimSpace(applicationID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "application_id cannot be empty or whitespace")
		return
	}

	if strings.TrimSpace(entityID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "entity_id cannot be empty or whitespace")
		return
	}

	// Get all environment groups for the application
	appEnvs, err := r.client.GetAppEnvs(applicationID, "environmentGroups")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Entity Group",
			fmt.Sprintf("Could not get environment groups: %s", err.Error()),
		)
		return
	}

	envIDList, err := r.client.GetEnvDetails(appEnvs, "id")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Entity Group",
			fmt.Sprintf("Could not get environment details: %s", err.Error()),
		)
		return
	}

	// Verify entity exists
	found := false
	for _, id := range envIDList {
		if id == entityID {
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Entity Group Not Found",
			fmt.Sprintf("Entity group '%s' not found in application '%s'.", entityID, applicationID),
		)
		return
	}

	// Get full entity group details to verify it's not root
	appRootEnvGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Entity Group",
			fmt.Sprintf("Could not get application root environment group: %s", err.Error()),
		)
		return
	}

	for _, envGroup := range appRootEnvGroup.EnvironmentGroups {
		if envGroup.ID == entityID {
			// Prevent root environment group
			if envGroup.ParentID == "" {
				resp.Diagnostics.AddError(
					"Cannot Import Root Entity Group",
					"Cannot manage root environment group (parent_id cannot be empty).",
				)
				return
			}

			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateEntityGroupID(applicationID, entityID))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entity_id"), entityID)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entity_name"), envGroup.Name)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entity_description"), envGroup.Description)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("parent_id"), envGroup.ParentID)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Entity Group Not Found",
		fmt.Sprintf("Entity group '%s' not found in application '%s'.", entityID, applicationID),
	)
}

// populateStateFromAPI fetches entity group data from API and populates the state model
func (r *EntityGroupResource) populateStateFromAPI(ctx context.Context, state *EntityGroupResourceModel) error {
	applicationID, entityID, err := parseEntityGroupID(state.ID.ValueString())
	if err != nil {
		return err
	}

	appRootEnvGroup, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		return err
	}

	if appRootEnvGroup == nil {
		return errors.New("application root environment group not found")
	}

	for _, envGroup := range appRootEnvGroup.EnvironmentGroups {
		if envGroup.ID == entityID {
			if envGroup.ParentID == "" {
				return errors.New("parent_id cannot be empty")
			}

			state.EntityID = types.StringValue(entityID)
			state.EntityName = types.StringValue(envGroup.Name)
			if desc, ok := envGroup.Description.(string); ok {
				state.EntityDescription = types.StringValue(desc)
			} else {
				state.EntityDescription = types.StringValue("")
			}
			state.ParentID = types.StringValue(envGroup.ParentID)
			return nil
		}
	}

	return fmt.Errorf("entity group %s not found for application %s", entityID, applicationID)
}

// Helper functions
func generateEntityGroupID(applicationID, entityID string) string {
	return fmt.Sprintf("apps/%s/root-environment-group/groups/%s", applicationID, entityID)
}

func parseEntityGroupID(id string) (applicationID, entityID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 5 {
		err = fmt.Errorf("invalid entity group ID format: %s", id)
		return
	}

	applicationID = parts[1]
	entityID = parts[4]
	return
}
