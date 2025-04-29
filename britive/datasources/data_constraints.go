package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceConstraints - Terraform Constraints DataSource
type DataSourceConstraints struct {
	Resource *schema.Resource
	helper   *DataSourceConstraintsHelper
}

// DataSourceConstraintsHelper - DataSource Constraints helper functions
type DataSourceConstraintsHelper struct {
}

// NewDataSourceConstraintsHelper - Initializes new constraints resource helper
func NewDataSourceConstraintsHelper() *DataSourceConstraintsHelper {
	return &DataSourceConstraintsHelper{}
}

// NewDataSourceConstraints - Initializes new DataSourceConstraints
func NewDataSourceConstraints() *DataSourceConstraints {
	dataSourceConstraints := &DataSourceConstraints{
		helper: NewDataSourceConstraintsHelper(),
	}
	dataSourceConstraints.Resource = &schema.Resource{
		ReadContext: dataSourceConstraints.resourceRead,
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The identifier of the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permission_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the permission associated with the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permission_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "role",
				Description:  "The type of permission",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"constraint_types": {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "A set of constraints supported for the given profile permission",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
	return dataSourceConstraints
}

func (dataSourceConstraints *DataSourceConstraints) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	profileId := d.Get("profile_id").(string)
	permissionName := d.Get("permission_name").(string)
	permissionType := d.Get("permission_type").(string)
	supportedConstraintTypes, err := c.GetSupportedConstraintTypes(profileId, permissionName, permissionType)

	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("profileID %s, permission name %s, permission type %s", profileId, permissionName, permissionType))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(dataSourceConstraints.helper.generateUniqueID(profileId, permissionName, permissionType))

	d.Set("constraint_types", supportedConstraintTypes)

	return nil
}

func (dataSourceConstraintsHelper *DataSourceConstraintsHelper) generateUniqueID(profileID, permissionName, permissionType string) string {
	return fmt.Sprintf("paps/%s/permissions/%s/%s/supported-constraint-types", profileID, permissionName, permissionType)
}
