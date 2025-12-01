package britive

import (
	"context"
	"fmt"
	"net/url"

	"github.com/britive/terraform-provider-britive/britive/resources"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type britiveProvider struct {
	version string
	baseURL string
	token   string
	client  *britive_client.Client // Store API client here
}

var (
	_   provider.Provider = &britiveProvider{}
	tag string
)

func New() provider.Provider {
	const defaultVersion = "1.0"

	return &britiveProvider{
		version: func() string {
			if len(tag) == 0 {
				return defaultVersion
			}
			return tag
		}(),
	}
}

func (p *britiveProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "britive"
	resp.Version = p.version
}

func (p *britiveProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Britive Provider enables Terraform to interact with the Britive REST API.",
		Attributes: map[string]schema.Attribute{
			"tenant": schema.StringAttribute{
				Description: "The tenant URL or domain of your Britive instance.",
				Required:    true,
			},
			"token": schema.StringAttribute{
				Description: "The authentication token used to connect to the Britive API.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *britiveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring the Britive provider")

	var config britive_client.BritiveProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate tenant URL
	if config.Tenant.IsNull() || config.Tenant.ValueString() == britive_client.EmptyString {
		resp.Diagnostics.AddAttributeError(
			path.Root("tenant"),
			"Missing Tenant",
			"You must specify the tenant URL for your Britive instance.",
		)
		return
	}
	currentValue := config.Tenant.ValueString()
	u, err := url.Parse(currentValue)
	if err != nil || u.Scheme == "" || u.Host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("tenant"),
			"Invalid Tenant URL",
			fmt.Sprintf("The provided tenant URL %q is not valid. A full URL with scheme is required, e.g. https://example.britive.com.", currentValue),
		)
		return
	}

	p.baseURL = fmt.Sprintf("%s/api", currentValue)

	// Set token
	if config.Token.IsNull() || config.Token.ValueString() == britive_client.EmptyString {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing API Token",
			"You must provide an authentication token to connect to the Britive API.",
		)
		return
	}
	p.token = config.Token.ValueString()

	// Create API client
	client, err := britive_client.NewClient(p.baseURL, p.token, p.version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Britive client",
			"Failed to authenticate or initialize the Britive API client: "+err.Error(),
		)
		return
	}
	p.client = client

	// Pass provider configuration to resources and data sources
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *britiveProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Add datasources here
	}
}

func (p *britiveProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Add resources here
		resources.NewResourceApplication,
		resources.NewResourceProfile,
	}
}
