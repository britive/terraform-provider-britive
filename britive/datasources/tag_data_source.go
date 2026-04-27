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
	_ datasource.DataSource              = &TagDataSource{}
	_ datasource.DataSourceWithConfigure = &TagDataSource{}
)

func NewTagDataSource() datasource.DataSource {
	return &TagDataSource{}
}

type TagDataSource struct {
	client *britive.Client
}

type TagDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	TagID types.String `tfsdk:"tag_id"`
}

func (d *TagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *TagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive tag.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the tag.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("tag_id"),
					),
				},
			},
			"tag_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the tag.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("tag_id"),
					),
				},
			},
		},
	}
}

func (d *TagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TagDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tag *britive.Tag
	var err error

	if !data.TagID.IsNull() && !data.TagID.IsUnknown() && data.TagID.ValueString() != "" {
		tag, err = d.client.GetTag(data.TagID.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("Tag Not Found",
				fmt.Sprintf("Tag with id '%s' was not found.", data.TagID.ValueString()))
			return
		}
	} else {
		tag, err = d.client.GetTagByName(data.Name.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("Tag Not Found",
				fmt.Sprintf("Tag '%s' was not found.", data.Name.ValueString()))
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Tag", err.Error())
		return
	}

	data.ID = types.StringValue(tag.ID)
	data.Name = types.StringValue(tag.Name)
	data.TagID = types.StringValue(tag.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
