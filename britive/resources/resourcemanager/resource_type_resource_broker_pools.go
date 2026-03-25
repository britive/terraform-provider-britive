package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceBrokerPoolsResource struct {
	client *britive.Client
}

type ResourceBrokerPoolsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ResourceID  types.String `tfsdk:"resource_id"`
	BrokerPools types.Set    `tfsdk:"broker_pools"`
}

func NewResourceBrokerPoolsResource() resource.Resource {
	return &ResourceBrokerPoolsResource{}
}

func (r *ResourceBrokerPoolsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_broker_pools"
}

func (r *ResourceBrokerPoolsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Britive broker pools associated to a server access resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Required:    true,
				Description: "The id of server access resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"broker_pools": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "The broker pool names to be associated to the resource",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ResourceBrokerPoolsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceBrokerPoolsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceBrokerPoolsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverAccessResourceID := plan.ResourceID.ValueString()

	serverAccessResourceName, err := r.client.GetResourceName(serverAccessResourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Name", err.Error())
		return
	}

	var brokerPoolNames []string
	resp.Diagnostics.Append(plan.BrokerPools.ElementsAs(ctx, &brokerPoolNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err = r.client.AddBrokerPoolsResource(brokerPoolNames, serverAccessResourceName)
	if err != nil {
		resp.Diagnostics.AddError("Error Adding Broker Pools to Resource", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new broker pools %#v for the resource: %s", brokerPoolNames, serverAccessResourceID)

	plan.ID = types.StringValue(fmt.Sprintf("resources/%s/broker-pools", serverAccessResourceID))

	// Read back to confirm
	retrievedPools, err := r.client.GetBrokerPoolsResource(serverAccessResourceName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pools", err.Error())
		return
	}

	var poolNames []string
	for _, pool := range *retrievedPools {
		poolNames = append(poolNames, pool.Name)
	}

	poolsSet, diags := types.SetValueFrom(ctx, types.StringType, poolNames)
	resp.Diagnostics.Append(diags...)
	plan.BrokerPools = poolsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceBrokerPoolsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceBrokerPoolsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverAccessResourceID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Reading broker pools for resource %s", serverAccessResourceID)

	state.ResourceID = types.StringValue(serverAccessResourceID)

	serverAccessResourceName, err := r.client.GetResourceName(serverAccessResourceID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Name", err.Error())
		return
	}

	brokerPools, err := r.client.GetBrokerPoolsResource(serverAccessResourceName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pools", err.Error())
		return
	}

	log.Printf("[INFO] Received broker pools %#v for resource %s", brokerPools, serverAccessResourceName)

	var poolNames []string
	for _, pool := range *brokerPools {
		poolNames = append(poolNames, pool.Name)
	}

	poolsSet, diags := types.SetValueFrom(ctx, types.StringType, poolNames)
	resp.Diagnostics.Append(diags...)
	state.BrokerPools = poolsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceBrokerPoolsResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are ForceNew, so update should never be called
	resp.Diagnostics.AddError("Unexpected Update", "All fields are immutable - update should not be called")
}

func (r *ResourceBrokerPoolsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceBrokerPoolsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverAccessResourceID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting broker pools for resource: %s", serverAccessResourceID)

	err = r.client.DeleteBrokerPoolsResource(serverAccessResourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Broker Pools", err.Error())
		return
	}

	log.Printf("[INFO] Broker pools for the resource %s are deleted", serverAccessResourceID)
}

func (r *ResourceBrokerPoolsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var resourceName string

	// Support two formats: "resources/{resource_name}/broker-pools" or "{resource_name}/broker-pools"
	if strings.HasPrefix(importID, "resources/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 || parts[2] != "broker-pools" {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resources/{resource_name}/broker-pools' or '{resource_name}/broker-pools', got: %s", importID))
			return
		}
		resourceName = parts[1]
	} else {
		parts := strings.Split(importID, "/")
		if len(parts) != 2 || parts[1] != "broker-pools" {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resources/{resource_name}/broker-pools' or '{resource_name}/broker-pools', got: %s", importID))
			return
		}
		resourceName = parts[0]
	}

	if strings.TrimSpace(resourceName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Resource name cannot be empty")
		return
	}

	log.Printf("[INFO] Importing broker pools for the resource : %s", resourceName)

	// Look up resource by name to get ID
	serverAccessResource, err := r.client.GetServerAccessResourceByName(resourceName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Not Found", fmt.Sprintf("Resource %s not found", resourceName))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource", err.Error())
		return
	}

	serverAccessResourceID := serverAccessResource.ResourceID

	brokerPools, err := r.client.GetBrokerPoolsResource(resourceName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pools", err.Error())
		return
	}

	var poolNames []string
	for _, pool := range *brokerPools {
		poolNames = append(poolNames, pool.Name)
	}

	var state ResourceBrokerPoolsResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resources/%s/broker-pools", serverAccessResourceID))
	state.ResourceID = types.StringValue(serverAccessResourceID)

	poolsSet, diags := types.SetValueFrom(ctx, types.StringType, poolNames)
	resp.Diagnostics.Append(diags...)
	state.BrokerPools = poolsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ResourceBrokerPoolsResource) parseUniqueID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid resource ID format: %s (expected 'resources/{resourceID}/broker-pools')", id)
	}
	return parts[1], nil
}
