package datasources

import (
	"context"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &DataSourceIdentityProvider{}
	_ datasource.DataSourceWithConfigure = &DataSourceIdentityProvider{}
)

type DataSourceIdentityProvider struct {
	client *britive_client.Client
}

func NewDataSourceIdentityProvider() datasource.DataSource {
	return &DataSourceIdentityProvider{}
}

func (dip *DataSourceIdentityProvider) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_identity_provider"
}

func (dip *DataSourceIdentityProvider) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive_client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	dip.client = client
	tflog.Info(ctx, "Configured DataSourceIdentityProvider with Britive client")
}

func (dip *DataSourceIdentityProvider) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive identity provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of identity provider",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of identity provider",
				Validators: []validator.String{
					validate.StringFunc(
						"profileID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The type of the identity provider",
			},
		},
	}
}

func (dip *DataSourceIdentityProvider) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_identity_provider datasource")

	var plan britive_client.DataSourceIdentityProviderPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching identity providers", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	identityProviderName := plan.Name.ValueString()

	identityProvider, err := dip.client.GetIdentityProviderByName(ctx, identityProviderName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch identity provider", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch identity provider, err:%#v", err))
		return
	}

	plan.ID = types.StringValue(identityProvider.ID)
	plan.Name = types.StringValue(identityProvider.Name)
	plan.Type = types.StringValue(identityProvider.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
		"constraints": plan,
	})
}
