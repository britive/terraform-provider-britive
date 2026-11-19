package datasources

import (
	"context"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/brokerpool"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ResourceManagerBrokerPoolDataSource{}
	_ datasource.DataSourceWithConfigure = &ResourceManagerBrokerPoolDataSource{}
)

// NewResourceManagerBrokerPoolDataSource is a helper function to simplify the provider implementation.
func NewResourceManagerBrokerPoolDataSource() datasource.DataSource {
	return &ResourceManagerBrokerPoolDataSource{}
}

// ResourceManagerBrokerPoolDataSource looks up an existing broker pool by its exact name,
// together with its active brokers and linked resources. It never returns token values.
type ResourceManagerBrokerPoolDataSource struct {
	client *britive.Client
}

// ResourceManagerBrokerPoolDataSourceModel describes the data source data model.
type ResourceManagerBrokerPoolDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	PoolID            types.String `tfsdk:"pool_id"`
	Description       types.String `tfsdk:"description"`
	KeepAliveSeconds  types.Int64  `tfsdk:"keep_alive_seconds"`
	DisconnectSeconds types.Int64  `tfsdk:"disconnect_seconds"`
	BrokerCount       types.Int64  `tfsdk:"broker_count"`
	Brokers           types.List   `tfsdk:"brokers"`
	Resources         types.List   `tfsdk:"resources"`
}

// Metadata returns the data source type name.
func (d *ResourceManagerBrokerPoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_broker_pool"
}

// Schema defines the schema for the data source.
func (d *ResourceManagerBrokerPoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an existing Britive resource manager broker pool by its exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The identifier of the broker pool, in the form resource-manager/broker-pools/{pool_id}.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The exact name of the broker pool.",
			},
			"pool_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the broker pool.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the broker pool.",
			},
			"keep_alive_seconds": schema.Int64Attribute{
				Computed:    true,
				Description: "How often, in seconds, brokers check in with Britive.",
			},
			"disconnect_seconds": schema.Int64Attribute{
				Computed:    true,
				Description: "After how many seconds without a check-in a broker is marked disconnected.",
			},
			"broker_count": schema.Int64Attribute{
				Computed:    true,
				Description: "The number of brokers in the pool.",
			},
			"brokers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The active brokers in the pool.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"broker_id":        schema.StringAttribute{Computed: true, Description: "The broker ID."},
						"broker_name":      schema.StringAttribute{Computed: true, Description: "The broker name."},
						"status":           schema.StringAttribute{Computed: true, Description: "The broker status."},
						"status_timestamp": schema.StringAttribute{Computed: true, Description: "When the broker status last changed."},
						"created":          schema.StringAttribute{Computed: true, Description: "When the broker joined the pool."},
					},
				},
			},
			"resources": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The resources linked to the pool.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"resource_id":        schema.StringAttribute{Computed: true, Description: "The resource ID."},
						"name":               schema.StringAttribute{Computed: true, Description: "The resource name."},
						"resource_type_name": schema.StringAttribute{Computed: true, Description: "The resource type of the resource."},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ResourceManagerBrokerPoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read finds the pool by exact name, then reads its active brokers and linked resources.
func (d *ResourceManagerBrokerPoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceManagerBrokerPoolDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	// The name filter matches names that start with the value, so an exact match is picked out.
	pools, err := d.client.ListBrokerPools(name)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pools", err.Error())
		return
	}

	var matches []britive.RemoteBrokerPool
	for _, pool := range pools {
		if pool.Name == name {
			matches = append(matches, pool)
		}
	}

	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Broker Pool Not Found", fmt.Sprintf("No broker pool named %q was found.", name))
		return
	case 1:
	default:
		resp.Diagnostics.AddError("Multiple Broker Pools Found", fmt.Sprintf("%d broker pools are named %q. Pool names must be unique for this lookup.", len(matches), name))
		return
	}
	pool := matches[0]

	brokers, err := d.client.GetBrokerPoolActiveBrokers(pool.PoolID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Brokers", err.Error())
		return
	}

	linked, err := d.client.GetBrokerPoolLinkedResources(pool.PoolID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Resources", err.Error())
		return
	}

	data.ID = types.StringValue(brokerpool.PoolCompositeID(pool.PoolID))
	data.PoolID = types.StringValue(pool.PoolID)
	data.Description = types.StringValue(pool.Description)
	data.KeepAliveSeconds = types.Int64Value(int64(pool.KeepAliveSeconds))
	data.DisconnectSeconds = types.Int64Value(int64(pool.DisconnectSeconds))
	data.BrokerCount = types.Int64Value(int64(pool.BrokerCount))
	data.Brokers = brokerpool.BrokersToList(ctx, brokers, &resp.Diagnostics)
	data.Resources = brokerpool.LinkedResourcesToList(ctx, linked, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
