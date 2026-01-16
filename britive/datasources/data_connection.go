package datasources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &DataSourceConnection{}
	_ datasource.DataSourceWithConfigure = &DataSourceConnection{}
)

type DataSourceConnection struct {
	client *britive_client.Client
}

func NewDataSourceConnection() datasource.DataSource {
	return &DataSourceConnection{}
}

func (dc *DataSourceConnection) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_connection"
}

func (dc *DataSourceConnection) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

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

	dc.client = client
	tflog.Info(ctx, "Configured DataSourceConnections with Britive client")
}

func (dc *DataSourceConnection) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of connection",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of connection",
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
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of connection",
			},
			"auth_type": schema.StringAttribute{
				Computed:    true,
				Description: "auth type of connection",
			},
		},
	}
}

func (dc *DataSourceConnection) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_connection datasource")

	var plan britive_client.DataSourceSingleConnectionPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching connection", map[string]interface{}{
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

	allConnections, err := dc.client.GetAllConnections(ctx, settingType)
	if errors.Is(err, britive_client.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to fetch connection", "connection not found")
		tflog.Error(ctx, "connections not found")
		return
	} else if errors.Is(err, britive_client.ErrNotSupported) {
		resp.Diagnostics.AddError("Failed to fetch connection", fmt.Sprintf("setting type is '%s'", settingType))
		tflog.Error(ctx, fmt.Sprintf("%s setting type is ", settingType))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch connection", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch all connections, error: %#v", err))
		return
	}

	connectionName := plan.Name.ValueString()

	isConnectionFound := false
	allConnectionNames := make([]string, 0)
	for _, conn := range allConnections {
		if strings.EqualFold(conn.Name, connectionName) {
			plan.ID = types.StringValue(conn.ID)
			plan.Name = types.StringValue(conn.Name)
			plan.Type = types.StringValue(conn.Type)
			plan.AuthType = types.StringValue(conn.AuthType)
			plan.SettingType = types.StringValue(settingType)
			isConnectionFound = true
		}
		allConnectionNames = append(allConnectionNames, conn.Name+",")
	}

	if !isConnectionFound {
		totalConnections := len(allConnectionNames) - 1
		var errMsg error
		if totalConnections >= 0 {
			allConnectionNames[totalConnections] = allConnectionNames[totalConnections][:len(allConnectionNames[totalConnections])-1]
			errMsg = fmt.Errorf("Invalid connection name.\nTry with %v", allConnectionNames)
		} else {
			errMsg = fmt.Errorf("Invalid connection name.")
		}
		resp.Diagnostics.AddError("Failed to fetch connection", errMsg.Error())
		tflog.Error(ctx, errMsg.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
		"connection": plan,
	})
}
