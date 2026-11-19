package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/brokerpool"
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
	_ resource.Resource                = &BrokerPoolResource{}
	_ resource.ResourceWithConfigure   = &BrokerPoolResource{}
	_ resource.ResourceWithImportState = &BrokerPoolResource{}
)

// BrokerPoolResource manages a broker pool. Its tokens (including the "default" token Britive
// creates with every pool) are managed by BrokerPoolTokenResource and its labels by
// BrokerPoolLabelsResource. The pool's brokers and linked resources are exposed only by the
// britive_resource_manager_broker_pool data source.
type BrokerPoolResource struct {
	client *britive.Client
}

// BrokerPoolResourceModel describes the resource data model.
type BrokerPoolResourceModel struct {
	ID                types.String `tfsdk:"id"`
	PoolID            types.String `tfsdk:"pool_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	KeepAliveSeconds  types.Int64  `tfsdk:"keep_alive_seconds"`
	DisconnectSeconds types.Int64  `tfsdk:"disconnect_seconds"`
	BrokerCount       types.Int64  `tfsdk:"broker_count"`
}

// NewBrokerPoolResource is a helper function to simplify the provider implementation.
func NewBrokerPoolResource() resource.Resource {
	return &BrokerPoolResource{}
}

// Metadata returns the resource type name.
func (r *BrokerPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_broker_pool"
}

// Schema defines the schema for the resource.
func (r *BrokerPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager broker pool. Its default token is managed with britive_resource_manager_broker_pool_token and name \"default\".",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the broker pool, in the form resource-manager/broker-pools/{pool_id}.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.StringAttribute{
				Description: "The unique identifier of the broker pool. Pass this to britive_resource_manager_broker_pool_token.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the broker pool. Can be changed in place.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the broker pool. Defaults to an empty string; removing it clears the description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"keep_alive_seconds": schema.Int64Attribute{
				Description: "How often, in seconds, brokers check in with Britive. Defaults to 60; removing it resets it to 60.",
				Optional:    true,
				Computed:    true,
				Default:     staticInt64Default{brokerpool.KeepAliveSecondsDefault},
			},
			"disconnect_seconds": schema.Int64Attribute{
				Description: "After how many seconds without a check-in a broker is marked disconnected. Defaults to 86400; removing it resets it to 86400.",
				Optional:    true,
				Computed:    true,
				Default:     staticInt64Default{brokerpool.DisconnectSecondsDefault},
				Validators: []validator.Int64{
					brokerpool.Int64Between(1, math.MaxInt32),
				},
			},
			"broker_count": schema.Int64Attribute{
				Description: "The number of brokers in the pool.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *BrokerPoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the pool with its timings. Nothing is read back: the create response plus the
// plan already give the full state. Calls: POST pool, plus PUT when the API ignored the timings.
func (r *BrokerPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BrokerPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Creating broker pool: %s", plan.Name.ValueString())

	request := britive.RemoteBrokerPoolCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}
	if isKnownInt64(plan.KeepAliveSeconds) {
		request.KeepAliveSeconds = int(plan.KeepAliveSeconds.ValueInt64())
	}
	if isKnownInt64(plan.DisconnectSeconds) {
		request.DisconnectSeconds = int(plan.DisconnectSeconds.ValueInt64())
	}

	pool, err := r.client.CreateBrokerPool(request)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Broker Pool", err.Error())
		return
	}
	poolID := pool.PoolID

	// Any failure after this point would leave a pool Terraform doesn't know about, so the
	// half-created pool is deleted before the error is returned.
	fail := func(summary string, err error) {
		if delErr := r.client.DeleteBrokerPool(poolID); delErr != nil {
			resp.Diagnostics.AddError(summary, fmt.Sprintf("%s\n\nThe broker pool %s was created but could not be cleaned up: %s", err.Error(), poolID, delErr.Error()))
			return
		}
		resp.Diagnostics.AddError(summary, err.Error())
	}

	// The API docs list the timings in the create body, but the UI never sends them. If the
	// API ignored them, set them with an update, which costs one call only in that case.
	if timingsDiffer(plan, pool) {
		updated, err := r.client.UpdateBrokerPool(poolID, britive.RemoteBrokerPoolUpdateRequest{
			Name:              plan.Name.ValueString(),
			Description:       plan.Description.ValueString(),
			KeepAliveSeconds:  int(int64OrDefault(plan.KeepAliveSeconds, int64(pool.KeepAliveSeconds))),
			DisconnectSeconds: int(int64OrDefault(plan.DisconnectSeconds, int64(pool.DisconnectSeconds))),
		})
		if err != nil {
			fail("Error Setting Broker Pool Timings", err)
			return
		}
		pool = updated
	}

	// The create response has no description, so the planned one (which was sent) is kept.
	description := plan.Description
	applyBrokerPool(poolID, pool, &plan)
	plan.Description = description

	log.Printf("[INFO] Created broker pool: %s", poolID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *BrokerPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BrokerPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := state.PoolID.ValueString()
	log.Printf("[INFO] Reading broker pool: %s", poolID)

	found, err := r.refresh(poolID, &state)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool", err.Error())
		return
	}
	if !found {
		log.Printf("[WARN] Broker pool %s not found, removing from state", poolID)
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the pool's details. The pool response is used as-is.
// Calls: PUT pool when name/description/timings changed, otherwise GET pool (for broker_count).
func (r *BrokerPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state BrokerPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := state.PoolID.ValueString()
	log.Printf("[INFO] Updating broker pool: %s", poolID)

	// The update call replaces the whole pool, so every field is sent whenever any changed.
	var pool *britive.RemoteBrokerPool
	var err error
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) ||
		!plan.KeepAliveSeconds.Equal(state.KeepAliveSeconds) || !plan.DisconnectSeconds.Equal(state.DisconnectSeconds) {
		pool, err = r.client.UpdateBrokerPool(poolID, britive.RemoteBrokerPoolUpdateRequest{
			Name:              plan.Name.ValueString(),
			Description:       plan.Description.ValueString(),
			KeepAliveSeconds:  int(int64OrDefault(plan.KeepAliveSeconds, state.KeepAliveSeconds.ValueInt64())),
			DisconnectSeconds: int(int64OrDefault(plan.DisconnectSeconds, state.DisconnectSeconds.ValueInt64())),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Broker Pool", err.Error())
			return
		}
	} else {
		pool, err = r.client.GetBrokerPool(poolID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Broker Pool", err.Error())
			return
		}
	}

	// The planned description (which was sent) is kept, as on create: the API may leave it
	// out of the response.
	description := plan.Description
	applyBrokerPool(poolID, pool, &plan)
	plan.Description = description

	log.Printf("[INFO] Updated broker pool: %s", poolID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the pool. The API also deletes its tokens, labels and resource links.
func (r *BrokerPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BrokerPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := state.PoolID.ValueString()
	log.Printf("[INFO] Deleting broker pool: %s", poolID)

	if err := r.client.DeleteBrokerPool(poolID); err != nil {
		resp.Diagnostics.AddError("Error Deleting Broker Pool", err.Error())
		return
	}

	log.Printf("[INFO] Deleted broker pool: %s", poolID)
}

// ImportState accepts "resource-manager/broker-pools/{pool_id}" or "{pool_id}". The framework
// calls Read right after, which fills in everything else and fails the import if the pool does
// not exist, so no lookup is made here.
func (r *BrokerPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	poolID, err := parseBrokerPoolImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), brokerpool.PoolCompositeID(poolID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pool_id"), poolID)...)
}

// Helper functions

// refresh reads the pool into model.
// It returns false when the pool no longer exists. Used by Read.
// Calls: GET pool.
func (r *BrokerPoolResource) refresh(poolID string, model *BrokerPoolResourceModel) (bool, error) {
	pool, err := r.client.GetBrokerPool(poolID)
	if errors.Is(err, britive.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	applyBrokerPool(poolID, pool, model)

	return true, nil
}

// applyBrokerPool copies the API's pool fields onto model.
func applyBrokerPool(poolID string, pool *britive.RemoteBrokerPool, model *BrokerPoolResourceModel) {
	model.ID = types.StringValue(brokerpool.PoolCompositeID(poolID))
	model.PoolID = types.StringValue(poolID)
	model.Name = types.StringValue(pool.Name)
	model.Description = types.StringValue(pool.Description)
	model.KeepAliveSeconds = types.Int64Value(int64(pool.KeepAliveSeconds))
	model.DisconnectSeconds = types.Int64Value(int64(pool.DisconnectSeconds))
	model.BrokerCount = types.Int64Value(int64(pool.BrokerCount))
}

// timingsDiffer reports whether the API's pool has different timings than the configured ones.
func timingsDiffer(plan BrokerPoolResourceModel, pool *britive.RemoteBrokerPool) bool {
	return (isKnownInt64(plan.KeepAliveSeconds) && plan.KeepAliveSeconds.ValueInt64() != int64(pool.KeepAliveSeconds)) ||
		(isKnownInt64(plan.DisconnectSeconds) && plan.DisconnectSeconds.ValueInt64() != int64(pool.DisconnectSeconds))
}

// int64OrDefault returns v when it is known, otherwise fallback.
func int64OrDefault(v types.Int64, fallback int64) int64 {
	if isKnownInt64(v) {
		return v.ValueInt64()
	}
	return fallback
}

// parseBrokerPoolImportID extracts the pool ID from "resource-manager/broker-pools/{pool_id}"
// or a bare "{pool_id}".
func parseBrokerPoolImportID(id string) (string, error) {
	parts := strings.Split(id, "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		return parts[0], nil
	case len(parts) == 3 && parts[0] == "resource-manager" && parts[1] == "broker-pools" && parts[2] != "":
		return parts[2], nil
	default:
		return "", fmt.Errorf("import ID must be 'resource-manager/broker-pools/{pool_id}' or '{pool_id}', got: %s", id)
	}
}

func isKnownInt64(v types.Int64) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
