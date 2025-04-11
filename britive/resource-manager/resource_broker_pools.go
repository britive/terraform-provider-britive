package resourcemanager

//package britive

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceBrokerPools - Terraform Resource for Broker Pools associated to a server access resource
type ResourceBrokerPools struct {
	Resource     *schema.Resource
	helper       *ResourceBrokerPoolsHelper
	validation   *britive.Validation
	importHelper *britive.ImportHelper
}

// NewResourceBrokerPools - Initializes new broker pools to be associated to a resource
func NewResourceBrokerPools(v *Validation, importHelper *ImportHelper) *ResourceBrokerPools {
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
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Description: "The broker pool ids to be associated to the resource",
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
	brokerPoolIds := d.Get("broker_pools").([]interface{})
	brokerPoolIdsString := make([]string, len(brokerPoolIds))
	for i, b := range brokerPoolIds {
		brokerPoolIdsString[i] = b.(string)
	}

	err := c.AddBrokerPoolsResource(brokerPoolIdsString, serverAccessResourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new broker pools %#v for the resource: %#v", brokerPoolIdsString, serverAccessResourceID)
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

	brokerPoolIds, err := rbp.helper.getBrokerPoolIds(serverAccessResourceID, m)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, brokerPoolId := range brokerPoolIds {
		log.Printf("[INFO] Deleting broker pool %s for resource: %s", brokerPoolId, serverAccessResourceID)
		err = c.DeleteBrokerPoolsResource(serverAccessResourceID, brokerPoolId)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Broker pool %s for resource %s is deleted", brokerPoolId, serverAccessResourceID)
	}
	d.SetId("")

	return diags
}

func (rbp *ResourceBrokerPools) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := rbp.importHelper.ParseImportID([]string{"resources/(?P<name>[^/]+)/broker-pools", "(?P<name>[^/]+)/broker-pools"}, d); err != nil {
		return nil, err
	}
	serverAccessResourceName := d.Get("name").(string)
	if strings.TrimSpace(serverAccessResourceName) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("name")
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

	serverAccessResourceID, err := rbph.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading broker pools for resource %s", serverAccessResourceID)

	if err := d.Set("resource_id", serverAccessResourceID); err != nil {
		return err
	}

	brokerPoolIds, err := rbph.getBrokerPoolIds(serverAccessResourceID, m)
	if err != nil {
		return err
	}

	if err := d.Set("broker_pools", brokerPoolIds); err != nil {
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
		err = NewInvalidResourceIDError("brokerPools", ID)
		return
	}

	serverAccessResourceID = brokerPoolsResourceParts[1]
	return
}

func (resourceBrokerPoolsHelper *ResourceBrokerPoolsHelper) getBrokerPoolIds(serverAccessResourceID string, m interface{}) (brokerPoolIds []string, err error) {
	c := m.(*britive.Client)
	brokerPools, err := c.GetBrokerPoolsResource(serverAccessResourceID)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("broker pools for resource %s", serverAccessResourceID)
	}
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Received broker pools %#v for resource %#v", brokerPools, serverAccessResourceID)

	for _, brokerPool := range *brokerPools {
		brokerPoolIds = append(brokerPoolIds, brokerPool.BrokerPoolID)
	}
	return brokerPoolIds, nil
}

//endregion
