package britive

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/datasources"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive/resources"
	"github.com/britive/terraform-provider-britive/britive/resources/resourcemanager"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
)

var version string

// Provider - Britive Provider
func Provider(v string) *schema.Provider {
	version = v
	validation := validate.NewValidation()
	importHelper := imports.NewImportHelper()

	resourceTag := resources.NewResourceTag(importHelper)
	resourceTagMember := resources.NewResourceTagMember(importHelper)
	resourceProfile := resources.NewResourceProfile(validation, importHelper)
	resourceProfilePermission := resources.NewResourceProfilePermission(importHelper)
	resourceProfileSessionAttribute := resources.NewResourceProfileSessionAttribute(importHelper)
	resourcePermission := resources.NewResourcePermission(validation, importHelper)
	resourceRole := resources.NewResourceRole(validation, importHelper)
	resourcePolicy := resources.NewResourcePolicy(importHelper)
	resourceProfilePolicy := resources.NewResourceProfilePolicy(importHelper)
	resourceConstraint := resources.NewResourceConstraint(importHelper)
	resourceProfileAdditionalSettings := resources.NewResourceProfileAdditionalSettings(importHelper)
	resourceApplication := resources.NewResourceApplication(validation, importHelper)
	resourceEntityGroup := resources.NewResourceEntityGroup(importHelper)
	resourceEntityEnvironment := resources.NewResourceEntityEnvironment(importHelper)
	resourceAdvancedSettings := resources.NewResourceAdvancedSettings(validation, importHelper)
	resourceResourceType := resourcemanager.NewResourceResourceType(validation, importHelper)
	resourceResourceTypePermissions := resourcemanager.NewResourceResourceTypePermissions(importHelper)
	resourceResponseTemplate := resourcemanager.NewResourceResponseTemplate(validation, importHelper)
	resourceResourceLabel := resourcemanager.NewResourceResourceLabel(validation, importHelper)
	resourceResourceManagerProfile := resourcemanager.NewResourceResourceManagerProfile(validation, importHelper)
	resourceResourceManagerProfilePolicy := resourcemanager.NewResourceResourceManagerProfilePolicy(validation, importHelper)
	resourceResourceManagerProfilePermission := resourcemanager.NewResourceResourceManagerProfilePermission(validation, importHelper)
	resourceServerAccess := resourcemanager.NewResourceServerAccess(validation, importHelper)
	resourceBrokerPools := resourcemanager.NewResourceBrokerPools(validation, importHelper)
	resourceResourceManagerResourcePolicy := resourcemanager.NewResourceResourcePolicy(validation, importHelper)
	resourceProfilePolicyPriority := resources.NewResourcePolicyPriority(validation, importHelper)

	dataSourceIdentityProvider := datasources.NewDataSourceIdentityProvider()
	dataSourceApplication := datasources.NewDataSourceApplication()
	dataSourceConstraints := datasources.NewDataSourceConstraints()
	dataSourceConnections := datasources.NewDataSourceConnection()
	dataSourceAllConnections := datasources.NewDataSourceAllConnections()
	dataSourceEscalationPolicy := datasources.NewDataSourceEscalationPolicy()
	dataSourceResourceManagerProfilePermissions := datasources.NewDataSourceResourceManagerProfilePermissions()

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"tenant": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_TENANT", nil),
				Description: "This is the Britive Tenant URL",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_TOKEN", nil),
				Description: "This is the API Token to interact with your Britive API",
			},
			"config_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_CONFIG", "~/.britive/tf.config"),
				Description: "This is the file path for Britive provider configuration. The default configuration path is ~/.britive/tf.config",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"britive_tag":                                       resourceTag.Resource,
			"britive_tag_member":                                resourceTagMember.Resource,
			"britive_profile":                                   resourceProfile.Resource,
			"britive_profile_permission":                        resourceProfilePermission.Resource,
			"britive_profile_session_attribute":                 resourceProfileSessionAttribute.Resource,
			"britive_permission":                                resourcePermission.Resource,
			"britive_role":                                      resourceRole.Resource,
			"britive_policy":                                    resourcePolicy.Resource,
			"britive_profile_policy":                            resourceProfilePolicy.Resource,
			"britive_constraint":                                resourceConstraint.Resource,
			"britive_profile_additional_settings":               resourceProfileAdditionalSettings.Resource,
			"britive_application":                               resourceApplication.Resource,
			"britive_entity_group":                              resourceEntityGroup.Resource,
			"britive_entity_environment":                        resourceEntityEnvironment.Resource,
			"britive_advanced_settings":                         resourceAdvancedSettings.Resource,
			"britive_resource_manager_resource_type":            resourceResourceType.Resource,
			"britive_resource_manager_resource_type_permission": resourceResourceTypePermissions.Resource,
			"britive_resource_manager_response_template":        resourceResponseTemplate.Resource,
			"britive_resource_manager_resource_label":           resourceResourceLabel.Resource,
			"britive_resource_manager_profile":                  resourceResourceManagerProfile.Resource,
			"britive_resource_manager_profile_policy":           resourceResourceManagerProfilePolicy.Resource,
			"britive_resource_manager_profile_permission":       resourceResourceManagerProfilePermission.Resource,
			"britive_resource_manager_resource":                 resourceServerAccess.Resource,
			"britive_resource_manager_resource_broker_pools":    resourceBrokerPools.Resource,
			"britive_resource_manager_resource_policy":          resourceResourceManagerResourcePolicy.Resource,
			"britive_profile_policy_prioritization":             resourceProfilePolicyPriority.Resource,
		},
		DataSourcesMap: map[string]*schema.Resource{
			"britive_identity_provider":                    dataSourceIdentityProvider.Resource,
			"britive_application":                          dataSourceApplication.Resource,
			"britive_supported_constraints":                dataSourceConstraints.Resource,
			"britive_connection":                           dataSourceConnections.Resource,
			"britive_all_connections":                      dataSourceAllConnections.Resource,
			"britive_escalation_policy":                    dataSourceEscalationPolicy.Resource,
			"britive_resource_manager_profile_permissions": dataSourceResourceManagerProfilePermissions.Resource,
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func getProviderConfigurationFromFile(d *schema.ResourceData) (string, string, error) {
	log.Print("[DEBUG] Trying to load configuration from file")
	if configPath, ok := d.GetOk("config_path"); ok && configPath.(string) != "" {
		path, err := homedir.Expand(configPath.(string))
		if err != nil {
			log.Printf("[DEBUG] Failed to expand config file path %s, error %s", configPath, err)
			return "", "", nil
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("[DEBUG] Terraform config file %s does not exist, error %s", path, err)
			return "", "", nil
		}
		log.Printf("[DEBUG] Terraform configuration file is: %s", path)
		configFile, err := os.Open(path)
		if err != nil {
			log.Printf("[DEBUG] Unable to open Terraform configuration file %s", path)
			return "", "", fmt.Errorf("unable to open terraform configuration file. error %v", err)
		}
		defer configFile.Close()

		configBytes, _ := ioutil.ReadAll(configFile)
		var config britive.Config
		err = json.Unmarshal(configBytes, &config)
		if err != nil {
			log.Printf("[DEBUG] Failed to parse config file %s", path)
			return "", "", fmt.Errorf("invalid terraform configuration file format. error %v", err)
		}
		return config.Tenant, config.Token, nil
	}
	return "", "", nil
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	var err error

	token := d.Get("token").(string)
	tenant := d.Get("tenant").(string)

	if tenant == "" && token == "" {
		tenant, token, err = getProviderConfigurationFromFile(d)
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}
	if tenant == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Initializing provider, tenant parameter is missing",
		})
	}
	if token == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Initializing provider, token parameter is missing",
		})
	}
	if diags != nil && len(diags) > 0 {
		return nil, diags
	}

	apiBaseURL := fmt.Sprintf("%s/api", strings.TrimSuffix(tenant, "/"))
	c, err := britive.NewClient(apiBaseURL, token, version)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create Britive client",
			Detail:   "Unable to authenticate user for Britive client",
		})

		return nil, diags
	}

	return c, diags
}
