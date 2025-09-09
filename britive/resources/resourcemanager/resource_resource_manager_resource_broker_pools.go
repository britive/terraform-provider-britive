package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceBrokerPools - Terraform Resource for Broker Pools associated to a server access resource
type ResourceBrokerPools struct {
	Resource     *schema.Resource
	helper       *ResourceBrokerPoolsHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceBrokerPools - Initializes new broker pools to be associated to a resource
func NewResourceBrokerPools(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceBrokerPools {
	rbp := &ResourceBrokerPools{
		helper:       NewResourceBrokerPoolsHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rbp.Resource = &schema.Resource{
		CreateContext: rbp.resourceCreate,
		ReadContext:   rbp.resourceRead,
		DeleteContext: rbp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rbp.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The id of server access resource",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"broker_pools": {
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Description: "The broker pool names to be associated to the resource",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
	return rbp
}

//region Resource Broker Pools Context Operations

func (rbp *ResourceBrokerPools) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	serverAccessResourceID := d.Get("resource_id").(string)
	serverAccessResourceName, err := c.GetResourceName(serverAccessResourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	brokerPoolNames := d.Get("broker_pools").(*schema.Set)
	brokerPoolNamesString := make([]string, brokerPoolNames.Len())
	for i, b := range brokerPoolNames.List() {
		brokerPoolNamesString[i] = b.(string)
	}

	err = c.AddBrokerPoolsResource(brokerPoolNamesString, serverAccessResourceName)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new broker pools %#v for the resource: %#v", brokerPoolNamesString, serverAccessResourceID)
	d.SetId(rbp.helper.generateUniqueID(serverAccessResourceID))

	rbp.resourceRead(ctx, d, m)

	return diags
}

func (rbp *ResourceBrokerPools) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rbp.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rbp *ResourceBrokerPools) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	serverAccessResourceID, err := rbp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting the broker pools for resource: %s", serverAccessResourceID)

	err = c.DeleteBrokerPoolsResource(serverAccessResourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Broker pools for the resource %s are deleted", serverAccessResourceID)

	d.SetId("")

	return diags
}

func (rbp *ResourceBrokerPools) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := rbp.importHelper.ParseImportID([]string{"resources/(?P<name>[^/]+)/broker-pools", "(?P<name>[^/]+)/broker-pools"}, d); err != nil {
		return nil, err
	}
	serverAccessResourceName := d.Get("name").(string)
	if strings.TrimSpace(serverAccessResourceName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing broker pools for the resource : %s", serverAccessResourceName)

	d.SetId(rbp.helper.generateUniqueID(serverAccessResourceName))

	err := rbp.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceBrokerPoolsHelper - Resource Broker Pools helper functions
type ResourceBrokerPoolsHelper struct {
}

// NewResourceBrokerPoolsHelper - Initializes new broker pools resource helper
func NewResourceBrokerPoolsHelper() *ResourceBrokerPoolsHelper {
	return &ResourceBrokerPoolsHelper{}
}

//region Resource Broker Pools helper functions

func (rbph *ResourceBrokerPoolsHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	serverAccessResourceID, err := rbph.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading broker pools for resource %s", serverAccessResourceID)

	if err := d.Set("resource_id", serverAccessResourceID); err != nil {
		return err
	}

	serverAccessResourceName, err := c.GetResourceName(serverAccessResourceID)
	if err != nil {
		return err
	}

	brokerPoolNames, err := rbph.getBrokerPoolNames(serverAccessResourceName, m)
	if err != nil {
		return err
	}

	if err := d.Set("broker_pools", brokerPoolNames); err != nil {
		return err
	}

	return nil
}

func (resourceBrokerPoolsHelper *ResourceBrokerPoolsHelper) generateUniqueID(serverAccessResourceID string) string {
	return fmt.Sprintf("resources/%s/broker-pools", serverAccessResourceID)
}

func (resourceBrokerPoolsHelper *ResourceBrokerPoolsHelper) parseUniqueID(ID string) (serverAccessResourceID string, err error) {
	brokerPoolsResourceParts := strings.Split(ID, "/")
	if len(brokerPoolsResourceParts) < 3 {
		err = errs.NewInvalidResourceIDError("brokerPools", ID)
		return
	}

	serverAccessResourceID = brokerPoolsResourceParts[1]
	return
}

func (resourceBrokerPoolsHelper *ResourceBrokerPoolsHelper) getBrokerPoolNames(serverAccessResourceName string, m interface{}) (brokerPoolNames []string, err error) {
	c := m.(*britive.Client)
	brokerPools, err := c.GetBrokerPoolsResource(serverAccessResourceName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("broker pools for resource %s", serverAccessResourceName)
	}
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Received broker pools %#v for resource %#v", brokerPools, serverAccessResourceName)

	for _, brokerPool := range *brokerPools {
		brokerPoolNames = append(brokerPoolNames, brokerPool.Name)
	}
	return brokerPoolNames, nil
}

//endregion
