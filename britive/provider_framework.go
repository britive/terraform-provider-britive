package britive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/datasources"
	"github.com/britive/terraform-provider-britive/britive/resources"
	"github.com/britive/terraform-provider-britive/britive/resources/resourcemanager"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/go-homedir"
)

// Ensure the implementation satisfies the provider.Provider interface.
var _ provider.Provider = &BritiveProvider{}

// BritiveProvider defines the provider implementation.
type BritiveProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance testing.
	version string
}

// BritiveProviderModel describes the provider data model.
type BritiveProviderModel struct {
	Tenant     types.String `tfsdk:"tenant"`
	Token      types.String `tfsdk:"token"`
	ConfigPath types.String `tfsdk:"config_path"`
}

// Metadata returns the provider type name.
func (p *BritiveProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "britive"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *BritiveProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Britive.",
		Attributes: map[string]schema.Attribute{
			"tenant": schema.StringAttribute{
				Optional:    true,
				Description: "This is the Britive Tenant URL. Can also be set with the BRITIVE_TENANT environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "This is the API Token to interact with your Britive API. Can also be set with the BRITIVE_TOKEN environment variable.",
			},
			"config_path": schema.StringAttribute{
				Optional:    true,
				Description: "This is the file path for Britive provider configuration. The default configuration path is ~/.britive/tf.config. Can also be set with the BRITIVE_CONFIG environment variable.",
			},
		},
	}
}

// Configure prepares a Britive API client for data sources and resources.
func (p *BritiveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config BritiveProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from environment variables if not set in config
	tenant := config.Tenant.ValueString()
	token := config.Token.ValueString()
	configPath := config.ConfigPath.ValueString()

	if tenant == "" {
		tenant = os.Getenv("BRITIVE_TENANT")
	}

	if token == "" {
		token = os.Getenv("BRITIVE_TOKEN")
	}

	if configPath == "" {
		configPath = os.Getenv("BRITIVE_CONFIG")
		if configPath == "" {
			configPath = "~/.britive/tf.config"
		}
	}

	// Try to load from config file if tenant and token are still not set
	if tenant == "" && token == "" {
		fileTenant, fileToken, err := getProviderConfigurationFromConfigFile(configPath)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to load configuration from file",
				fmt.Sprintf("Error loading configuration from %s: %s", configPath, err.Error()),
			)
			return
		}
		if fileTenant != "" {
			tenant = fileTenant
		}
		if fileToken != "" {
			token = fileToken
		}
	}

	// Validate required configuration
	if tenant == "" {
		resp.Diagnostics.AddError(
			"Missing Britive Tenant",
			"The provider cannot create the Britive API client as there is a missing or empty value for the Britive tenant. "+
				"Set the tenant value in the configuration or use the BRITIVE_TENANT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Britive Token",
			"The provider cannot create the Britive API client as there is a missing or empty value for the Britive token. "+
				"Set the token value in the configuration or use the BRITIVE_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure tenant has https:// scheme
	if !strings.Contains(tenant, "://") {
		tenant = "https://" + tenant
	}

	// Create Britive API client
	apiBaseURL := fmt.Sprintf("%s/api", strings.TrimSuffix(tenant, "/"))
	client, err := britive.NewClient(apiBaseURL, token, p.version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Britive API Client",
			"An unexpected error occurred when creating the Britive API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Britive Client Error: "+err.Error(),
		)
		return
	}

	// Make the Britive client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *BritiveProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewIdentityProviderDataSource,
		datasources.NewApplicationDataSource,
		datasources.NewSupportedConstraintsDataSource,
		datasources.NewConnectionDataSource,
		datasources.NewAllConnectionsDataSource,
		datasources.NewEscalationPolicyDataSource,
		datasources.NewResourceManagerProfilePermissionsDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *BritiveProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewTagResource,
		resources.NewConstraintResource,
		resources.NewTagMemberResource,
		resources.NewEntityGroupResource,
		resources.NewEntityEnvironmentResource,
		resources.NewPolicyResource,
		resources.NewPermissionResource,
		resources.NewRoleResource,
		resources.NewProfilePermissionResource,
		resources.NewProfileSessionAttributeResource,
		resources.NewProfileAdditionalSettingsResource,
		resources.NewAdvancedSettingsResource,
		resources.NewProfilePolicyResource,
		resources.NewProfilePolicyPrioritizationResource,
		resources.NewApplicationResource,
		resources.NewProfileResource,
		// Resource Manager resources
		resourcemanager.NewResourceTypeResource,
		resourcemanager.NewResourceTypePermissionsResource,
		resourcemanager.NewResponseTemplateResource,
		resourcemanager.NewResourceLabelResource,
		resourcemanager.NewResourceResource,
		resourcemanager.NewProfileResource,
		resourcemanager.NewResourcePolicyResource,
		resourcemanager.NewProfilePermissionResource,
		resourcemanager.NewProfilePolicyResource,
		resourcemanager.NewResourceBrokerPoolsResource,
		resourcemanager.NewRMProfilePolicyPrioritizationResource,
	}
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BritiveProvider{
			version: version,
		}
	}
}

// getProviderConfigurationFromConfigFile loads configuration from the config file.
func getProviderConfigurationFromConfigFile(configPath string) (string, string, error) {
	if configPath == "" {
		return "", "", nil
	}

	path, err := homedir.Expand(configPath)
	if err != nil {
		return "", "", nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", "", nil
	}

	configFile, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("unable to open terraform configuration file: %v", err)
	}
	defer configFile.Close()

	configBytes, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("unable to read terraform configuration file: %v", err)
	}

	var config britive.Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return "", "", fmt.Errorf("invalid terraform configuration file format: %v", err)
	}

	return config.Tenant, config.Token, nil
}
