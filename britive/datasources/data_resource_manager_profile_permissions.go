package datasources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &DataSourceResourceManagerProfilePermissions{}
	_ datasource.DataSourceWithConfigure = &DataSourceResourceManagerProfilePermissions{}
)

type DataSourceResourceManagerProfilePermissions struct {
	client *britive_client.Client
}

func NewDataSourceResourceManagerProfilePermissions() datasource.DataSource {
	return &DataSourceResourceManagerProfilePermissions{}
}

func (drmpp *DataSourceResourceManagerProfilePermissions) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_profile_permissions"
}

func (drmpp *DataSourceResourceManagerProfilePermissions) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {

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

	drmpp.client = client
	tflog.Info(ctx, "Configured DataSourceResourceManagerProfilePermissions with Britive client")
}

func (drmpp *DataSourceResourceManagerProfilePermissions) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Datasource for retrieving Britive Resource Manager Profile Permissions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of connection",
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "Unique identifier of profile",
			},
			"permissions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "resource manager profile permissions",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of permission",
						},
						"permission_id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of permission",
						},
						"version": schema.ListAttribute{
							Computed:    true,
							Description: "Versions of permission",
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (drmpp *DataSourceResourceManagerProfilePermissions) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_profile_permission datasource")

	var plan britive_client.DataResourceManagerProfilePermissionsPlan
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during fetching resource manager profile permissions", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	profileIDArr := strings.Split(plan.ProfileID.ValueString(), "/")
	profileID := profileIDArr[len(profileIDArr)-1]

	tflog.Info(ctx, fmt.Sprintf("Reading all available permissions for profile: %s", profileID))

	allAvailablePermissions, err := drmpp.client.GetAvailablePermissions(ctx, profileID)
	if err != nil {
		if errors.Is(err, britive_client.ErrNotFound) {
			resp.Diagnostics.AddError("Failed to get permissions", fmt.Sprintf("%s", errs.NewNotFoundErrorf("permissions not found").Error()))
			tflog.Error(ctx, fmt.Sprintf("Failed to get permissions, error:%s", errs.NewNotFoundErrorf("permissions not found").Error()))
			return
		}
		resp.Diagnostics.AddError("Failed to get permissions", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to get permissions, error:%s", err.Error()))
		return
	}

	tflog.Info(ctx, fmt.Sprintf(" all available permissions: %#v", allAvailablePermissions))

	if allAvailablePermissions == nil || allAvailablePermissions.Permissions == nil {
		resp.Diagnostics.AddError("Failed to get permissions", "received nil response or nil permissions list from GetAvailablePermissions")
		tflog.Error(ctx, "Failed to get permissions, error: received nil response or nil permissions list from GetAvailablePermissions")
		return
	}

	var permissions []britive_client.DataResourceManagerPermissionPlan

	for _, val := range allAvailablePermissions.Permissions {
		var perm britive_client.DataResourceManagerPermissionPlan

		if name, ok := val["name"].(string); ok {
			perm.Name = types.StringValue(name)
		}
		if pid, ok := val["permissionId"].(string); ok {
			perm.PermissionID = types.StringValue(pid)
		}

		permissionVersions, err := drmpp.client.GetPermissionVersions(ctx, perm.PermissionID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get permission versions", err.Error())
			tflog.Error(ctx, fmt.Sprintf("Failed to get permission versions, error:%#v", err))
			return
		}

		var version []string
		for _, v := range permissionVersions {
			version = append(version, v["version"].(string))
		}
		version = append(version, "local")
		version = append(version, "latest")

		perm.Version = version

		permissions = append(permissions, perm)
	}

	plan.ID = types.StringValue(fmt.Sprintf("profile/%s/available-permissions", profileID))
	plan.Permissions = permissions

	tflog.Info(ctx, fmt.Sprintf("Set all available permissions: %#v", permissions))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
	tflog.Info(ctx, "Update completed and state set", map[string]interface{}{
		"permissions": plan,
	})
}
