package britive

import (
	"context"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataIdentityProvider() *schema.Resource {
	return &schema.Resource{
		ReadContext: resourceIdentityProviderReadByName,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceIdentityProviderReadByName(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	identityProviderName := d.Get("name").(string)

	if identityProviderName == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Name must be passed to get identity provider"),
		})
		return diags
	}

	identityProvider, err := c.GetIdentityProviderByName(identityProviderName)
	if err != nil {
		return diag.FromErr(err)
	}

	if identityProvider == nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("No identity provider found matching %s", identityProviderName),
		})
		return diags
	}
	d.SetId(identityProvider.ID)
	if err := d.Set("name", identityProvider.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", identityProvider.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("type", identityProvider.Type); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
