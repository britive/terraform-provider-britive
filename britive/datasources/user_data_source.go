package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &UserDataSource{}
	_ datasource.DataSourceWithConfigure = &UserDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

type UserDataSource struct {
	client *britive.Client
}

type UserDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	UserID types.String `tfsdk:"user_id"`
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the user.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The username of the user.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("user_id"),
					),
				},
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the user.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("user_id"),
					),
				},
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *britive.User
	var err error

	if !data.UserID.IsNull() && !data.UserID.IsUnknown() && data.UserID.ValueString() != "" {
		user, err = d.client.GetUser(data.UserID.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("User Not Found",
				fmt.Sprintf("User with id '%s' was not found.", data.UserID.ValueString()))
			return
		}
	} else {
		user, err = d.client.GetUserByName(data.Name.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("User Not Found",
				fmt.Sprintf("User '%s' was not found.", data.Name.ValueString()))
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User", err.Error())
		return
	}

	data.ID = types.StringValue(user.UserID)
	data.Name = types.StringValue(user.Username)
	data.UserID = types.StringValue(user.UserID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
