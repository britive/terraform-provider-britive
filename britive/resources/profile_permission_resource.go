package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ProfilePermissionResource{}
	_ resource.ResourceWithConfigure   = &ProfilePermissionResource{}
	_ resource.ResourceWithImportState = &ProfilePermissionResource{}
)

func NewProfilePermissionResource() resource.Resource {
	return &ProfilePermissionResource{}
}

type ProfilePermissionResource struct {
	client *britive.Client
}

type ProfilePermissionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	AppName        types.String `tfsdk:"app_name"`
	ProfileID      types.String `tfsdk:"profile_id"`
	ProfileName    types.String `tfsdk:"profile_name"`
	PermissionName types.String `tfsdk:"permission_name"`
	PermissionType types.String `tfsdk:"permission_type"`
}

func (r *ProfilePermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile_permission"
}

func (r *ProfilePermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive profile permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the profile permission.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The application name of the application the profile is associated with.",
			},
			"profile_id": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the profile.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the profile.",
			},
			"permission_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of permission.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of permission.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ProfilePermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ProfilePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfilePermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := plan.ProfileID.ValueString()
	permissionName := plan.PermissionName.ValueString()
	permissionType := plan.PermissionType.ValueString()

	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation: "add",
		Permission: britive.ProfilePermission{
			ProfileID: profileID,
			Name:      permissionName,
			Type:      permissionType,
		},
	}

	err := r.client.ExecuteProfilePermissionRequest(profileID, profilePermissionRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Profile Permission",
			fmt.Sprintf("Could not create profile permission: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(generateProfilePermissionID(profilePermissionRequest.Permission))

	// app_name and profile_name are Optional+Computed but not returned by the API at create time.
	// Set them to null if still unknown to avoid "provider returned unknown value" errors.
	if plan.AppName.IsUnknown() {
		plan.AppName = types.StringNull()
	}
	if plan.ProfileName.IsUnknown() {
		plan.ProfileName = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfilePermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfilePermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profilePermission, err := parseProfilePermissionID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Permission ID",
			fmt.Sprintf("Could not parse profile permission ID: %s", err.Error()),
		)
		return
	}

	pp, err := r.client.GetProfilePermission(profilePermission.ProfileID, *profilePermission)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Profile Permission",
			fmt.Sprintf("Could not read profile permission: %s", err.Error()),
		)
		return
	}

	state.ID = types.StringValue(generateProfilePermissionID(*pp))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfilePermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields have RequiresReplace, so Update should never be called
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All profile permission fields require replacement. This should not happen.",
	)
}

func (r *ProfilePermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfilePermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profilePermission, err := parseProfilePermissionID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Profile Permission ID",
			fmt.Sprintf("Could not parse profile permission ID: %s", err.Error()),
		)
		return
	}

	profilePermissionRequest := britive.ProfilePermissionRequest{
		Operation:  "remove",
		Permission: *profilePermission,
	}

	err = r.client.ExecuteProfilePermissionRequest(profilePermission.ProfileID, profilePermissionRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Profile Permission",
			fmt.Sprintf("Could not delete profile permission: %s", err.Error()),
		)
		return
	}
}

func (r *ProfilePermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two import formats:
	// 1. apps/{app_name}/paps/{profile_name}/permissions/{permission_name}/type/{permission_type}
	// 2. {app_name}/{profile_name}/{permission_name}/{permission_type}
	idRegexes := []string{
		`^apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/permissions/(?P<permission_name>.+)/type/(?P<permission_type>[^/]+)$`,
		`^(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<permission_name>.+)/(?P<permission_type>[^/]+)$`,
	}

	var appName, profileName, permissionName, permissionType string

	for _, pattern := range idRegexes {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(req.ID); matches != nil {
			for i, matchName := range re.SubexpNames() {
				if i == 0 {
					continue
				}
				switch matchName {
				case "app_name":
					appName = matches[i]
				case "profile_name":
					profileName = matches[i]
				case "permission_name":
					permissionName = matches[i]
				case "permission_type":
					permissionType = matches[i]
				}
			}
			break
		}
	}

	if appName == "" || profileName == "" || permissionName == "" || permissionType == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID %q doesn't match expected formats", req.ID),
		)
		return
	}

	// Validate fields
	if strings.TrimSpace(appName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "app_name cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(profileName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "profile_name cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(permissionName) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "permission_name cannot be empty or whitespace")
		return
	}
	if strings.TrimSpace(permissionType) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "permission_type cannot be empty or whitespace")
		return
	}

	// Get application by name
	app, err := r.client.GetApplicationByName(appName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Application Not Found",
			fmt.Sprintf("Application '%s' not found.", appName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Permission",
			fmt.Sprintf("Could not get application '%s': %s", appName, err.Error()),
		)
		return
	}

	// Get profile by name
	profile, err := r.client.GetProfileByName(app.AppContainerID, profileName)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Profile Not Found",
			fmt.Sprintf("Profile '%s' not found.", profileName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Permission",
			fmt.Sprintf("Could not get profile '%s': %s", profileName, err.Error()),
		)
		return
	}

	// Get profile permission
	profilePermission, err := r.client.GetProfilePermission(profile.ProfileID, britive.ProfilePermission{
		Name: permissionName,
		Type: permissionType,
	})
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Profile Permission Not Found",
			fmt.Sprintf("Permission '%s' of type '%s' not found in profile '%s'.", permissionName, permissionType, profileName),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Profile Permission",
			fmt.Sprintf("Could not get profile permission: %s", err.Error()),
		)
		return
	}

	profilePermission.ProfileID = profile.ProfileID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), generateProfilePermissionID(*profilePermission))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profile.ProfileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_name"), permissionName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_type"), permissionType)...)
	// Note: app_name and profile_name are not set (cleared like in SDKv2 version)
}

// Helper functions
func generateProfilePermissionID(profilePermission britive.ProfilePermission) string {
	return fmt.Sprintf("paps/%s/permissions/%s/type/%s", profilePermission.ProfileID, profilePermission.Name, profilePermission.Type)
}

func parseProfilePermissionID(id string) (*britive.ProfilePermission, error) {
	idFormat := `^paps/([^/]+)/permissions/(.+)/type/([^/]+)$`

	re := regexp.MustCompile(idFormat)
	matches := re.FindStringSubmatch(id)

	if matches == nil {
		return nil, fmt.Errorf("invalid profile permission ID format: %s", id)
	}

	return &britive.ProfilePermission{
		ProfileID: matches[1],
		Name:      matches[2],
		Type:      matches[3],
	}, nil
}
