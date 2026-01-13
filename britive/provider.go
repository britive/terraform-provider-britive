package britive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/britive/terraform-provider-britive/britive/datasources"
	"github.com/britive/terraform-provider-britive/britive/resources"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/go-homedir"
)

// FileConfig represents the structure of the config file
type FileConfig struct {
	Tenant string `json:"tenant"`
	Token  string `json:"token"`
}

type britiveProvider struct {
	version string
	baseURL string
	token   string
	client  *britive_client.Client
}

var _ provider.Provider = &britiveProvider{}
var tag string

func New() provider.Provider {
	const defaultVersion = "1.0"
	return &britiveProvider{
		version: func() string {
			if tag == "" {
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
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must not be empty or whitespace"),
				},
			},
			"token": schema.StringAttribute{
				Description: "The authentication token used to connect to the Britive API.",
				Optional:    true,
				Sensitive:   true,
			},
			"config_path": schema.StringAttribute{
				Description: "The file path for Britive provider configuration. Default is ~/.britive/tf.config",
				Optional:    true,
			},
		},
	}
}

// loadConfigFromFile reads the config file and returns tenant and token
func loadConfigFromFile(configPath string) (string, string, error) {
	if configPath == "" {
		return "", "", nil
	}

	expandedPath, err := homedir.Expand(configPath)
	if err != nil {
		log.Printf("[DEBUG] Failed to expand config file path %s: %v", configPath, err)
		return "", "", err
	}

	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		log.Printf("[DEBUG] Config file does not exist at path: %s", expandedPath)
		return "", "", nil
	}

	configBytes, err := os.ReadFile(expandedPath)
	if err != nil {
		log.Printf("[DEBUG] Unable to read config file %s: %v", expandedPath, err)
		return "", "", err
	}

	var fileConfig FileConfig
	if err := json.Unmarshal(configBytes, &fileConfig); err != nil {
		log.Printf("[DEBUG] Failed to parse config file %s: %v", expandedPath, err)
		return "", "", err
	}

	return fileConfig.Tenant, fileConfig.Token, nil
}

func (p *britiveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring the Britive provider")

	var config britive_client.BritiveProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.ConfigPath.IsNull() && config.ConfigPath.ValueString() != "" {
		fileTenant, fileToken, err := loadConfigFromFile(config.ConfigPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to load configuration file",
				fmt.Sprintf("Failed to load configuration file %q: %v", config.ConfigPath.ValueString(), err),
			)
			return
		}

		if config.Tenant.IsNull() && fileTenant != "" {
			config.Tenant = types.StringValue(fileTenant)
		}
		if config.Token.IsNull() && fileToken != "" {
			config.Token = types.StringValue(fileToken)
		}
	}

	if config.Tenant.IsNull() || config.Tenant.ValueString() == "" {
		config.Tenant = types.StringValue(os.Getenv("BRITIVE_TENANT"))
	}
	if config.Token.IsNull() || config.Token.ValueString() == "" {
		config.Token = types.StringValue(os.Getenv("BRITIVE_TOKEN"))
	}

	if config.Tenant.IsNull() || config.Tenant.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("tenant"),
			"Missing Tenant",
			"You must specify the tenant URL for your Britive instance.",
		)
		return
	}

	u, err := url.Parse(config.Tenant.ValueString())
	if err != nil || u.Scheme == "" || u.Host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("tenant"),
			"Invalid Tenant URL",
			fmt.Sprintf("The provided tenant URL %q is not valid.", config.Tenant.ValueString()),
		)
		return
	}
	p.baseURL = fmt.Sprintf("%s/api", config.Tenant.ValueString())

	if config.Token.IsNull() || config.Token.ValueString() == "" {
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
		datasources.NewDataSourceApplication,
	}
}

func (p *britiveProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewResourceConstraint,
		resources.NewResourceApplication,
		resources.NewResourceProfile,
		resources.NewResourceAdvancedSettings,
		resources.NewResourceProfilePermission,
		resources.NewResourceProfileAdditionalSettings,
		resources.NewResourceEntityGroup,
		resources.NewResourceEntityEnvironment,
	}
}
