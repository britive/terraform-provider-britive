package datasources

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

type TagAttributeDataModel struct {
	AttributeName  types.String `tfsdk:"attribute_name"`
	AttributeValue types.String `tfsdk:"attribute_value"`
}

type TagDataSourceModel struct {
	ID                 types.String            `tfsdk:"id"`
	Name               types.String            `tfsdk:"name"`
	TagID              types.String            `tfsdk:"tag_id"`
	IdentityProviderID types.String            `tfsdk:"identity_provider_id"`
	Description        types.String            `tfsdk:"description"`
	Disabled           types.Bool              `tfsdk:"disabled"`
	Requestable        types.Bool              `tfsdk:"requestable"`
	External           types.Bool              `tfsdk:"external"`
	ExternalID         types.String            `tfsdk:"external_id"`
	Attributes         []TagAttributeDataModel `tfsdk:"attributes"`
}

func (d *TagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *TagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive tag by id, or by name optionally scoped to an identity provider. " +
			"Tag names are unique per identity provider only; a name that matches tags on several providers is an error " +
			"unless identity_provider_id is set.",
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
			"identity_provider_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The identity provider the tag belongs to. With name, narrows the lookup to that provider.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("tag_id")),
				},
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the tag.",
			},
			"disabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the tag is disabled.",
			},
			"requestable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the tag is requestable.",
			},
			"external": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the tag is connected to its identity provider through SCIM.",
			},
			"external_id": schema.StringAttribute{
				Computed:    true,
				Description: "The identity provider's identifier for a connected tag, when supplied.",
			},
			"attributes": schema.SetNestedAttribute{
				Computed:    true,
				Description: "The attributes of the tag.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"attribute_name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the attribute.",
						},
						"attribute_value": schema.StringAttribute{
							Computed:    true,
							Description: "The value of the attribute.",
						},
					},
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

	if !data.TagID.IsNull() && !data.TagID.IsUnknown() && data.TagID.ValueString() != "" {
		t, err := d.client.GetTag(data.TagID.ValueString())
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("Tag Not Found",
				fmt.Sprintf("Tag with id '%s' was not found.", data.TagID.ValueString()))
			return
		}
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Tag", err.Error())
			return
		}
		tag = t
	} else {
		name := data.Name.ValueString()
		tags, err := d.client.GetTagsByName(name)
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError("Tag Not Found", fmt.Sprintf("Tag '%s' was not found.", name))
			return
		}
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Tag", err.Error())
			return
		}

		if !data.IdentityProviderID.IsNull() && !data.IdentityProviderID.IsUnknown() && data.IdentityProviderID.ValueString() != "" {
			tags = filterTagsByIdentityProvider(tags, data.IdentityProviderID.ValueString())
			if len(tags) == 0 {
				resp.Diagnostics.AddError("Tag Not Found",
					fmt.Sprintf("Tag '%s' was not found on identity provider '%s'.", name, data.IdentityProviderID.ValueString()))
				return
			}
		}

		if len(tags) > 1 {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Ambiguous Tag Name",
				fmt.Sprintf("%d tags are named '%s'. Set identity_provider_id to choose one, or look the tag up by tag_id:\n%s",
					len(tags), name, describeTagCandidates(tags)))
			return
		}
		tag = &tags[0]
	}

	data.ID = types.StringValue(tag.ID)
	data.TagID = types.StringValue(tag.ID)
	data.Name = types.StringValue(tag.Name)
	data.Description = types.StringValue(tag.Description)
	data.Disabled = types.BoolValue(strings.EqualFold(tag.Status, "Inactive"))
	data.Requestable = types.BoolValue(tag.Requestable)
	data.External = types.BoolValue(tag.External)
	data.ExternalID = types.StringValue("")
	data.IdentityProviderID = types.StringValue("")
	if len(tag.UserTagIdentityProviders) > 0 {
		data.IdentityProviderID = types.StringValue(tag.UserTagIdentityProviders[0].IdentityProvider.ID)
		if ext, ok := tag.UserTagIdentityProviders[0].ExternalID.(string); ok {
			data.ExternalID = types.StringValue(ext)
		}
	}
	data.Attributes = make([]TagAttributeDataModel, 0, len(tag.Attributes))
	for _, a := range tag.Attributes {
		data.Attributes = append(data.Attributes, TagAttributeDataModel{
			AttributeName:  types.StringValue(a.AttributeName),
			AttributeValue: types.StringValue(a.AttributeValue),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func filterTagsByIdentityProvider(tags []britive.Tag, identityProviderID string) []britive.Tag {
	out := make([]britive.Tag, 0, len(tags))
	for _, t := range tags {
		if len(t.UserTagIdentityProviders) > 0 && t.UserTagIdentityProviders[0].IdentityProvider.ID == identityProviderID {
			out = append(out, t)
		}
	}
	return out
}

func describeTagCandidates(tags []britive.Tag) string {
	lines := make([]string, 0, len(tags))
	for _, t := range tags {
		idp := "unknown identity provider"
		if len(t.UserTagIdentityProviders) > 0 {
			idp = fmt.Sprintf("%s, %s", t.UserTagIdentityProviders[0].IdentityProvider.Name, t.UserTagIdentityProviders[0].IdentityProvider.ID)
		}
		kind := "placeholder"
		if t.External {
			kind = "connected"
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s; %s)", t.ID, idp, kind))
	}
	return strings.Join(lines, "\n")
}
