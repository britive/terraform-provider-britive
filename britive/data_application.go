package britive

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataApplication() *schema.Resource {
	return &schema.Resource{
		ReadContext: resourceApplicationReadByName,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceApplicationReadByName(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	applicationName := d.Get("name").(string)

	if applicationName == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Name must be passed to get application"),
		})
		return diags
	}

	applications, err := c.GetApplications()
	if err != nil {
		return diag.FromErr(err)
	}
	var application *britive.Application
	for _, app := range *applications {
		if strings.ToLower(app.CatalogAppDisplayName) == strings.ToLower(applicationName) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("App %v", app),
			})
			application = &app
			break
		}
	}

	if application == nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("No application found matching %s", applicationName),
		})
		return diags
	}
	d.SetId(application.AppContainerID)
	if err := d.Set("name", application.CatalogAppDisplayName); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
