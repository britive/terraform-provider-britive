package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &EscalationPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &EscalationPolicyDataSource{}
)

func NewEscalationPolicyDataSource() datasource.DataSource {
	return &EscalationPolicyDataSource{}
}

type EscalationPolicyDataSource struct {
	client *britive.Client
}

type EscalationPolicyDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	IMConnectionID types.String `tfsdk:"im_connection_id"`
}

func (d *EscalationPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation_policy"
}

func (d *EscalationPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive escalation policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the escalation policy.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of policy.",
			},
			"im_connection_id": schema.StringAttribute{
				Required:    true,
				Description: "Id of IM connection.",
			},
		},
	}
}

func (d *EscalationPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *EscalationPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EscalationPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyName := data.Name.ValueString()
	imConnectionID := data.IMConnectionID.ValueString()

	// Paginate through all policies
	more := true
	var policyNames []string

	for i := 0; more; i++ {
		response, err := d.client.GetEscalationPolicies(i, imConnectionID, policyName)
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Escalation Policy Not Found",
				fmt.Sprintf("Escalation policy '%s' not found.", policyName),
			)
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Escalation Policies",
				fmt.Sprintf("Could not read escalation policies: %s", err.Error()),
			)
			return
		}

		policies := response.Policies
		for _, policy := range policies {
			if policy["name"] == policyName {
				data.ID = types.StringValue(policy["id"])
				data.Name = types.StringValue(policyName)

				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
			policyNames = append(policyNames, policy["name"])
		}
		more = response.More
	}

	// Policy not found, provide helpful error with available policies
	if len(policyNames) > 0 {
		resp.Diagnostics.AddError(
			"Escalation Policy Not Found",
			fmt.Sprintf("Escalation policy '%s' not found. Available policies: %v", policyName, policyNames),
		)
	} else {
		resp.Diagnostics.AddError(
			"Escalation Policy Not Found",
			fmt.Sprintf("Escalation policy '%s' not found.", policyName),
		)
	}
}
