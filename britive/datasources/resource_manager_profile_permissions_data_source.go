package datasources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ResourceManagerProfilePermissionsDataSource{}
	_ datasource.DataSourceWithConfigure = &ResourceManagerProfilePermissionsDataSource{}
)

func NewResourceManagerProfilePermissionsDataSource() datasource.DataSource {
	return &ResourceManagerProfilePermissionsDataSource{}
}

type ResourceManagerProfilePermissionsDataSource struct {
	client *britive.Client
}

type ResourceManagerProfilePermissionsDataSourceModel struct {
	ID          types.String             `tfsdk:"id"`
	ProfileID   types.String             `tfsdk:"profile_id"`
	Permissions []PermissionDetailsModel `tfsdk:"permissions"`
}

type PermissionDetailsModel struct {
	Name         types.String `tfsdk:"name"`
	PermissionID types.String `tfsdk:"permission_id"`
	Version      types.Set    `tfsdk:"version"`
}

func (d *ResourceManagerProfilePermissionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_profile_permissions"
}

func (d *ResourceManagerProfilePermissionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches available permissions for a resource manager profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The identifier for this data source.",
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "Profile ID (can include path prefix).",
			},
			"permissions": schema.SetNestedAttribute{
				Computed:    true,
				Description: "Available Permissions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of permission.",
						},
						"permission_id": schema.StringAttribute{
							Computed:    true,
							Description: "Permission ID.",
						},
						"version": schema.SetAttribute{
							ElementType: types.StringType,
							Computed:    true,
							Description: "Versions of the permission.",
						},
					},
				},
			},
		},
	}
}

func (d *ResourceManagerProfilePermissionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ResourceManagerProfilePermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceManagerProfilePermissionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract profile ID from path if needed
	profileIDArr := strings.Split(data.ProfileID.ValueString(), "/")
	profileID := profileIDArr[len(profileIDArr)-1]

	allAvailablePermissions, err := d.client.GetAvailablePermissions(profileID)
	if err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Permissions Not Found",
				fmt.Sprintf("Permissions not found for profile '%s'.", profileID),
			)
		} else {
			resp.Diagnostics.AddError(
				"Error Reading Permissions",
				fmt.Sprintf("Could not read permissions for profile '%s': %s", profileID, err.Error()),
			)
		}
		return
	}

	if allAvailablePermissions == nil || allAvailablePermissions.Permissions == nil {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"Received nil response or nil permissions list from API.",
		)
		return
	}

	permissions := make([]PermissionDetailsModel, 0, len(allAvailablePermissions.Permissions))

	for _, val := range allAvailablePermissions.Permissions {
		perm := PermissionDetailsModel{}

		if name, ok := val["name"].(string); ok {
			perm.Name = types.StringValue(name)
		}
		if pid, ok := val["permissionId"].(string); ok {
			perm.PermissionID = types.StringValue(pid)
		}

		// Get permission versions
		if !perm.PermissionID.IsNull() {
			permissionVersions, err := d.client.GetPermissionVersions(perm.PermissionID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Reading Permission Versions",
					fmt.Sprintf("Could not read versions for permission '%s': %s", perm.PermissionID.ValueString(), err.Error()),
				)
				return
			}

			versions := make([]string, 0, len(permissionVersions)+2)
			for _, v := range permissionVersions {
				if version, ok := v["version"].(string); ok {
					versions = append(versions, version)
				}
			}
			// Add standard versions
			versions = append(versions, "local", "latest")

			versionsSet, diags := types.SetValueFrom(ctx, types.StringType, versions)
			resp.Diagnostics.Append(diags...)
			perm.Version = versionsSet
		}

		permissions = append(permissions, perm)
	}

	data.ID = types.StringValue(fmt.Sprintf("profile/%s/available-permissions", profileID))
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
