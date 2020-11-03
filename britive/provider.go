package britive

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
)

//Provider - godoc
func Provider() *schema.Provider {
	importHelper := NewImportHelper()

	resourceTag := NewResourceTag(importHelper)
	resourceTagMember := NewResourceTagMember(importHelper)

	dataSourceIdentityProvider := NewDataSourceIdentityProvider()

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("BRITIVE_HOST", nil),
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
			"britive_tag":        resourceTag.Resource,
			"britive_tag_member": resourceTagMember.Resource,
		},
		DataSourcesMap: map[string]*schema.Resource{
			"britive_identity_provider": dataSourceIdentityProvider.Resource,
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	token := d.Get("token").(string)
	host := d.Get("host").(string)

	var diags diag.Diagnostics

	if host == "" && token == "" {
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
			host = config.Host
			token = config.Token
		}
	}
	if host == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Initialising provider, host parameter is missing",
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

	c, err := britive.NewClient(&host, &token)
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
