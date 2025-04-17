package datasources

import (
	"context"
	"errors"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceIdentityProvider - Terraform IdentityProvider DataSource
type DataSourceIdentityProvider struct {
	Resource *schema.Resource
}

// NewDataSourceIdentityProvider - Initializes new DataSourceIdentityProvider
func NewDataSourceIdentityProvider() *DataSourceIdentityProvider {
	dataSourceIdentityProvider := &DataSourceIdentityProvider{}
	dataSourceIdentityProvider.Resource = &schema.Resource{
		ReadContext: dataSourceIdentityProvider.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the identity provider",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The type of the identity provider",
			},
		},
	}
	return dataSourceIdentityProvider
}

func (dataSourceIdentityProvider *DataSourceIdentityProvider) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	identityProviderName := d.Get("name").(string)

	identityProvider, err := c.GetIdentityProviderByName(identityProviderName)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("identity provider %s", identityProviderName))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(identityProvider.ID)

	if err := d.Set("name", identityProvider.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("type", identityProvider.Type); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
