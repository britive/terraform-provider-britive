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

// DataSourceUser - Terraform User DataSource
type DataSourceUser struct {
	Resource *schema.Resource
}

// NewDataSourceUser - Initializes new DataSourceUser
func NewDataSourceUser() *DataSourceUser {
	dataSourceUser := &DataSourceUser{}
	dataSourceUser.Resource = &schema.Resource{
		ReadContext: dataSourceUser.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The username of the user",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"name", "user_id"},
			},
			"user_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The unique identifier of the user",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"name", "user_id"},
			},
		},
	}
	return dataSourceUser
}

func (dataSourceUser *DataSourceUser) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var user *britive.User
	var err error

	if userID, ok := d.GetOk("user_id"); ok {
		user, err = c.GetUser(userID.(string))
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("user with id %s", userID.(string)))
		}
	} else {
		username := d.Get("name").(string)
		user, err = c.GetUserByName(username)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("user %s", username))
		}
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(user.UserID)

	if err := d.Set("name", user.Username); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("user_id", user.UserID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
