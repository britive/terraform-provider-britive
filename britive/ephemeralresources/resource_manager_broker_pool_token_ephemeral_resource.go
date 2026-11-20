// Package ephemeralresources holds the provider's ephemeral resources. Terraform opens an
// ephemeral resource on every plan and apply and keeps its result in memory for that run only,
// so values read here are never written to the state or to a plan file.
package ephemeralresources

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/brokerpool"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResource              = &BrokerPoolTokenEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &BrokerPoolTokenEphemeralResource{}
)

// BrokerPoolTokenEphemeralResource reads the value of a broker pool token, including the
// pool's "default" token. The pool resource does not keep token values in the state; this is
// the only way to get one in Terraform.
type BrokerPoolTokenEphemeralResource struct {
	client *britive.Client
}

// BrokerPoolTokenEphemeralResourceModel describes the ephemeral resource data model.
type BrokerPoolTokenEphemeralResourceModel struct {
	TokenID    types.String `tfsdk:"token_id"`
	PoolID     types.String `tfsdk:"pool_id"`
	Name       types.String `tfsdk:"name"`
	Status     types.String `tfsdk:"status"`
	TokenValue types.String `tfsdk:"token_value"`
}

// NewBrokerPoolTokenEphemeralResource is a helper function to simplify the provider implementation.
func NewBrokerPoolTokenEphemeralResource() ephemeral.EphemeralResource {
	return &BrokerPoolTokenEphemeralResource{}
}

// Metadata returns the ephemeral resource type name.
func (r *BrokerPoolTokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_broker_pool_token"
}

// Schema defines the schema for the ephemeral resource.
func (r *BrokerPoolTokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the value of a Britive resource manager broker pool token. The value is never stored in the state or in a plan file.",
		Attributes: map[string]schema.Attribute{
			"token_id": schema.StringAttribute{
				Description: "The token to read, as {pool_id}/{token_name}: the id of a token in the tokens map of a britive_resource_manager_broker_pool resource. Use {pool_id}/default for the pool's default token.",
				Required:    true,
			},
			"pool_id": schema.StringAttribute{
				Description: "The ID of the broker pool.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The token name.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The token status: ACTIVE or INACTIVE.",
				Computed:    true,
			},
			"token_value": schema.StringAttribute{
				Description: "The secret value brokers use to join the pool.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure adds the provider configured client to the ephemeral resource.
func (r *BrokerPoolTokenEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Open reads the token. The API has no call that reads one token (GET .../tokens/{name}
// returns HTTP 500), so the pool's tokens are listed and the token is found by name.
// Calls: GET tokens.
func (r *BrokerPoolTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data BrokerPoolTokenEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID, name, err := brokerpool.ParseTokenID(data.TokenID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			pathTokenID, "Invalid Token ID", fmt.Sprintf("token_id %s", err.Error()),
		)
		return
	}

	log.Printf("[INFO] Reading value of broker pool token %s in pool %s", name, poolID)

	token, err := r.client.GetBrokerPoolToken(poolID, name)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Broker Pool Token Not Found",
			fmt.Sprintf("Broker pool %s has no token named %q. Add it to the pool's tokens, or read a token that exists.", poolID, name),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Token", err.Error())
		return
	}

	data.PoolID = types.StringValue(poolID)
	data.Name = types.StringValue(name)
	data.Status = types.StringValue(token.Status)
	data.TokenValue = types.StringValue(token.TokenValue)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

var pathTokenID = path.Root("token_id")
