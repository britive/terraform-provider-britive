package datasources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ConnectionDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectionDataSource{}
)

func NewConnectionDataSource() datasource.DataSource {
	return &ConnectionDataSource{}
}

type ConnectionDataSource struct {
	client *britive.Client
}

type ConnectionDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	SettingType types.String `tfsdk:"setting_type"`
	Type        types.String `tfsdk:"type"`
	AuthType    types.String `tfsdk:"auth_type"`
}

func (d *ConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection"
}

func (d *ConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the connection.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of connection.",
			},
			"setting_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Advanced Setting Type. Defaults to 'ITSM'.",
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
	}
}

func (d *ConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectionDataSourceModel

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

	connectionName := data.Name.ValueString()
	isConnectionFound := false
	allConnectionNames := make([]string, 0)

	for _, conn := range allConnections {
		if strings.EqualFold(conn.Name, connectionName) {
			data.ID = types.StringValue(conn.ID)
			data.Name = types.StringValue(connectionName)
			data.Type = types.StringValue(conn.Type)
			data.AuthType = types.StringValue(conn.AuthType)
			isConnectionFound = true
			break
		}
		allConnectionNames = append(allConnectionNames, conn.Name)
	}

	if !isConnectionFound {
		if len(allConnectionNames) > 0 {
			resp.Diagnostics.AddError(
				"Connection Not Found",
				fmt.Sprintf("Connection '%s' not found. Available connections: %v", connectionName, allConnectionNames),
			)
		} else {
			resp.Diagnostics.AddError(
				"Connection Not Found",
				fmt.Sprintf("Connection '%s' not found.", connectionName),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
