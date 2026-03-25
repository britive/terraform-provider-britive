package datasources

import (
	"context"
	"errors"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &ApplicationDataSource{}
)

// NewApplicationDataSource is a helper function to simplify the provider implementation.
func NewApplicationDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

// ApplicationDataSource is the data source implementation.
type ApplicationDataSource struct {
	client *britive.Client
}

// ApplicationDataSourceModel describes the data source data model.
type ApplicationDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	EnvironmentIDs           types.Set    `tfsdk:"environment_ids"`
	EnvironmentGroupIDs      types.Set    `tfsdk:"environment_group_ids"`
	EnvironmentIDsNames      types.Set    `tfsdk:"environment_ids_names"`
	EnvironmentGroupIDsNames types.Set    `tfsdk:"environment_group_ids_names"`
}

// EnvironmentIDNameModel describes nested environment ID/name pairs
type EnvironmentIDNameModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// Metadata returns the data source type name.
func (d *ApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

// Schema defines the schema for the data source.
func (d *ApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a Britive application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the application.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the application.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"environment_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "A set of environment ids for the application.",
			},
			"environment_group_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "A set of environment group ids for the application.",
			},
			"environment_ids_names": schema.SetNestedAttribute{
				Computed:    true,
				Description: "A set of environment ids and names for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The environment id.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The environment name.",
						},
					},
				},
			},
			"environment_group_ids_names": schema.SetNestedAttribute{
				Computed:    true,
				Description: "A set of environment group ids and names for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The environment group id.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The environment group name.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data.
func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get application from API
	applicationName := data.Name.ValueString()
	application, err := d.client.GetApplicationByName(applicationName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Application Not Found",
			fmt.Sprintf("Application '%s' was not found.", applicationName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application",
			fmt.Sprintf("Could not read application '%s': %s", applicationName, err.Error()),
		)
		return
	}

	// Set ID and name
	data.ID = types.StringValue(application.AppContainerID)
	data.Name = types.StringValue(application.CatalogAppDisplayName)

	// Get environments
	appEnvs, err := d.client.GetAppEnvs(data.ID.ValueString(), "environments")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application Environments",
			fmt.Sprintf("Could not read environments for application '%s': %s", applicationName, err.Error()),
		)
		return
	}

	// Get environment groups
	appEnvGroups, err := d.client.GetAppEnvs(data.ID.ValueString(), "environmentGroups")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Application Environment Groups",
			fmt.Sprintf("Could not read environment groups for application '%s': %s", applicationName, err.Error()),
		)
		return
	}

	// Get environment IDs
	envIdList, err := d.client.GetEnvDetails(appEnvs, "id")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Environment IDs",
			fmt.Sprintf("Could not process environment IDs: %s", err.Error()),
		)
		return
	}
	envIdsSet, diags := types.SetValueFrom(ctx, types.StringType, envIdList)
	resp.Diagnostics.Append(diags...)
	data.EnvironmentIDs = envIdsSet

	// Get environment group IDs
	envGrpIdList, err := d.client.GetEnvDetails(appEnvGroups, "id")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Environment Group IDs",
			fmt.Sprintf("Could not process environment group IDs: %s", err.Error()),
		)
		return
	}
	envGrpIdsSet, diags := types.SetValueFrom(ctx, types.StringType, envGrpIdList)
	resp.Diagnostics.Append(diags...)
	data.EnvironmentGroupIDs = envGrpIdsSet

	// Get environment IDs and names
	envIdNameList, err := d.client.GetEnvFullDetails(appEnvs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Environment Details",
			fmt.Sprintf("Could not process environment details: %s", err.Error()),
		)
		return
	}
	envIDNames := make([]EnvironmentIDNameModel, 0, len(envIdNameList))
	for _, itemMap := range envIdNameList {
		envIDNames = append(envIDNames, EnvironmentIDNameModel{
			ID:   types.StringValue(itemMap["id"]),
			Name: types.StringValue(itemMap["name"]),
		})
	}
	envIDNamesSet, diags := types.SetValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		},
	}, envIDNames)
	resp.Diagnostics.Append(diags...)
	data.EnvironmentIDsNames = envIDNamesSet

	// Get environment group IDs and names
	envGrpIdNameList, err := d.client.GetEnvFullDetails(appEnvGroups)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Processing Environment Group Details",
			fmt.Sprintf("Could not process environment group details: %s", err.Error()),
		)
		return
	}
	envGrpIDNames := make([]EnvironmentIDNameModel, 0, len(envGrpIdNameList))
	for _, itemMap := range envGrpIdNameList {
		envGrpIDNames = append(envGrpIDNames, EnvironmentIDNameModel{
			ID:   types.StringValue(itemMap["id"]),
			Name: types.StringValue(itemMap["name"]),
		})
	}
	envGrpIDNamesSet, diags := types.SetValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		},
	}, envGrpIDNames)
	resp.Diagnostics.Append(diags...)
	data.EnvironmentGroupIDsNames = envGrpIDNamesSet

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
