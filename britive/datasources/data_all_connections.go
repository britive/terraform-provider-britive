package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &DataSourceAllConnections{}
	_ datasource.DataSourceWithConfigure = &DataSourceAllConnections{}
)

type DataSourceAllConnections struct {
	client *britive_client.Client
}

func NewDataSourceAllConnections() datasource.DataSource {
	return &DataSourceAllConnections{}
}

func (dac *DataSourceAllConnections) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_all_connections"
}

func (dac *DataSourceAllConnections) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

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

	dac.client = client
	tflog.Info(ctx, "Configured DataSourceAllConnections with Britive client")
}

func (dac *DataSourceAllConnections) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive all connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of all connections",
			},
			"setting_type": schema.StringAttribute{
				Optional:    true,
				Description: "Advanced Setting Type",
				Validators: []validator.String{
					validate.StringFunc(
						"settingType",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"connections": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All Connections",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of connection",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of connection",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of connection",
						},
						"auth_type": schema.StringAttribute{
							Computed:    true,
							Description: "Auth type of connection",
						},
					},
				},
			},
		},
	}
}

func (dac *DataSourceAllConnections) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_all_connections datasource")

	var plan britive_client.DataSourceAllConnectionsPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching all connections", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	var settingType string
	if plan.SettingType.IsNull() || plan.SettingType.IsUnknown() {
		settingType = "ITSM"
	} else {
		settingType = plan.SettingType.ValueString()
	}

	allConnections, err := dac.client.GetAllConnections(ctx, settingType)
	if errors.Is(err, britive_client.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to fetch all connections", "connections not found")
		tflog.Error(ctx, "connections not found")
		return
	} else if errors.Is(err, britive_client.ErrNotSupported) {
		resp.Diagnostics.AddError("Failed to fetch all connections", fmt.Sprintf("setting type is '%s'", settingType))
		tflog.Error(ctx, fmt.Sprintf("%s setting type is ", settingType))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch all connections", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch all connections, error: %#v", err))
		return
	}

	var results []britive_client.DataSourceConnectionPlan
	for _, conn := range allConnections {
		results = append(results, britive_client.DataSourceConnectionPlan{
			ID:       types.StringValue(conn.ID),
			Name:     types.StringValue(conn.Name),
			Type:     types.StringValue(conn.Type),
			AuthType: types.StringValue(conn.AuthType),
		})
	}

	plan.SettingType = types.StringValue(settingType)
	plan.Connections = results
	plan.ID = types.StringValue("all-connectios")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
		"connections": plan,
	})
}
