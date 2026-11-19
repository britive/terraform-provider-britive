package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/brokerpool"
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
	_ resource.Resource                = &BrokerPoolTokenResource{}
	_ resource.ResourceWithConfigure   = &BrokerPoolTokenResource{}
	_ resource.ResourceWithImportState = &BrokerPoolTokenResource{}
)

// BrokerPoolTokenResource manages a token of a broker pool. A resource named "default" takes
// over the pool's default token, which Britive creates with every pool, instead of creating a
// new one.
//
// The token value is never kept in the state (not even in private state, which is also written
// to the state file): the API returns it on every read, and the
// britive_resource_manager_broker_pool_token ephemeral resource reads it when it is needed.
type BrokerPoolTokenResource struct {
	client *britive.Client
}

// BrokerPoolTokenResourceModel describes the resource data model.
type BrokerPoolTokenResourceModel struct {
	ID          types.String `tfsdk:"id"`
	PoolID      types.String `tfsdk:"pool_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	Created     types.String `tfsdk:"created"`
}

// NewBrokerPoolTokenResource is a helper function to simplify the provider implementation.
func NewBrokerPoolTokenResource() resource.Resource {
	return &BrokerPoolTokenResource{}
}

// Metadata returns the resource type name.
func (r *BrokerPoolTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_broker_pool_token"
}

// Schema defines the schema for the resource.
func (r *BrokerPoolTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a token of a Britive resource manager broker pool. With name \"default\", it manages the pool's default token, which Britive creates with every pool. The token value is not stored in the state; read it with the britive_resource_manager_broker_pool_token ephemeral resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the token, in the form resource-manager/broker-pools/{pool_id}/tokens/{name}.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.StringAttribute{
				Description: "The ID of the broker pool the token belongs to. Changing it creates a new token.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The token name. Only letters, numbers, underscore and hyphen; unique in the pool. \"default\" manages the pool's default token. Changing it creates a new token.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(brokerpool.NamePattern, "must contain only letters, numbers, underscore and hyphen"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The token description. If not set, the description is empty.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The token status: ACTIVE or INACTIVE.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(brokerpool.TokenStatusActive, brokerpool.TokenStatusInactive),
				},
			},
			"created": schema.StringAttribute{
				Description: "When the token was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *BrokerPoolTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the token. The API always creates tokens INACTIVE, so an ACTIVE status is
// set with a follow-up update. Calls: POST, plus PATCH when the status is ACTIVE.
// The "default" token is handled by createDefault instead.
func (r *BrokerPoolTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BrokerPoolTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(plan.PoolID.ValueString())
	name := plan.Name.ValueString()

	if name == brokerpool.DefaultTokenName {
		r.createDefault(ctx, poolID, plan, resp)
		return
	}

	log.Printf("[INFO] Creating broker pool token %s in pool %s", name, poolID)

	created, err := r.client.CreateBrokerPoolToken(poolID, britive.RemoteBrokerPoolTokenCreateRequest{
		TokenName:   name,
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Broker Pool Token", err.Error())
		return
	}

	// The create and update responses both carry the full token, so no read-back is needed.
	token := created
	if plan.Status.ValueString() != created.Status {
		status := plan.Status.ValueString()
		updated, err := r.client.UpdateBrokerPoolToken(poolID, name, britive.RemoteBrokerPoolTokenUpdateRequest{Status: &status})
		if err != nil {
			// Don't leave a token Terraform doesn't know about.
			if delErr := r.client.DeleteBrokerPoolToken(poolID, name); delErr != nil {
				resp.Diagnostics.AddError("Error Setting Broker Pool Token Status", fmt.Sprintf("%s\n\nThe token %s was created but could not be cleaned up: %s", err.Error(), name, delErr.Error()))
				return
			}
			resp.Diagnostics.AddError("Error Setting Broker Pool Token Status", err.Error())
			return
		}
		token = updated
	}

	plan.ID = types.StringValue(brokerPoolTokenCompositeID(poolID, name))
	mapBrokerPoolToken(token, &plan)

	log.Printf("[INFO] Created broker pool token %s in pool %s", name, poolID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// createDefault takes over the pool's default token: it sets the configured status and
// description on the existing token with one PATCH, or creates the token again with POST when
// it was deleted. Calls: GET tokens, plus POST when it is missing, plus PATCH when a value
// differs.
//
// When Britive rejects the PATCH (for example, INACTIVE on the last active token), the error is
// returned and nothing is saved, so the resource is not tainted and the next apply takes over
// the token again. The token itself is never deleted here.
func (r *BrokerPoolTokenResource) createDefault(ctx context.Context, poolID string, plan BrokerPoolTokenResourceModel, resp *resource.CreateResponse) {
	name := brokerpool.DefaultTokenName
	description := plan.Description.ValueString()
	status := plan.Status.ValueString()

	token, err := r.client.GetBrokerPoolToken(poolID, name)
	switch {
	case errors.Is(err, britive.ErrNotFound):
		log.Printf("[INFO] Default token of broker pool %s not found, creating it", poolID)
		token, err = r.client.CreateBrokerPoolToken(poolID, britive.RemoteBrokerPoolTokenCreateRequest{
			TokenName:   name,
			Description: description,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error Creating Broker Pool Default Token", err.Error())
			return
		}
	case err != nil:
		resp.Diagnostics.AddError("Error Reading Broker Pool Default Token", err.Error())
		return
	default:
		log.Printf("[INFO] Taking over the default token of broker pool %s", poolID)
	}

	var update britive.RemoteBrokerPoolTokenUpdateRequest
	changed := false
	if token.Status != status {
		update.Status = &status
		changed = true
	}
	if token.Description != description {
		update.Description = &description
		changed = true
	}

	if changed {
		updated, err := r.client.UpdateBrokerPoolToken(poolID, name, update)
		if err != nil {
			// Nothing is saved, so the next apply takes over the token again. The token is not
			// deleted: brokers may use it.
			resp.Diagnostics.AddError(
				"Error Updating Broker Pool Default Token",
				fmt.Sprintf("Britive rejected the update of the default token of broker pool %s: %s\n\n"+
					"The token was not changed and the resource was not saved. Fix the configuration "+
					"(for example, make another token ACTIVE first) and apply again.", poolID, err.Error()),
			)
			return
		}
		token = updated
	}

	plan.ID = types.StringValue(brokerPoolTokenCompositeID(poolID, name))
	mapBrokerPoolToken(token, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *BrokerPoolTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BrokerPoolTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(state.PoolID.ValueString())
	name := state.Name.ValueString()

	log.Printf("[INFO] Reading broker pool token %s in pool %s", name, poolID)

	token, err := r.client.GetBrokerPoolToken(poolID, name)
	if errors.Is(err, britive.ErrNotFound) {
		log.Printf("[WARN] Broker pool token %s in pool %s not found, removing from state", name, poolID)
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Token", err.Error())
		return
	}

	state.ID = types.StringValue(brokerPoolTokenCompositeID(poolID, name))
	mapBrokerPoolToken(token, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the token's status and/or description in one call.
func (r *BrokerPoolTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state BrokerPoolTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(state.PoolID.ValueString())
	name := state.Name.ValueString()

	var update britive.RemoteBrokerPoolTokenUpdateRequest
	changed := false
	if !plan.Status.Equal(state.Status) {
		status := plan.Status.ValueString()
		update.Status = &status
		changed = true
	}
	if !plan.Description.Equal(state.Description) {
		// A removed description is sent as "" to clear it.
		description := plan.Description.ValueString()
		update.Description = &description
		changed = true
	}

	plan.ID = state.ID

	// Only status and description can change in place (pool_id and name force a new token),
	// and the update response carries the full token, so no read-back is needed.
	if changed {
		log.Printf("[INFO] Updating broker pool token %s in pool %s", name, poolID)
		token, err := r.client.UpdateBrokerPoolToken(poolID, name, update)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Broker Pool Token", err.Error())
			return
		}
		mapBrokerPoolToken(token, &plan)
	} else {
		plan.Created = state.Created
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the token. A token that is already gone (for example because its pool was
// deleted first) counts as deleted. When Britive rejects deleting the "default" token (for
// example, because it is the last active token), the resource is removed from the state with a
// warning; the token stays in Britive and is deleted together with the pool.
func (r *BrokerPoolTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BrokerPoolTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(state.PoolID.ValueString())
	name := state.Name.ValueString()

	log.Printf("[INFO] Deleting broker pool token %s in pool %s", name, poolID)

	err := r.client.DeleteBrokerPoolToken(poolID, name)
	if err == nil {
		return
	}
	if name == brokerpool.DefaultTokenName && isBritiveRejection(err) {
		resp.Diagnostics.AddWarning(
			"Broker Pool Default Token Not Deleted",
			fmt.Sprintf("Britive rejected deleting the default token of broker pool %s: %s\n\n"+
				"The resource was removed from the Terraform state. The token stays in Britive and is deleted together with the pool.", poolID, err.Error()),
		)
		return
	}
	resp.Diagnostics.AddError("Error Deleting Broker Pool Token", err.Error())
}

// ImportState accepts "resource-manager/broker-pools/{pool_id}/tokens/{name}" or
// "{pool_id}/{name}". The framework calls Read right after, which reads the token and fails the
// import if it does not exist, so no lookup is made here.
func (r *BrokerPoolTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	poolID, name, err := parseBrokerPoolTokenImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), brokerPoolTokenCompositeID(poolID, name))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pool_id"), poolID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

// Helper functions

// britiveRejectionPattern matches the "<errorCode>: <message>" errors the client returns when
// Britive answers with an error body, for example "E1001: Cannot delete the last active token.".
var britiveRejectionPattern = regexp.MustCompile(`^[A-Za-z]+-?[0-9]+: `)

// isBritiveRejection reports whether err is an error answer from Britive, as opposed to a
// network or client error.
func isBritiveRejection(err error) bool {
	return err != nil && britiveRejectionPattern.MatchString(err.Error())
}

// mapBrokerPoolToken copies the API's token onto model. description keeps null when the
// user never set it and the API returns none. The token value is deliberately not copied.
func mapBrokerPoolToken(token *britive.RemoteBrokerPoolToken, model *BrokerPoolTokenResourceModel) {
	model.Description = preserveOptionalString(token.Description, model.Description)
	model.Status = types.StringValue(token.Status)
	model.Created = types.StringValue(token.Created)
}

func brokerPoolTokenCompositeID(poolID, name string) string {
	return brokerpool.TokenCompositeID(poolID, name)
}

// parseBrokerPoolTokenImportID extracts the pool ID and token name from
// "resource-manager/broker-pools/{pool_id}/tokens/{name}" or "{pool_id}/{name}".
func parseBrokerPoolTokenImportID(id string) (string, string, error) {
	return parseBrokerPoolChildImportID(id, "tokens", "name")
}

// parseBrokerPoolChildImportID extracts the pool ID and child name from
// "resource-manager/broker-pools/{pool_id}/{kind}/{name}" or "{pool_id}/{name}".
func parseBrokerPoolChildImportID(id, kind, nameLabel string) (string, string, error) {
	poolID, name, err := brokerpool.ParseChildID(id, kind, nameLabel)
	if err != nil {
		return "", "", fmt.Errorf("import ID %w", err)
	}
	return poolID, name, nil
}
