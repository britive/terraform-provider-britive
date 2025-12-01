package datasources

import (
	"context"
	"fmt"
	"regexp"

	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &DataSourceApplication{}
	_ datasource.DataSourceWithConfigure = &DataSourceApplication{}
)

type DataSourceApplication struct {
	client *britive_client.Client
}

func NewDataSourceApplication() datasource.DataSource {
	return &DataSourceApplication{}
}

func (da *DataSourceApplication) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_application"
}

func (da *DataSourceApplication) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

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

	da.client = client
	tflog.Info(ctx, "Configured DataSourceApplication with Britive client")
}

func (da *DataSourceApplication) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive application metadata.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of application",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the application",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`\S`),
						"must not be empty or whitespace",
					),
				},
			},
			"environment_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of environment ids for the application",
			},
			"environment_group_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of environment group ids for the application",
			},
		},
		Blocks: map[string]schema.Block{
			"environment_ids_names": schema.SetNestedBlock{
				Description: "A set of environment IDs and names for the application",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "The environment id",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`\S`),
									"must not be empty or whitespace",
								),
							},
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The environment name",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`\S`),
									"must not be empty or whitespace",
								),
							},
						},
					},
				},
			},
			"environment_group_ids_names": schema.SetNestedBlock{
				Description: "A set of environment group IDs and names for the application",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "The environment group id",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`\S`),
									"must not be empty or whitespace",
								),
							},
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The environment group name",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`\S`),
									"must not be empty or whitespace",
								),
							},
						},
					},
				},
			},
		},
	}
}

func (da *DataSourceApplication) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_application datasource")

	var plan britive_client.DataSourceApplicationPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching application", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	application, err := da.client.GetApplicationByName(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get application",
			fmt.Sprintf("Error: %v, Check applicatio name: %s", err, plan.Name.ValueString()),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get application of name: %s", err, plan.Name.ValueString()))
		return
	}

	plan.ID = types.StringValue(application.AppContainerID)

	appEnvs, err := da.client.GetAppEnvs(ctx, plan.ID.ValueString(), "environments")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get app environments",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get app environments", err))
		return
	}

	appEnvGroups, err := da.client.GetAppEnvs(ctx, plan.ID.ValueString(), "environmentGroups")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get app environment groups.",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get app environment groups.", err))
		return
	}

	envIdList, err := da.client.GetEnvDetails(appEnvs, "id")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get environment details",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get environment details", err))
		return
	}
	var envIdListElem []attr.Value
	for _, elem := range envIdList {
		envIdListElem = append(envIdListElem, types.StringValue(elem))
	}

	envIdSet, diag := types.SetValue(types.StringType, envIdListElem)
	if diag.HasError() {
		tflog.Error(ctx, "Failed to map environment id's to set")
		return
	}

	plan.EnvironmentIDs = envIdSet

	envGrpIdList, err := da.client.GetEnvDetails(appEnvGroups, "id")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get environment group details",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get environment group details", err))
		return
	}

	var envGrpIdListElem []attr.Value
	for _, elem := range envGrpIdList {
		envGrpIdListElem = append(envGrpIdListElem, types.StringValue(elem))
	}

	envGrpIdSet, diag := types.SetValue(types.StringType, envGrpIdListElem)
	if diag.HasError() {
		tflog.Error(ctx, "Failed to map environment groups to set")
		return
	}
	plan.EnvironmentGroupIDs = envGrpIdSet

	envIdNameList, err := da.client.GetEnvFullDetails(appEnvs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get full details of environtments",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get full details of environments", err))
		return
	}

	plan.EnvironmentIDsNames = nil
	for _, elem := range envIdNameList {
		var elemEnvIdNameData britive_client.DataSourceEnvironmentIDNamePlan
		elemEnvIdNameData.ID = types.StringValue(elem["id"])
		elemEnvIdNameData.Name = types.StringValue(elem["name"])
		plan.EnvironmentIDsNames = append(plan.EnvironmentIDsNames, elemEnvIdNameData)
	}

	envGrpIdNameList, err := da.client.GetEnvFullDetails(appEnvGroups)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get full details of environtment groups",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, fmt.Sprintf("Error: %v, Failed to get full details of environment groups", err))
		return
	}

	plan.EnvironmentGroupIDsNames = nil
	for _, elem := range envGrpIdNameList {
		var elemEnvGrpNameData britive_client.DataSourceEnvironmentGroupIDNamePlan
		elemEnvGrpNameData.ID = types.StringValue(elem["id"])
		elemEnvGrpNameData.Name = types.StringValue(elem["name"])
		plan.EnvironmentGroupIDsNames = append(plan.EnvironmentGroupIDsNames, elemEnvGrpNameData)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
		"application": plan,
	})

}
