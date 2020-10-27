package britive

import (
	"context"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//Provider - godoc
func Provider() *schema.Provider {
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
		},
		ResourcesMap: map[string]*schema.Resource{
			"britive_tag":                resourceTag(),
			"britive_tag_member":         resourceTagMember(),
			"britive_profile":            resourceProfile(),
			"britive_profile_permission": resourceProfilePermission(),
			"britive_profile_tag":        resourceProfileTag(),
			"britive_profile_identity":   resourceProfileIdentity(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"britive_identity_provider": dataIdentityProvider(),
			"britive_application":       dataApplication(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	token := d.Get("token").(string)
	var host *string

	hVal, ok := d.GetOk("host")
	if ok {
		tempHost := hVal.(string)
		host = &tempHost
	}

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if token != "" {
		c, err := britive.NewClient(host, &token)
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

	c, err := britive.NewClient(host, nil)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create Britive client",
			Detail:   "Unable to create anonymous Britive client",
		})
		return nil, diags
	}

	return c, diags
}
