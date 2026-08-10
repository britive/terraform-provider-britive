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
	_ datasource.DataSource              = &SupportedConstraintsDataSource{}
	_ datasource.DataSourceWithConfigure = &SupportedConstraintsDataSource{}
)

func NewSupportedConstraintsDataSource() datasource.DataSource {
	return &SupportedConstraintsDataSource{}
}

type SupportedConstraintsDataSource struct {
	client *britive.Client
}

type SupportedConstraintsDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	ProfileID       types.String `tfsdk:"profile_id"`
	PermissionName  types.String `tfsdk:"permission_name"`
	PermissionType  types.String `tfsdk:"permission_type"`
	ConstraintTypes types.Set    `tfsdk:"constraint_types"`
}

func (d *SupportedConstraintsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_supported_constraints"
}

func (d *SupportedConstraintsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches supported constraint types for a profile permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this data source.",
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"permission_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the permission associated with the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"permission_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The type of permission. Defaults to 'role'.",
			},
			"constraint_types": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "A set of constraints supported for the given profile permission.",
			},
		},
	}
}

func (d *SupportedConstraintsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SupportedConstraintsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SupportedConstraintsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := data.ProfileID.ValueString()
	permissionName := data.PermissionName.ValueString()
	permissionType := data.PermissionType.ValueString()

	// Default to "role" if not specified
	if permissionType == "" {
		permissionType = "role"
		data.PermissionType = types.StringValue("role")
	}

	supportedConstraintTypes, err := d.client.GetSupportedConstraintTypes(profileID, permissionName, permissionType)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Constraint Types Not Found",
			fmt.Sprintf("Constraint types not found for profile '%s', permission '%s', type '%s'.", profileID, permissionName, permissionType),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Constraint Types",
			fmt.Sprintf("Could not read constraint types: %s", err.Error()),
		)
		return
	}

	// Generate unique ID
	data.ID = types.StringValue(fmt.Sprintf("paps/%s/permissions/%s/%s/supported-constraint-types", profileID, permissionName, permissionType))

	// Convert to Set
	constraintTypesSet, diags := types.SetValueFrom(ctx, types.StringType, supportedConstraintTypes)
	resp.Diagnostics.Append(diags...)
	data.ConstraintTypes = constraintTypesSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
