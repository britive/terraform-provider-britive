package britive

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DataSourceEscalationPolicy struct {
	Resource *schema.Resource
}

func NewDataSourceEscalationPolicy() *DataSourceEscalationPolicy {
	dsep := &DataSourceEscalationPolicy{}
	dsep.Resource = &schema.Resource{
		ReadContext: dsep.resourceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of policy",
			},
			"im_connection_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Id of IM connection",
			},
		},
	}
	return dsep
}

func (dsep *DataSourceEscalationPolicy) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	policyName := d.Get("name").(string)
	imConnectionId := d.Get("im_connection_id").(string)
	more := true

	log.Printf("[INFO] list all '%s' escalation policies", policyName)

	var policyNames []string
	for page := 0; more == true; page++ {
		response, err := c.GetEscalationPolicies(page, imConnectionId, policyName)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(NewNotFoundErrorf(policyName))
		} else if err != nil {
			return diag.FromErr(err)
		}

		policies := response.Policies
		for _, policy := range policies {
			if policy["name"] == policyName {
				d.Set("name", policyName)
				d.SetId(policy["id"])
				return nil
			}
			policyNames = append(policyNames, policy["name"]+",")
		}
		more = response.More
	}

	lastPolicy := policyNames[len(policyNames)-1]
	policyNames[len(policyNames)-1] = lastPolicy[:len(lastPolicy)-1]

	errorMsg := fmt.Sprintf("%s, try with %s", policyName, policyNames)
	return diag.FromErr(NewNotFoundErrorf(errorMsg))
}
