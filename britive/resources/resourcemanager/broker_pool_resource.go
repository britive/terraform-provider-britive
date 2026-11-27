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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                   = &BrokerPoolResource{}
	_ resource.ResourceWithConfigure      = &BrokerPoolResource{}
	_ resource.ResourceWithImportState    = &BrokerPoolResource{}
	_ resource.ResourceWithValidateConfig = &BrokerPoolResource{}
	_ resource.ResourceWithModifyPlan     = &BrokerPoolResource{}
)

// BrokerPoolResource manages a broker pool and its tokens (including the "default" token
// Britive creates with every pool). The tokens argument is required. Its labels are managed by
// BrokerPoolLabelsResource. The pool's brokers and linked resources are exposed only by the
// britive_resource_manager_broker_pool data source.
//
// Tokens are not a separate resource because Britive never lets a pool drop below one ACTIVE
// token: a separate token resource could not be destroyed before its pool, and Terraform does
// not order changes across separate resources. Here one DELETE removes the pool with its
// tokens, and the provider orders token changes itself (see brokerpool.NextTokenStep).
//
// Token values are never kept in the state (not even in private state, which is also written to
// the state file). The britive_resource_manager_broker_pool_token ephemeral resource reads them.
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
	Tokens            types.Map    `tfsdk:"tokens"`
}

// maxTokenSteps stops the token changes if they ever loop. Any valid change needs at most 6.
const maxTokenSteps = 12

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
		Description: "Manages a Britive resource manager broker pool and its tokens. Token values are not stored in the state; read them with the britive_resource_manager_broker_pool_token ephemeral resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the broker pool, in the form resource-manager/broker-pools/{pool_id}.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.StringAttribute{
				Description: "The unique identifier of the broker pool.",
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
			"tokens": schema.MapNestedAttribute{
				Description: "The tokens of the pool, as a map with the token name as the key. The map is the exact set of tokens: a token in Britive that is not in the map is deleted. Use the key \"default\" to keep the default token Britive creates with every pool. At least 1 token. At most 3 tokens, at most 2 ACTIVE and at least 1 ACTIVE (Britive's limits, checked in the plan). When the pool is created, it must include \"default\"; after that, the default token can be changed or removed like any other token.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"status": schema.StringAttribute{
							Description: "The token status: ACTIVE or INACTIVE.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(brokerpool.TokenStatusActive, brokerpool.TokenStatusInactive),
							},
						},
						"description": schema.StringAttribute{
							Description: "The token description. Defaults to an empty string.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
						"id": schema.StringAttribute{
							Description: "The token ID, in the form {pool_id}/{token_name}. Pass it to the token_id of the britive_resource_manager_broker_pool_token ephemeral resource. Unknown until the token exists.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								// UseNonNullStateForUnknown, not UseStateForUnknown: a token added to an
								// existing pool has a null prior value, which must stay unknown.
								stringplanmodifier.UseNonNullStateForUnknown(),
							},
						},
						"created": schema.StringAttribute{
							Description: "When the token was created.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								// UseNonNullStateForUnknown, not UseStateForUnknown: a token added to an
								// existing pool has a null prior value, which must stay unknown.
								stringplanmodifier.UseNonNullStateForUnknown(),
							},
						},
					},
				},
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

// ValidateConfig checks the tokens map in the plan, before any API call: token names, and
// Britive's limits (at least 1 token, at most 3 tokens, at most 2 ACTIVE, at least 1 ACTIVE).
// The limits must be checked here: tokens are changed one call at a time, so Britive would
// only reject a set that breaks them part-way through the apply.
func (r *BrokerPoolResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var tokens types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("tokens"), &tokens)...)
	if resp.Diagnostics.HasError() || tokens.IsNull() || tokens.IsUnknown() {
		return
	}

	desired := make(map[string]brokerpool.TokenSpec, len(tokens.Elements()))
	for name, element := range tokens.Elements() {
		if !brokerpool.NamePattern.MatchString(name) {
			resp.Diagnostics.AddAttributeError(
				path.Root("tokens").AtMapKey(name),
				"Invalid Token Name",
				fmt.Sprintf("Token name %q must contain only letters, numbers, underscore and hyphen.", name),
			)
		}
		spec := brokerpool.TokenSpec{}
		object, ok := element.(types.Object)
		if ok && !object.IsNull() && !object.IsUnknown() {
			if status, ok := object.Attributes()["status"].(types.String); ok && !status.IsNull() && !status.IsUnknown() {
				spec.Status = status.ValueString()
			}
		}
		desired[name] = spec
	}

	for _, problem := range brokerpool.ValidateTokens(desired) {
		resp.Diagnostics.AddAttributeError(path.Root("tokens"), "Invalid Broker Pool Tokens", problem)
	}
}

// ModifyPlan requires the default token in tokens when the pool is created. Britive creates
// the default token with every pool, so the configuration must say how to keep it. After the
// pool exists, the default token is changed or removed like any other token.
func (r *BrokerPoolResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only on create: no prior state, and not a destroy.
	if !req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var tokens types.Map
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("tokens"), &tokens)...)
	if resp.Diagnostics.HasError() || tokens.IsNull() || tokens.IsUnknown() {
		return
	}
	if _, ok := tokens.Elements()[brokerpool.DefaultTokenName]; !ok {
		resp.Diagnostics.AddAttributeError(
			path.Root("tokens"),
			"Missing Default Broker Pool Token",
			"tokens must include \"default\" when the pool is created, for example: default = { status = \"ACTIVE\" }",
		)
	}
}

// Create creates the pool with its timings and changes the pool's tokens to match tokens.
// Calls: POST pool, plus PUT when the API ignored the timings, plus GET tokens and one call per
// token change.
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
	// It also carries the value of the default token, which is dropped here.
	description := plan.Description
	applyBrokerPool(poolID, pool, &plan)
	plan.Description = description

	log.Printf("[INFO] Created broker pool: %s", poolID)

	r.applyTokens(ctx, poolID, &plan, &resp.Diagnostics)

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

	tokens, err := r.client.GetBrokerPoolTokens(poolID)
	if errors.Is(err, britive.ErrNotFound) {
		log.Printf("[WARN] Broker pool %s not found, removing from state", poolID)
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Tokens", err.Error())
		return
	}
	state.Tokens = brokerpool.TokensToMap(ctx, poolID, tokens, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the pool's details. The pool response is used as-is. When tokens differs from
// the state, the pool's tokens are changed to match it.
// Calls: PUT pool when name/description/timings changed, otherwise GET pool (for broker_count),
// plus GET tokens and one call per token change when the tokens changed.
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

	if plan.Tokens.Equal(state.Tokens) {
		plan.Tokens = state.Tokens
	} else {
		r.applyTokens(ctx, poolID, &plan, &resp.Diagnostics)
	}

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
// not exist, so no lookup is made here. Read also reads the name, status and description of
// each token (never the value).
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

// applyTokens changes the pool's tokens to match plan.Tokens and puts the tokens Britive holds
// afterwards into plan.Tokens. If a call fails, plan.Tokens holds the tokens Britive really has
// and an error is added, so the next plan shows what is left to change.
func (r *BrokerPoolResource) applyTokens(ctx context.Context, poolID string, plan *BrokerPoolResourceModel, diags *diag.Diagnostics) {
	desired := make(map[string]brokerpool.TokenModel, len(plan.Tokens.Elements()))
	diags.Append(plan.Tokens.ElementsAs(ctx, &desired, false)...)
	if diags.HasError() {
		return
	}
	specs := make(map[string]brokerpool.TokenSpec, len(desired))
	for name, token := range desired {
		specs[name] = brokerpool.TokenSpec{Status: token.Status.ValueString(), Description: token.Description.ValueString()}
	}

	// ValidateConfig skips the ACTIVE checks while a status is unknown, so check the set again
	// now that every status is known, before any call changes the pool's tokens.
	if problems := brokerpool.ValidateTokens(specs); len(problems) > 0 {
		held, err := r.client.GetBrokerPoolTokens(poolID)
		if err == nil {
			plan.Tokens = brokerpool.TokensToMap(ctx, poolID, held, diags)
		}
		diags.AddError("Invalid Broker Pool Tokens", strings.Join(problems, " "))
		return
	}

	held, err := r.syncTokens(poolID, specs)
	plan.Tokens = brokerpool.TokensToMap(ctx, poolID, held, diags)
	if err != nil {
		diags.AddError("Error Updating Broker Pool Tokens", err.Error())
	}
}

// syncTokens makes one API call at a time until the pool's tokens match desired, and returns
// the tokens Britive holds afterwards. The order of the calls comes from
// brokerpool.NextTokenStep, so every call is one Britive accepts. On an error, the tokens are
// read again so the caller can save what Britive really holds; if that read fails too, nil is
// returned, which leaves an empty map in the state until the next refresh reads them.
// Calls: GET tokens, plus one POST, PATCH or DELETE per step.
func (r *BrokerPoolResource) syncTokens(poolID string, desired map[string]brokerpool.TokenSpec) ([]britive.RemoteBrokerPoolToken, error) {
	held, err := r.client.GetBrokerPoolTokens(poolID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < maxTokenSteps; i++ {
		step, ok := brokerpool.NextTokenStep(brokerPoolTokenSpecs(held), desired)
		if !ok {
			return held, nil
		}
		log.Printf("[INFO] Broker pool %s tokens: %s", poolID, step)
		held, err = r.applyTokenStep(poolID, step, held)
		if err != nil {
			if fresh, readErr := r.client.GetBrokerPoolTokens(poolID); readErr == nil {
				held = fresh
			} else {
				held = nil
			}
			return held, fmt.Errorf("Could not %s: %w", step.Describe(), err)
		}
	}
	return held, fmt.Errorf("The tokens still do not match the configuration after %d changes.", maxTokenSteps)
}

// applyTokenStep makes the API call for step and returns held with its result.
func (r *BrokerPoolResource) applyTokenStep(poolID string, step brokerpool.TokenStep, held []britive.RemoteBrokerPoolToken) ([]britive.RemoteBrokerPoolToken, error) {
	switch step.Op {
	case brokerpool.TokenOpDelete:
		if err := r.client.DeleteBrokerPoolToken(poolID, step.Name); err != nil {
			return held, err
		}
		next := make([]britive.RemoteBrokerPoolToken, 0, len(held))
		for _, token := range held {
			if token.TokenName != step.Name {
				next = append(next, token)
			}
		}
		return next, nil

	case brokerpool.TokenOpCreate:
		request := britive.RemoteBrokerPoolTokenCreateRequest{TokenName: step.Name}
		if step.Description != nil {
			request.Description = *step.Description
		}
		created, err := r.client.CreateBrokerPoolToken(poolID, request)
		if err != nil {
			return held, err
		}
		token := *created
		token.TokenName = step.Name
		if token.Status == "" {
			token.Status = brokerpool.TokenStatusInactive
		}
		if token.Description == "" {
			token.Description = request.Description
		}
		return append(held, token), nil

	default:
		updated, err := r.client.UpdateBrokerPoolToken(poolID, step.Name, britive.RemoteBrokerPoolTokenUpdateRequest{
			Status:      step.Status,
			Description: step.Description,
		})
		if err != nil {
			return held, err
		}
		next := make([]britive.RemoteBrokerPoolToken, 0, len(held))
		for _, token := range held {
			if token.TokenName == step.Name {
				merged := *updated
				merged.TokenName = step.Name
				if step.Status != nil {
					merged.Status = *step.Status
				} else if merged.Status == "" {
					merged.Status = token.Status
				}
				if step.Description != nil {
					merged.Description = *step.Description
				} else if merged.Description == "" {
					merged.Description = token.Description
				}
				if merged.Created == "" {
					merged.Created = token.Created
				}
				token = merged
			}
			next = append(next, token)
		}
		return next, nil
	}
}

// brokerPoolTokenSpecs returns the managed fields of the API's tokens, by name.
func brokerPoolTokenSpecs(tokens []britive.RemoteBrokerPoolToken) map[string]brokerpool.TokenSpec {
	specs := make(map[string]brokerpool.TokenSpec, len(tokens))
	for _, token := range tokens {
		specs[token.TokenName] = brokerpool.TokenSpec{Status: token.Status, Description: token.Description}
	}
	return specs
}

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
