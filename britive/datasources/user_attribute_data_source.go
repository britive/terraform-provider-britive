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
	_ datasource.DataSource              = &UserAttributeDataSource{}
	_ datasource.DataSourceWithConfigure = &UserAttributeDataSource{}
)

func NewUserAttributeDataSource() datasource.DataSource {
	return &UserAttributeDataSource{}
}

type UserAttributeDataSource struct {
	client *britive.Client
}

type UserAttributeDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	AttributeSchemaID types.String `tfsdk:"attribute_schema_id"`
}

func (d *UserAttributeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_attribute"
}

func (d *UserAttributeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive user attribute by name or attribute schema ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the user attribute (same as attribute_schema_id).",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the user attribute.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("attribute_schema_id"),
					),
				},
			},
			"attribute_schema_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the user attribute.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("attribute_schema_id"),
					),
				},
			},
		},
	}
}

func (d *UserAttributeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserAttributeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserAttributeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var attribute *britive.UserAttribute
	var err error

	if !data.AttributeSchemaID.IsNull() && !data.AttributeSchemaID.IsUnknown() && data.AttributeSchemaID.ValueString() != "" {
		attribute, err = d.client.GetAttribute(data.AttributeSchemaID.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("User Attribute Not Found",
				fmt.Sprintf("User attribute with id '%s' was not found.", data.AttributeSchemaID.ValueString()))
			return
		}
	} else {
		attribute, err = d.client.GetAttributeByName(data.Name.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("User Attribute Not Found",
				fmt.Sprintf("User attribute '%s' was not found.", data.Name.ValueString()))
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User Attribute", err.Error())
		return
	}

	data.ID = types.StringValue(attribute.ID)
	data.Name = types.StringValue(attribute.Name)
	data.AttributeSchemaID = types.StringValue(attribute.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
