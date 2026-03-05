package datasources

import (
	"context"
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
	_ datasource.DataSource              = &DataSourceSupportedConstraints{}
	_ datasource.DataSourceWithConfigure = &DataSourceSupportedConstraints{}
)

type DataSourceSupportedConstraints struct {
	client *britive_client.Client
	helper *DataSourceSupportedConstraintsHelper
}

type DataSourceSupportedConstraintsHelper struct{}

func NewDataSourceSupportedConstraints() datasource.DataSource {
	return &DataSourceSupportedConstraints{}
}

func NewDataSourceSupportedConstraintsHelper() *DataSourceSupportedConstraintsHelper {
	return &DataSourceSupportedConstraintsHelper{}
}

func (dsc *DataSourceSupportedConstraints) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_supported_constraints"
}

func (dsc *DataSourceSupportedConstraints) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

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

	dsc.client = client
	tflog.Info(ctx, "Configured DataSourceSupportedConstraints with Britive client")
}

func (dsc *DataSourceSupportedConstraints) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive supported constraints.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of constraints",
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of profile",
				Validators: []validator.String{
					validate.StringFunc(
						"profileID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"permission_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of permission associated with profile",
				Validators: []validator.String{
					validate.StringFunc(
						"profileID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"permission_type": schema.StringAttribute{
				Optional:    true,
				Description: "The type of permission",
				Validators: []validator.String{
					validate.StringFunc(
						"profileID",
						validate.StringIsNotWhiteSpace(),
					),
				},
			},
			"constraint_types": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Description: "List of constraints supported for givem profile permission",
				ElementType: types.StringType,
			},
		},
	}
}

func (dsc *DataSourceSupportedConstraints) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_supported_constraints datasource")

	var plan britive_client.DataSourceSupportedConstraintsPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching supported constraints", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileId := plan.ProfileID.ValueString()
	permissionName := plan.PermissionName.ValueString()
	var permissionType string
	if plan.PermissionType.IsNull() || plan.PermissionType.IsUnknown() {
		permissionType = "role"
	} else {
		permissionType = plan.PermissionType.ValueString()
	}
	supportedConstraintTypes, err := dsc.client.GetSupportedConstraintTypes(ctx, profileId, permissionName, permissionType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch supported constraint types", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to fetch supported constraint types, error: %#v", err))
		return
	}

	plan.ID = types.StringValue(dsc.helper.generateUniqueID(profileId, permissionName, permissionType))

	constraintTypes, diags := types.SetValueFrom(
		ctx,
		types.StringType,
		supportedConstraintTypes,
	)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	plan.ConstraintTypes = constraintTypes

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
		"constraints": plan,
	})
}

func (dsch *DataSourceSupportedConstraintsHelper) generateUniqueID(profileID, permissionName, permissionType string) string {
	return fmt.Sprintf("paps/%s/permissions/%s/%s/supported-constraint-types", profileID, permissionName, permissionType)
}
