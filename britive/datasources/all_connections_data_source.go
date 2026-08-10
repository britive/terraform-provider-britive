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
	_ datasource.DataSource              = &AllConnectionsDataSource{}
	_ datasource.DataSourceWithConfigure = &AllConnectionsDataSource{}
)

func NewAllConnectionsDataSource() datasource.DataSource {
	return &AllConnectionsDataSource{}
}

type AllConnectionsDataSource struct {
	client *britive.Client
}

type AllConnectionsDataSourceModel struct {
	ID          types.String      `tfsdk:"id"`
	SettingType types.String      `tfsdk:"setting_type"`
	Connections []ConnectionModel `tfsdk:"connections"`
}

type ConnectionModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	AuthType types.String `tfsdk:"auth_type"`
}

func (d *AllConnectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_all_connections"
}

func (d *AllConnectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Britive connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The identifier for this data source.",
			},
			"setting_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Advanced Setting Type. Defaults to 'ITSM'.",
			},
			"connections": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of all connections.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Id of connection.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of connection.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of connection.",
						},
						"auth_type": schema.StringAttribute{
							Computed:    true,
							Description: "Auth type of connection.",
						},
					},
				},
			},
		},
	}
}

func (d *AllConnectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AllConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AllConnectionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settingType := data.SettingType.ValueString()
	if settingType == "" {
		settingType = "ITSM"
		data.SettingType = types.StringValue("ITSM")
	}

	allConnections, err := d.client.GetAllConnections(settingType)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Connections Not Found",
			"No connections found.",
		)
		return
	} else if errors.Is(err, britive.ErrNotSupported) {
		resp.Diagnostics.AddError(
			"Setting Type Not Supported",
			fmt.Sprintf("Setting type '%s' is not supported.", settingType),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Connections",
			fmt.Sprintf("Could not read connections: %s", err.Error()),
		)
		return
	}

	connections := make([]ConnectionModel, 0, len(allConnections))
	for _, conn := range allConnections {
		connections = append(connections, ConnectionModel{
			ID:       types.StringValue(conn.ID),
			Name:     types.StringValue(conn.Name),
			Type:     types.StringValue(conn.Type),
			AuthType: types.StringValue(conn.AuthType),
		})
	}

	data.ID = types.StringValue("all-connections")
	data.Connections = connections

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
