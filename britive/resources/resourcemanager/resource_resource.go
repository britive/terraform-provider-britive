package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceResource struct {
	client *britive.Client
}

type ResourceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ResourceType    types.String `tfsdk:"resource_type"`
	ResourceTypeID  types.String `tfsdk:"resource_type_id"`
	ParameterValues types.Map    `tfsdk:"parameter_values"`
	ResourceLabels  types.Map    `tfsdk:"resource_labels"`
}

func NewResourceResource() resource.Resource {
	return &ResourceResource{}
}

func (r *ResourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource"
}

func (r *ResourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager server access resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"resource_type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parameter_values": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"resource_labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *ResourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverAccessResource, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}

	log.Printf("[INFO] Adding new server access resource: %#v", serverAccessResource)

	sa, err := r.client.AddServerAccessResource(serverAccessResource)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource", err.Error())
		return
	}

	plan.ID = types.StringValue(sa.ResourceID)
	plan.ResourceTypeID = types.StringValue(sa.ResourceType.ResourceTypeID)

	log.Printf("[INFO] Submitted new server access resource: %#v", sa)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := state.ID.ValueString()

	log.Printf("[INFO] Reading server access resource: %s", resourceID)

	resource, err := r.client.GetServerAccessResource(resourceID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource", err.Error())
		return
	}

	state.Name = types.StringValue(resource.Name)
	state.Description = types.StringValue(resource.Description)
	state.ResourceType = types.StringValue(resource.ResourceType.Name)
	state.ResourceTypeID = types.StringValue(resource.ResourceType.ResourceTypeID)

	if len(resource.ResourceTypeParameterValues) > 0 {
		paramMap, diags := types.MapValueFrom(ctx, types.StringType, resource.ResourceTypeParameterValues)
		resp.Diagnostics.Append(diags...)
		state.ParameterValues = paramMap
	}

	// Build resource labels map, preserving state order when value sets match
	if len(resource.ResourceLabels) > 0 {
		// Extract current state labels for order preservation
		var stateLabels map[string]string
		if !state.ResourceLabels.IsNull() && !state.ResourceLabels.IsUnknown() {
			stateLabels = make(map[string]string)
			resp.Diagnostics.Append(state.ResourceLabels.ElementsAs(ctx, &stateLabels, false)...)
		}

		labelsStringMap := make(map[string]string)
		for k, v := range resource.ResourceLabels {
			if k == "Resource-Type" {
				continue
			}
			// If state has same value set (different order), preserve state order to avoid drift
			if stateVal, exists := stateLabels[k]; exists && sameValueSet(v, stateVal) {
				labelsStringMap[k] = stateVal
			} else {
				sort.Strings(v)
				labelsStringMap[k] = strings.Join(v, ",")
			}
		}
		if len(labelsStringMap) > 0 {
			labelsMap, diags := types.MapValueFrom(ctx, types.StringType, labelsStringMap)
			resp.Diagnostics.Append(diags...)
			state.ResourceLabels = labelsMap
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceResourceModel
	var state ResourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := state.ID.ValueString()

	serverAccessResource, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}
	serverAccessResource.ResourceID = resourceID

	log.Printf("[INFO] Updating server access resource: %s", resourceID)

	_, err = r.client.UpdateServerAccessResource(serverAccessResource, resourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource", err.Error())
		return
	}

	log.Printf("[INFO] Updated server access resource: %s", resourceID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := state.ID.ValueString()

	log.Printf("[INFO] Deleting server access resource: %s", resourceID)

	err := r.client.DeleteServerAccessResource(resourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource", err.Error())
		return
	}

	log.Printf("[INFO] Deleted server access resource: %s", resourceID)
}

func (r *ResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var resourceID string

	if strings.HasPrefix(importID, "resource-manager/resources/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/resources/{id}' or '{id}', got: %s", importID))
			return
		}
		resourceID = parts[2]
	} else {
		resourceID = importID
	}

	log.Printf("[INFO] Importing server access resource: %s", resourceID)

	resource, err := r.client.GetServerAccessResource(resourceID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Not Found", fmt.Sprintf("Resource %s not found", resourceID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource", err.Error())
		return
	}

	var state ResourceResourceModel
	state.ID = types.StringValue(resource.ResourceID)
	state.Name = types.StringValue(resource.Name)
	state.Description = types.StringValue(resource.Description)
	state.ResourceType = types.StringValue(resource.ResourceType.Name)
	state.ResourceTypeID = types.StringValue(resource.ResourceType.ResourceTypeID)

	if len(resource.ResourceTypeParameterValues) > 0 {
		paramMap, diags := types.MapValueFrom(ctx, types.StringType, resource.ResourceTypeParameterValues)
		resp.Diagnostics.Append(diags...)
		state.ParameterValues = paramMap
	}

	if len(resource.ResourceLabels) > 0 {
		labelsStringMap := make(map[string]string)
		for k, v := range resource.ResourceLabels {
			if k == "Resource-Type" {
				continue
			}
			sort.Strings(v)
			labelsStringMap[k] = strings.Join(v, ",")
		}
		if len(labelsStringMap) > 0 {
			labelsMap, diags := types.MapValueFrom(ctx, types.StringType, labelsStringMap)
			resp.Diagnostics.Append(diags...)
			state.ResourceLabels = labelsMap
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceResource) mapResourceToModel(ctx context.Context, plan *ResourceResourceModel) (britive.ServerAccessResource, error) {
	resource := britive.ServerAccessResource{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		ResourceType: britive.ServerAccessResourceType{
			Name: plan.ResourceType.ValueString(),
		},
		ResourceTypeParameterValues: make(map[string]string),
		ResourceLabels:              make(map[string][]string),
	}

	if !plan.ParameterValues.IsNull() {
		var paramValues map[string]string
		diags := plan.ParameterValues.ElementsAs(ctx, &paramValues, false)
		if diags.HasError() {
			return resource, fmt.Errorf("error parsing parameter_values")
		}
		resource.ResourceTypeParameterValues = paramValues
	}

	if !plan.ResourceLabels.IsNull() {
		var labelsMap map[string]string
		diags := plan.ResourceLabels.ElementsAs(ctx, &labelsMap, false)
		if diags.HasError() {
			return resource, fmt.Errorf("error parsing resource_labels")
		}
		// Convert map[string]string to map[string][]string by splitting on commas
		for k, v := range labelsMap {
			resource.ResourceLabels[k] = strings.Split(v, ",")
		}
	}

	// Add the special Resource-Type label
	resource.ResourceLabels["Resource-Type"] = []string{resource.ResourceType.Name}

	return resource, nil
}

// sameValueSet checks if two comma-separated value strings contain the same set of values.
func sameValueSet(apiValues []string, stateStr string) bool {
	stateValues := strings.Split(stateStr, ",")
	if len(apiValues) != len(stateValues) {
		return false
	}
	apiSorted := make([]string, len(apiValues))
	copy(apiSorted, apiValues)
	sort.Strings(apiSorted)
	stateSorted := make([]string, len(stateValues))
	copy(stateSorted, stateValues)
	sort.Strings(stateSorted)
	return strings.Join(apiSorted, ",") == strings.Join(stateSorted, ",")
}
