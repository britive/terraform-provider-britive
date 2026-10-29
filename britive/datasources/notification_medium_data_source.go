package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &NotificationMediumDataSource{}
	_ datasource.DataSourceWithConfigure = &NotificationMediumDataSource{}
)

func NewNotificationMediumDataSource() datasource.DataSource {
	return &NotificationMediumDataSource{}
}

type NotificationMediumDataSource struct {
	client *britive.Client
}

type NotificationMediumDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

func (d *NotificationMediumDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_medium"
}

func (d *NotificationMediumDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive notification medium.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the notification medium.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the notification medium, for example Email.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the notification medium, for example email.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the notification medium.",
			},
		},
	}
}

func (d *NotificationMediumDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *NotificationMediumDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NotificationMediumDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	medium, err := d.client.GetNotificationMediumByName(data.Name.ValueString())
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Notification Medium Not Found",
			fmt.Sprintf("Notification medium '%s' was not found.", data.Name.ValueString()))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Notification Medium", err.Error())
		return
	}

	data.ID = types.StringValue(medium.ID)
	data.Name = types.StringValue(medium.Name)
	data.Type = types.StringValue(medium.Type)
	data.Description = types.StringValue(medium.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
