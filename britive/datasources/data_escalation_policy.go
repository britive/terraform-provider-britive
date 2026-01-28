package datasources

import (
	"context"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &DataSourceEscalationPolicy{}
	_ datasource.DataSourceWithConfigure = &DataSourceEscalationPolicy{}
)

type DataSourceEscalationPolicy struct {
	client *britive_client.Client
}

func NewDataSourceEscalationPolicy() datasource.DataSource {
	return &DataSourceEscalationPolicy{}
}

func (dep *DataSourceEscalationPolicy) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_escalation_policy"
}

func (dep *DataSourceEscalationPolicy) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive_client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	dep.client = client
	tflog.Info(ctx, "Configured DataSourceEscalationPolicy with Britive client")
}

func (dep *DataSourceEscalationPolicy) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive escalation policies.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of connection",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of escalation policy",
			},
			"im_connection_id": schema.StringAttribute{
				Required:    true,
				Description: "Id of IM connection",
			},
		},
	}
}

func (dep *DataSourceEscalationPolicy) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_escalation_policy datasource")

	var plan britive_client.DataSourceEscalationPolicyPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching escalation policy", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	policyName := plan.Name.ValueString()
	imConnectionId := plan.ImConnectionID.ValueString()
	more := true

	tflog.Info(ctx, fmt.Sprintf("list all '%s' escalation policies", policyName))

	var policyNames []string
	for i := 0; more == true; i++ {
		response, err := dep.client.GetEscalationPolicies(ctx, i, imConnectionId, policyName)
		if err != nil {
			resp.Diagnostics.AddError("Unable to get policy", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Unable to get policy, error: %#v", err))
			return
		}

		policies := response.Policies
		for _, policy := range policies {
			if policy["name"] == policyName {
				plan.Name = types.StringValue(policyName)
				plan.ID = types.StringValue(policy["id"])

				resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
				if resp.Diagnostics.HasError() {
					tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
						"diagnostics": resp.Diagnostics,
					})
					return
				}
				tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
					"escalationPolicies": plan,
				})
				return
			}
			policyNames = append(policyNames, policy["name"]+",")
		}
		more = response.More
	}

	lastPolicy := policyNames[len(policyNames)-1]
	policyNames[len(policyNames)-1] = lastPolicy[:len(lastPolicy)-1]

	errorMsg := fmt.Sprintf("%s, try with %s", policyName, policyNames)
	resp.Diagnostics.AddError("Unable to get policy", errorMsg)
	tflog.Error(ctx, fmt.Sprintf("Unable to get policy, error: %s", errorMsg))
}
