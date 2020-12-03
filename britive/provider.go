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
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
)

//Provider - Britive Provider
func Provider() *schema.Provider {
	validation := NewValidation()
	importHelper := NewImportHelper()

	resourceTag := NewResourceTag(importHelper)
	resourceTagMember := NewResourceTagMember(importHelper)
	resourceProfile := NewResourceProfile(validation, importHelper)
	resourceProfilePermission := NewResourceProfilePermission(importHelper)
	resourceProfileIdentity := NewResourceProfileIdentity(importHelper)
	resourceProfileTag := NewResourceProfileTag(importHelper)

	dataSourceIdentityProvider := NewDataSourceIdentityProvider()
	dataSourceApplication := NewDataSourceApplication()

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"tenant": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_TENANT", nil),
			},
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_TOKEN", nil),
			},
			"config_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_CONFIG", "~/.britive/config"),
				Description: "Path to the britive config file, defaults to ~/.britive/config",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"britive_tag":                resourceTag.Resource,
			"britive_tag_member":         resourceTagMember.Resource,
			"britive_profile":            resourceProfile.Resource,
			"britive_profile_permission": resourceProfilePermission.Resource,
			"britive_profile_identity":   resourceProfileIdentity.Resource,
			"britive_profile_tag":        resourceProfileTag.Resource,
		},
		DataSourcesMap: map[string]*schema.Resource{
			"britive_identity_provider": dataSourceIdentityProvider.Resource,
			"britive_application":       dataSourceApplication.Resource,
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	token := d.Get("token").(string)
	tenant := d.Get("tenant").(string)

	var diags diag.Diagnostics

	if tenant == "" && token == "" {
		log.Print("[DEBUG] Trying to load configuration from file")
		if configPath, ok := d.GetOk("config_path"); ok && configPath.(string) != "" {
			path, err := homedir.Expand(configPath.(string))
			if err != nil {
				log.Printf("[DEBUG] Failed to expand config file path %s", configPath)
				return nil, diag.FromErr(err)
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				log.Printf("[DEBUG] Config file %s not exists", path)
				return nil, diag.FromErr(err)
			}
			log.Printf("[DEBUG] Configuration file is: %s", path)
			configFile, err := os.Open(path)
			if err != nil {
				log.Printf("[DEBUG] Unable to open config file %s", path)
				return nil, diag.FromErr(err)
			}
			defer configFile.Close()

			configBytes, _ := ioutil.ReadAll(configFile)
			var config britive.Config
			err = json.Unmarshal(configBytes, &config)
			if err != nil {
				log.Printf("[DEBUG] Failed to parse config file %s", path)
				return nil, diag.FromErr(err)
			}
			tenant = config.Tenant
			token = config.Token
		}
	}
	if tenant == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Initialising provider, tenant parameter is missing",
		})
	}
	if token == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Initialising provider, token parameter is missing",
		})
	}
	if diags != nil && len(diags) > 0 {
		return nil, diags
	}

	apiBaseURL := fmt.Sprintf("%s/api", strings.TrimSuffix(tenant, "/"))
	c, err := britive.NewClient(apiBaseURL, token)
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
