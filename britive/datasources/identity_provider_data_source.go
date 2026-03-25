package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &IdentityProviderDataSource{}
	_ datasource.DataSourceWithConfigure = &IdentityProviderDataSource{}
)

// NewIdentityProviderDataSource is a helper function to simplify the provider implementation.
func NewIdentityProviderDataSource() datasource.DataSource {
	return &IdentityProviderDataSource{}
}

// IdentityProviderDataSource is the data source implementation.
type IdentityProviderDataSource struct {
	client *britive.Client
}

// IdentityProviderDataSourceModel describes the data source data model.
type IdentityProviderDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

// Metadata returns the data source type name.
func (d *IdentityProviderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider"
}

// Schema defines the schema for the data source.
func (d *IdentityProviderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive identity provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the identity provider.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the identity provider.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the identity provider.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *IdentityProviderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data.
func (d *IdentityProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityProviderDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get identity provider from API
	identityProviderName := data.Name.ValueString()
	identityProvider, err := d.client.GetIdentityProviderByName(identityProviderName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Identity Provider Not Found",
			fmt.Sprintf("Identity provider '%s' was not found.", identityProviderName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Identity Provider",
			fmt.Sprintf("Could not read identity provider '%s': %s", identityProviderName, err.Error()),
		)
		return
	}

	// Map response body to model
	data.ID = types.StringValue(identityProvider.ID)
	data.Name = types.StringValue(identityProvider.Name)
	data.Type = types.StringValue(identityProvider.Type)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
