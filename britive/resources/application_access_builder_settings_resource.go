package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/schedulescan"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ApplicationAccessBuilderSettingsResource{}
	_ resource.ResourceWithConfigure      = &ApplicationAccessBuilderSettingsResource{}
	_ resource.ResourceWithImportState    = &ApplicationAccessBuilderSettingsResource{}
	_ resource.ResourceWithValidateConfig = &ApplicationAccessBuilderSettingsResource{}
)

// Association approver types, as the API encodes them.
const (
	associationApproverTypeEnvironment      = 0
	associationApproverTypeEnvironmentGroup = 1
)

// Approval timeout bounds enforced by the API, in minutes.
const (
	approvalTimeoutMinMinutes = 5
	approvalTimeoutMaxMinutes = 5 * 24 * 60
)

var approvalTimeoutPattern = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)

// ApplicationAccessBuilderSettingsResource manages an application's access builder settings
// together with all of its association approvers. The API stores association approvers inside
// the settings record (they are read from it, and removed through it), so one resource owns
// both: that keeps every write for an application in one place and lets the provider order the
// calls itself - approvers exist before access requests are enabled, and are removed before
// new ones are created, so an environment can move between approvers in one apply.
type ApplicationAccessBuilderSettingsResource struct {
	client *britive.Client
}

// ApplicationAccessBuilderSettingsResourceModel describes the resource data model. Every
// repeated block is a types.Set, not a plain Go slice - see
// ApplicationApproversGroupResourceModel.Members for why.
type ApplicationAccessBuilderSettingsResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	ApplicationID            types.String `tfsdk:"application_id"`
	AllowAccessRequest       types.Bool   `tfsdk:"allow_access_request"`
	ApprovalTimeout          types.String `tfsdk:"approval_timeout"`
	ProfileExpirationTimeout types.Int64  `tfsdk:"profile_expiration_timeout"`
	ManagerApprovalRequired  types.Bool   `tfsdk:"manager_approval_required"`
	ManagerApprovalCondition types.String `tfsdk:"manager_approval_condition"`
	NotificationMediums      types.Set    `tfsdk:"notification_medium"`
	MemberRules              types.Set    `tfsdk:"member_rule"`
	AssociationApprovers     types.Set    `tfsdk:"association_approver"`
}

// notificationMediumModel is a notification medium referenced by either id or name.
type notificationMediumModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

var notificationMediumObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	},
}

// memberRuleModel is a user or tag included in or excluded from access requests.
type memberRuleModel struct {
	Type      types.String `tfsdk:"type"`
	Condition types.String `tfsdk:"condition"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
}

var memberRuleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":      types.StringType,
		"condition": types.StringType,
		"id":        types.StringType,
		"name":      types.StringType,
	},
}

// associationApproverModel maps environments or environment groups to approvers groups. It has
// no computed id: a computed field inside a set element would make every element change on each
// plan. Approvers are matched to the API's records by name, which is unique per application.
type associationApproverModel struct {
	Name              types.String `tfsdk:"name"`
	Associations      types.Set    `tfsdk:"association"`
	ApproversGroupIDs types.Set    `tfsdk:"approvers_group_ids"`
}

// accessRequestAssociationModel is an environment or environment group, by name or ID.
type accessRequestAssociationModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

var accessRequestAssociationObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	},
}

var associationApproverObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":                types.StringType,
		"association":         types.SetType{ElemType: accessRequestAssociationObjectType},
		"approvers_group_ids": types.SetType{ElemType: types.StringType},
	},
}

func notificationMediumsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []notificationMediumModel {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var v []notificationMediumModel
	diags.Append(set.ElementsAs(ctx, &v, false)...)
	return v
}

func notificationMediumsToSet(ctx context.Context, v []notificationMediumModel, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, notificationMediumObjectType, v)
	diags.Append(d...)
	return set
}

func memberRulesFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []memberRuleModel {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var v []memberRuleModel
	diags.Append(set.ElementsAs(ctx, &v, false)...)
	return v
}

func memberRulesToSet(ctx context.Context, v []memberRuleModel, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, memberRuleObjectType, v)
	diags.Append(d...)
	return set
}

func associationApproversFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []associationApproverModel {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var v []associationApproverModel
	diags.Append(set.ElementsAs(ctx, &v, false)...)
	return v
}

func associationApproversToSet(ctx context.Context, v []associationApproverModel, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, associationApproverObjectType, v)
	diags.Append(d...)
	return set
}

func accessRequestAssociationsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []accessRequestAssociationModel {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var v []accessRequestAssociationModel
	diags.Append(set.ElementsAs(ctx, &v, false)...)
	return v
}

func accessRequestAssociationsToSet(ctx context.Context, v []accessRequestAssociationModel, diags *diag.Diagnostics) types.Set {
	set, d := types.SetValueFrom(ctx, accessRequestAssociationObjectType, v)
	diags.Append(d...)
	return set
}

func stringsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var v []string
	diags.Append(set.ElementsAs(ctx, &v, false)...)
	return v
}

// NewApplicationAccessBuilderSettingsResource is a helper function to simplify the provider implementation.
func NewApplicationAccessBuilderSettingsResource() resource.Resource {
	return &ApplicationAccessBuilderSettingsResource{}
}

// Metadata returns the resource type name.
func (r *ApplicationAccessBuilderSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_builder_settings"
}

// Schema defines the schema for the resource.
func (r *ApplicationAccessBuilderSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages access builder settings and association approvers for a Britive application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the access builder settings.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the application.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow_access_request": schema.BoolAttribute{
				Description: "Whether users can request access. Left unmanaged when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"approval_timeout": schema.StringAttribute{
				Description: "How long a request waits for approval, in DD:HH:MM format. Must be from 5 minutes (00:00:05) to 5 days (05:00:00). Left unmanaged when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(approvalTimeoutPattern, "must be in DD:HH:MM format"),
				},
			},
			"profile_expiration_timeout": schema.Int64Attribute{
				Description: "How long approved access lasts, in minutes. The valid range is enforced by the API. Left unmanaged when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"manager_approval_required": schema.BoolAttribute{
				Description: "Whether the requester's manager must approve. Left unmanaged when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"manager_approval_condition": schema.StringAttribute{
				Description: "Which manager approval is required. One of Any, All, Manager. Left unmanaged when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("Any", "All", "Manager"),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"notification_medium": schema.SetNestedBlock{
				Description: "A notification medium for access requests. When no notification_medium blocks are configured, notification mediums are left unmanaged. Removing every notification_medium block sends an empty list, which the API rejects while access requests are enabled.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the notification medium. Exactly one of id or name must be set.",
							Optional:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the notification medium, for example Email. Exactly one of id or name must be set.",
							Optional:    true,
						},
					},
				},
			},
			"member_rule": schema.SetNestedBlock{
				Description: "A user or tag included in or excluded from access requests.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The member type. One of User, Tag.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("User", "Tag"),
							},
						},
						"condition": schema.StringAttribute{
							Description: "Whether the member is included or excluded. One of Include, Exclude.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("Include", "Exclude"),
							},
						},
						"id": schema.StringAttribute{
							Description: "The ID of the user or tag. Exactly one of id or name must be set.",
							Optional:    true,
						},
						"name": schema.StringAttribute{
							Description: "The username of the user, or the name of the tag. Exactly one of id or name must be set.",
							Optional:    true,
						},
					},
				},
			},
			"association_approver": schema.SetNestedBlock{
				Description: "Maps environments or environment groups to approvers groups. When no association_approver blocks are configured, association approvers are left unmanaged. An environment or environment group can belong to only one association approver, including parent and child groups.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the association approver. Must be unique within the application.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"approvers_group_ids": schema.SetAttribute{
							Description: "The IDs of the approvers groups. At least one is required.",
							Required:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"association": schema.SetNestedBlock{
							Description: "An environment or environment group. At least one is required.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Description: "One of Environment, EnvironmentGroup (case-insensitive).",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOfCaseInsensitive("Environment", "EnvironmentGroup"),
										},
									},
									"value": schema.StringAttribute{
										Description: "The environment or environment group, by name or ID.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthAtLeast(1),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ApplicationAccessBuilderSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig checks required blocks, id/name exclusivity, unique approver names, and the
// approval and profile expiration timeouts. Unknown values are skipped.
func (r *ApplicationAccessBuilderSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ApplicationAccessBuilderSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if isSet(data.ApprovalTimeout) {
		if _, err := parseApprovalTimeoutMinutes(data.ApprovalTimeout.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("approval_timeout"), "Invalid Approval Timeout", err.Error())
		}
	}
	if !data.ProfileExpirationTimeout.IsNull() && !data.ProfileExpirationTimeout.IsUnknown() && data.ProfileExpirationTimeout.ValueInt64() < 1 {
		resp.Diagnostics.AddAttributeError(path.Root("profile_expiration_timeout"), "Invalid Profile Expiration Timeout", "profile_expiration_timeout must be at least 1 minute.")
	}

	for _, m := range notificationMediumsFromSet(ctx, data.NotificationMediums, &resp.Diagnostics) {
		validateIDOrName(m.ID, m.Name, path.Root("notification_medium"), "notification_medium", &resp.Diagnostics)
	}

	for _, m := range memberRulesFromSet(ctx, data.MemberRules, &resp.Diagnostics) {
		validateIDOrName(m.ID, m.Name, path.Root("member_rule"), "member_rule", &resp.Diagnostics)
	}

	names := make(map[string]bool)
	for _, a := range associationApproversFromSet(ctx, data.AssociationApprovers, &resp.Diagnostics) {
		validateBlockRequired(a.Associations, path.Root("association_approver"), "association", &resp.Diagnostics)
		if !a.ApproversGroupIDs.IsNull() && !a.ApproversGroupIDs.IsUnknown() && len(a.ApproversGroupIDs.Elements()) == 0 {
			resp.Diagnostics.AddAttributeError(path.Root("association_approver"), "Invalid Configuration", "Each association_approver must have at least one approvers_group_ids entry.")
		}
		if !isSet(a.Name) {
			continue
		}
		if names[a.Name.ValueString()] {
			resp.Diagnostics.AddAttributeError(
				path.Root("association_approver"),
				"Duplicate Association Approver Name",
				fmt.Sprintf("More than one association_approver block is named %q. Names must be unique within an application.", a.Name.ValueString()),
			)
		}
		names[a.Name.ValueString()] = true
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ApplicationAccessBuilderSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config ApplicationAccessBuilderSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueString()
	log.Printf("[INFO] Creating access builder settings for application: %s", applicationID)

	current, err := r.getOrCreateSettings(applicationID, &config)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Access Builder Settings", err.Error())
		return
	}

	plan.ID = types.StringValue(accessBuilderSettingsCompositeID(applicationID))
	resp.Diagnostics.Append(setManagedFields(ctx, resp.Private, &config)...)

	err = r.apply(ctx, &plan, &config, nil, current, &resp.Diagnostics)
	r.refreshUnknownFields(applicationID, &plan)
	if err != nil || resp.Diagnostics.HasError() {
		if err != nil {
			resp.Diagnostics.AddError("Error Creating Access Builder Settings", err.Error())
		}
		// Some changes may already have been applied. Saving state marks the resource as
		// tainted, so the next apply resets it and starts again.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	log.Printf("[INFO] Created access builder settings for application: %s", applicationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ApplicationAccessBuilderSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationAccessBuilderSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	log.Printf("[INFO] Reading access builder settings for application: %s", applicationID)

	settings, err := r.client.GetAccessBuilderSettings(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Access Builder Settings", err.Error())
		return
	}

	if err := r.mapModelToResource(ctx, applicationID, settings, &state, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error Reading Access Builder Settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ApplicationAccessBuilderSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state, config ApplicationAccessBuilderSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	log.Printf("[INFO] Updating access builder settings for application: %s", applicationID)

	current, err := r.getOrCreateSettings(applicationID, &config)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Access Builder Settings", err.Error())
		return
	}

	plan.ID = state.ID

	if err := r.apply(ctx, &plan, &config, &state, current, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error Updating Access Builder Settings", err.Error())
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	r.refreshUnknownFields(applicationID, &plan)
	resp.Diagnostics.Append(setManagedFields(ctx, resp.Private, &config)...)

	log.Printf("[INFO] Updated access builder settings for application: %s", applicationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete resets only what this resource manages: it removes the association approvers and
// member rules it manages, resets manager approval if configured, and disables access
// requests if configured (or if approvers must be removed, since the API rejects removing the
// last approver while access requests are enabled). The API has no delete route for the
// settings record itself, and its timeouts and notification mediums cannot be cleared.
func (r *ApplicationAccessBuilderSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationAccessBuilderSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managed, diags := getManagedFields(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueString()
	log.Printf("[INFO] Deleting access builder settings for application: %s", applicationID)

	settings, err := r.client.GetAccessBuilderSettings(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Access Builder Settings", err.Error())
		return
	}

	approversManaged := hasElements(state.AssociationApprovers)
	rulesManaged := hasElements(state.MemberRules)

	if settings.AllowAccessRequest && (managed.AllowAccessRequest || approversManaged) {
		disabled := false
		if err := r.client.PatchAccessBuilderSettings(applicationID, britive.AccessBuilderSettingsPatch{AllowAccessRequest: &disabled}); err != nil {
			resp.Diagnostics.AddError("Error Deleting Access Builder Settings", err.Error())
			return
		}
	}

	patch := britive.AccessBuilderSettingsPatch{}
	if rulesManaged {
		noMemberRules := []britive.AccessRequestMemberRule{}
		patch.MemberRules = &noMemberRules
	}
	if managed.ManagerApproval {
		patch.ManagerApproval = &britive.AccessRequestManagerApproval{Required: false, Condition: "Any"}
	}
	if approversManaged {
		noApprovers := []britive.AssociationApprover{}
		patch.AssociationApprovers = &noApprovers
	}
	if patch != (britive.AccessBuilderSettingsPatch{}) {
		if err := r.client.PatchAccessBuilderSettings(applicationID, patch); err != nil {
			resp.Diagnostics.AddError("Error Deleting Access Builder Settings", err.Error())
			return
		}
	}

	log.Printf("[INFO] Deleted access builder settings for application: %s", applicationID)
}

// ImportState imports the resource state.
func (r *ApplicationAccessBuilderSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	applicationID, err := parseAccessBuilderSettingsCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	log.Printf("[INFO] Importing access builder settings for application: %s", applicationID)

	settings, err := r.client.GetAccessBuilderSettings(applicationID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Access Builder Settings Not Found", fmt.Sprintf("Application %s has no access builder settings", applicationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Access Builder Settings", err.Error())
		return
	}

	state := ApplicationAccessBuilderSettingsResourceModel{
		ID:                   types.StringValue(accessBuilderSettingsCompositeID(applicationID)),
		ApplicationID:        types.StringValue(applicationID),
		NotificationMediums:  types.SetNull(notificationMediumObjectType),
		MemberRules:          types.SetNull(memberRuleObjectType),
		AssociationApprovers: types.SetNull(associationApproverObjectType),
	}
	if err := r.mapModelToResource(ctx, applicationID, settings, &state, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Error Getting Access Builder Settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

// managedFieldsKey is the private state key recording which optional settings fields are set
// in config, so Delete resets only those. Blocks don't need it: Read only tracks blocks that
// are configured, so a block is managed exactly when it has elements in state.
const managedFieldsKey = "managed_fields"

// managedFields records which optional scalar fields are set in config.
type managedFields struct {
	AllowAccessRequest bool `json:"allow_access_request"`
	ManagerApproval    bool `json:"manager_approval"`
}

// privateState is the subset of the framework's private state accessors used here.
type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func setManagedFields(ctx context.Context, private privateState, config *ApplicationAccessBuilderSettingsResourceModel) diag.Diagnostics {
	value, err := json.Marshal(managedFields{
		AllowAccessRequest: !config.AllowAccessRequest.IsNull(),
		ManagerApproval:    !config.ManagerApprovalRequired.IsNull() || !config.ManagerApprovalCondition.IsNull(),
	})
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error Saving Private State", err.Error())
		return diags
	}
	return private.SetKey(ctx, managedFieldsKey, value)
}

func getManagedFields(ctx context.Context, private privateState) (managedFields, diag.Diagnostics) {
	var managed managedFields
	value, diags := private.GetKey(ctx, managedFieldsKey)
	if diags.HasError() || len(value) == 0 {
		return managed, diags
	}
	if err := json.Unmarshal(value, &managed); err != nil {
		diags.AddError("Error Reading Private State", err.Error())
	}
	return managed, diags
}

// hasElements reports whether a set is known and has at least one element.
func hasElements(set types.Set) bool {
	return !set.IsNull() && !set.IsUnknown() && len(set.Elements()) > 0
}

// getOrCreateSettings returns the application's settings record, creating it first when it
// doesn't exist. The first association approver created for an application also creates the
// record, so an existing record is adopted rather than treated as a conflict.
func (r *ApplicationAccessBuilderSettingsResource) getOrCreateSettings(applicationID string, config *ApplicationAccessBuilderSettingsResourceModel) (*britive.AccessBuilderSettings, error) {
	settings, err := r.client.GetAccessBuilderSettings(applicationID)
	if err == nil {
		return settings, nil
	}
	if !errors.Is(err, britive.ErrNotFound) {
		return nil, err
	}

	managerApproval := britive.AccessRequestManagerApproval{Required: false, Condition: "Any"}
	if !config.ManagerApprovalRequired.IsNull() {
		managerApproval.Required = config.ManagerApprovalRequired.ValueBool()
	}
	if !config.ManagerApprovalCondition.IsNull() {
		managerApproval.Condition = config.ManagerApprovalCondition.ValueString()
	}
	if err := r.client.CreateAccessBuilderSettings(applicationID, managerApproval); err != nil {
		return nil, err
	}
	return r.client.GetAccessBuilderSettings(applicationID)
}

// refreshUnknownFields fills optional scalar fields that were left out of config (and so are
// still unknown in plan) with the API's current values.
func (r *ApplicationAccessBuilderSettingsResource) refreshUnknownFields(applicationID string, plan *ApplicationAccessBuilderSettingsResourceModel) {
	settings, err := r.client.GetAccessBuilderSettings(applicationID)
	if err != nil {
		settings = &britive.AccessBuilderSettings{}
	}

	if plan.AllowAccessRequest.IsUnknown() {
		plan.AllowAccessRequest = types.BoolValue(settings.AllowAccessRequest)
	}
	if plan.ApprovalTimeout.IsUnknown() {
		plan.ApprovalTimeout = types.StringNull()
		if settings.ApprovalTimeOut != nil {
			plan.ApprovalTimeout = types.StringValue(*settings.ApprovalTimeOut)
		}
	}
	if plan.ProfileExpirationTimeout.IsUnknown() {
		plan.ProfileExpirationTimeout = types.Int64Null()
		if settings.ProfileExpirationTimeout.Valid {
			plan.ProfileExpirationTimeout = types.Int64Value(settings.ProfileExpirationTimeout.Value)
		}
	}
	if plan.ManagerApprovalRequired.IsUnknown() {
		plan.ManagerApprovalRequired = types.BoolNull()
		if settings.ManagerApproval != nil {
			plan.ManagerApprovalRequired = types.BoolValue(settings.ManagerApproval.Required)
		}
	}
	if plan.ManagerApprovalCondition.IsUnknown() {
		plan.ManagerApprovalCondition = types.StringNull()
		if settings.ManagerApproval != nil {
			plan.ManagerApprovalCondition = types.StringValue(settings.ManagerApproval.Condition)
		}
	}
}

// resolvedAssociationApprover is an association_approver block resolved to API IDs.
type resolvedAssociationApprover struct {
	Name         string
	Associations []britive.AssociationApproverAssociation
	GroupIDs     []string
}

// apply makes the application's settings match the configured fields, starting from current.
// Only what is configured is changed: scalar fields are sent only when set in config, and a
// block is managed when it has elements in config, or had elements in prior state (removing
// every block of a kind sends an empty list, which clears it on the server). Every name is
// resolved before the first write, so a bad reference fails before anything changes. The call
// order follows the API's rules:
//
//  1. Disable access requests first when config disables them, or when every existing approver
//     is about to be removed (the API rejects removing the last approver while enabled).
//  2. Remove approvers that are no longer configured, in one settings PATCH.
//  3. Update changed approvers, those that give up associations first, then create new ones.
//     This frees an environment before another approver claims it.
//  4. PATCH the configured settings fields together.
//  5. Enable access requests last, once approvers and notification mediums exist.
func (r *ApplicationAccessBuilderSettingsResource) apply(ctx context.Context, plan, config, prior *ApplicationAccessBuilderSettingsResourceModel, current *britive.AccessBuilderSettings, diags *diag.Diagnostics) error {
	applicationID := plan.ApplicationID.ValueString()

	manageMediums := hasElements(plan.NotificationMediums) || (prior != nil && hasElements(prior.NotificationMediums))
	manageRules := hasElements(plan.MemberRules) || (prior != nil && hasElements(prior.MemberRules))
	manageApprovers := hasElements(plan.AssociationApprovers) || (prior != nil && hasElements(prior.AssociationApprovers))

	var mediums []britive.AccessRequestNotificationMedium
	var memberRules []britive.AccessRequestMemberRule
	var desired []resolvedAssociationApprover
	var err error
	if manageMediums {
		if mediums, err = r.resolveNotificationMediums(ctx, plan.NotificationMediums, diags); err != nil || diags.HasError() {
			return err
		}
	}
	if manageRules {
		if memberRules, err = r.resolveMemberRules(ctx, plan.MemberRules, diags); err != nil || diags.HasError() {
			return err
		}
	}
	if manageApprovers {
		if desired, err = r.resolveAssociationApprovers(ctx, applicationID, plan.AssociationApprovers, diags); err != nil || diags.HasError() {
			return err
		}
	}

	targetAllow := current.AllowAccessRequest
	if !config.AllowAccessRequest.IsNull() && !config.AllowAccessRequest.IsUnknown() {
		targetAllow = config.AllowAccessRequest.ValueBool()
	}

	desiredByName := make(map[string]resolvedAssociationApprover, len(desired))
	for _, d := range desired {
		desiredByName[d.Name] = d
	}
	currentByName := make(map[string]britive.AssociationApprover, len(current.AssociationApprovers))
	remaining := make([]britive.AssociationApprover, 0, len(current.AssociationApprovers))
	for _, c := range current.AssociationApprovers {
		currentByName[c.Name] = c
		if _, ok := desiredByName[c.Name]; ok {
			remaining = append(remaining, c)
		}
	}
	removing := manageApprovers && len(remaining) < len(current.AssociationApprovers)

	// 1. Disable access requests.
	enabled := current.AllowAccessRequest
	if enabled && (!targetAllow || (removing && len(remaining) == 0)) {
		disabled := false
		if err := r.client.PatchAccessBuilderSettings(applicationID, britive.AccessBuilderSettingsPatch{AllowAccessRequest: &disabled}); err != nil {
			return fmt.Errorf("disabling access requests: %w", err)
		}
		enabled = false
	}

	// 2. Remove approvers that are no longer configured.
	if removing {
		if err := r.client.PatchAccessBuilderSettings(applicationID, britive.AccessBuilderSettingsPatch{AssociationApprovers: &remaining}); err != nil {
			return fmt.Errorf("removing association approvers: %w", err)
		}
	}

	// 3. Update changed approvers, then create new ones.
	type approverChange struct {
		id       string
		desired  resolvedAssociationApprover
		shrinks  bool
		assocs   bool
		groupIDs bool
	}
	changes := make([]approverChange, 0)
	creates := make([]resolvedAssociationApprover, 0)
	for _, d := range desired {
		c, ok := currentByName[d.Name]
		if !ok {
			creates = append(creates, d)
			continue
		}
		currentAssocs := associationKeys(c.Associations)
		desiredAssocs := associationKeys(d.Associations)
		change := approverChange{
			id:       c.ID,
			desired:  d,
			shrinks:  !isSubset(currentAssocs, desiredAssocs),
			assocs:   !equalStrings(currentAssocs, desiredAssocs),
			groupIDs: !equalStrings(approversGroupIDs(c.ApproversGroups), sortedCopy(d.GroupIDs)),
		}
		if change.assocs || change.groupIDs {
			changes = append(changes, change)
		}
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].shrinks && !changes[j].shrinks })

	for _, ch := range changes {
		patch := britive.AssociationApproverPatch{}
		if ch.assocs {
			assocs := ch.desired.Associations
			patch.Associations = &assocs
		}
		if ch.groupIDs {
			groups := approversGroupRefs(ch.desired.GroupIDs)
			patch.ApproversGroups = &groups
		}
		if err := r.client.PatchAssociationApprover(applicationID, ch.id, patch); err != nil {
			return fmt.Errorf("updating association approver %q: %w", ch.desired.Name, withOverlapHint(err))
		}
	}

	for _, d := range creates {
		if _, err := r.client.CreateAssociationApprover(applicationID, britive.AssociationApprover{
			Name:            d.Name,
			Associations:    d.Associations,
			ApproversGroups: approversGroupRefs(d.GroupIDs),
		}); err != nil {
			return fmt.Errorf("creating association approver %q: %w", d.Name, withOverlapHint(err))
		}
	}

	// 4. PATCH the configured settings fields.
	patch := britive.AccessBuilderSettingsPatch{}
	if isSet(config.ApprovalTimeout) {
		v := config.ApprovalTimeout.ValueString()
		patch.ApprovalTimeOut = &v
	}
	if !config.ProfileExpirationTimeout.IsNull() && !config.ProfileExpirationTimeout.IsUnknown() {
		v := config.ProfileExpirationTimeout.ValueInt64()
		patch.ProfileExpirationTimeout = &v
	}
	if manageMediums {
		patch.NotificationMediums = &mediums
	}
	if manageRules {
		patch.MemberRules = &memberRules
	}
	if !config.ManagerApprovalRequired.IsNull() || !config.ManagerApprovalCondition.IsNull() {
		managerApproval := britive.AccessRequestManagerApproval{Required: false, Condition: "Any"}
		if current.ManagerApproval != nil {
			managerApproval = *current.ManagerApproval
		}
		if !config.ManagerApprovalRequired.IsNull() {
			managerApproval.Required = config.ManagerApprovalRequired.ValueBool()
		}
		if !config.ManagerApprovalCondition.IsNull() {
			managerApproval.Condition = config.ManagerApprovalCondition.ValueString()
		}
		patch.ManagerApproval = &managerApproval
	}
	if patch != (britive.AccessBuilderSettingsPatch{}) {
		if err := r.client.PatchAccessBuilderSettings(applicationID, patch); err != nil {
			return fmt.Errorf("updating access builder settings: %w", err)
		}
	}

	// 5. Enable access requests.
	if targetAllow && !enabled {
		allowed := true
		if err := r.client.PatchAccessBuilderSettings(applicationID, britive.AccessBuilderSettingsPatch{AllowAccessRequest: &allowed}); err != nil {
			return fmt.Errorf("enabling access requests: %w", err)
		}
	}

	return nil
}

// withOverlapHint adds guidance to the API's environment overlap errors.
func withOverlapHint(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "combination already exist") || strings.Contains(msg, "already exists for") {
		return fmt.Errorf("%w\n\nAn environment or environment group can belong to only one association approver, including parent and child groups. Add the approvers group to the association_approver that already covers it instead", err)
	}
	return err
}

// resolveNotificationMediums converts notification_medium blocks to API references, resolving
// names to IDs with a single list call.
func (r *ApplicationAccessBuilderSettingsResource) resolveNotificationMediums(ctx context.Context, set types.Set, diags *diag.Diagnostics) ([]britive.AccessRequestNotificationMedium, error) {
	mediums := notificationMediumsFromSet(ctx, set, diags)
	resolved := make([]britive.AccessRequestNotificationMedium, 0, len(mediums))

	var all []britive.NotificationMedium
	for _, m := range mediums {
		if isSet(m.ID) {
			resolved = append(resolved, britive.AccessRequestNotificationMedium{ID: m.ID.ValueString()})
			continue
		}
		if all == nil {
			var err error
			if all, err = r.client.ListNotificationMediums(); err != nil {
				return nil, err
			}
		}
		found := false
		for _, n := range all {
			if n.Name == m.Name.ValueString() {
				resolved = append(resolved, britive.AccessRequestNotificationMedium{ID: n.ID})
				found = true
				break
			}
		}
		if !found {
			return nil, errs.NewNotFoundErrorf("notification medium %q", m.Name.ValueString())
		}
	}
	return resolved, nil
}

// resolveMemberRules converts member_rule blocks to API member rules, resolving names to IDs.
func (r *ApplicationAccessBuilderSettingsResource) resolveMemberRules(ctx context.Context, set types.Set, diags *diag.Diagnostics) ([]britive.AccessRequestMemberRule, error) {
	rules := memberRulesFromSet(ctx, set, diags)
	resolved := make([]britive.AccessRequestMemberRule, 0, len(rules))
	for _, m := range rules {
		id, err := resolveUserOrTagID(r.client, m.Type.ValueString(), m.ID, m.Name)
		if err != nil {
			return nil, err
		}
		// The API stores the name sent with each member rule (the admin UI always sends it),
		// so send it too, or the rule is returned without a name.
		name, err := resolveUserOrTagName(r.client, m.Type.ValueString(), m.ID, m.Name)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, britive.AccessRequestMemberRule{
			ID:         id,
			Name:       name,
			MemberType: m.Type.ValueString(),
			Condition:  m.Condition.ValueString(),
		})
	}
	return resolved, nil
}

// resolveAssociationApprovers converts association_approver blocks to API IDs. Environments
// and environment groups are matched by name or ID against the application's environment tree,
// fetched once. Every unmatched value is reported together.
func (r *ApplicationAccessBuilderSettingsResource) resolveAssociationApprovers(ctx context.Context, applicationID string, set types.Set, diags *diag.Diagnostics) ([]resolvedAssociationApprover, error) {
	approvers := associationApproversFromSet(ctx, set, diags)
	if len(approvers) == 0 {
		return nil, nil
	}

	tree, err := r.client.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil {
		return nil, err
	}

	resolved := make([]resolvedAssociationApprover, 0, len(approvers))
	unmatched := make([]string, 0)
	for _, a := range approvers {
		ra := resolvedAssociationApprover{
			Name:     a.Name.ValueString(),
			GroupIDs: stringsFromSet(ctx, a.ApproversGroupIDs, diags),
		}
		for _, as := range accessRequestAssociationsFromSet(ctx, a.Associations, diags) {
			assocType := schedulescan.CanonicalCasing(as.Type.ValueString(), "Environment", "EnvironmentGroup")
			value := as.Value.ValueString()

			candidates, code := tree.Environments, associationApproverTypeEnvironment
			if assocType == "EnvironmentGroup" {
				candidates, code = tree.EnvironmentGroups, associationApproverTypeEnvironmentGroup
			}

			found := false
			for _, c := range candidates {
				if c.Name == value || c.ID == value {
					ra.Associations = append(ra.Associations, britive.AssociationApproverAssociation{ID: c.ID, Type: code})
					found = true
					break
				}
			}
			if !found {
				unmatched = append(unmatched, fmt.Sprintf("%s=%s", assocType, value))
			}
		}
		resolved = append(resolved, ra)
	}

	if len(unmatched) > 0 {
		return nil, errs.NewNotFoundErrorf("association %v", unmatched)
	}
	return resolved, nil
}

// mapModelToResource maps the API's settings onto state. A block is refreshed only when it is
// managed - it has elements in state - or when its state is null (import). An empty block in
// state means the block is not configured, so whatever the server has is left untracked.
// References keep the id or name form found in state's prior values; anything not found there
// (import, or changes made outside Terraform) uses the ID form, except associations, which use
// the environment name.
func (r *ApplicationAccessBuilderSettingsResource) mapModelToResource(ctx context.Context, applicationID string, settings *britive.AccessBuilderSettings, state *ApplicationAccessBuilderSettingsResourceModel, diags *diag.Diagnostics) error {
	state.AllowAccessRequest = types.BoolValue(settings.AllowAccessRequest)

	if settings.ApprovalTimeOut != nil {
		state.ApprovalTimeout = types.StringValue(*settings.ApprovalTimeOut)
	} else {
		state.ApprovalTimeout = types.StringNull()
	}
	if settings.ProfileExpirationTimeout.Valid {
		state.ProfileExpirationTimeout = types.Int64Value(settings.ProfileExpirationTimeout.Value)
	} else {
		state.ProfileExpirationTimeout = types.Int64Null()
	}

	if settings.ManagerApproval != nil {
		state.ManagerApprovalRequired = types.BoolValue(settings.ManagerApproval.Required)
		state.ManagerApprovalCondition = types.StringValue(settings.ManagerApproval.Condition)
	} else {
		state.ManagerApprovalRequired = types.BoolValue(false)
		state.ManagerApprovalCondition = types.StringValue("Any")
	}

	// Notification mediums.
	if trackBlock(state.NotificationMediums) {
		r.mapNotificationMediums(ctx, settings, state, diags)
	}

	// Member rules.
	if trackBlock(state.MemberRules) {
		r.mapMemberRules(ctx, settings, state, diags)
	}

	// Association approvers.
	if trackBlock(state.AssociationApprovers) {
		return r.mapAssociationApprovers(ctx, applicationID, settings, state, diags)
	}
	return nil
}

// trackBlock reports whether Read should refresh a block: it is managed (has elements), or its
// state is null (import).
func trackBlock(set types.Set) bool {
	return set.IsNull() || hasElements(set)
}

func (r *ApplicationAccessBuilderSettingsResource) mapNotificationMediums(ctx context.Context, settings *britive.AccessBuilderSettings, state *ApplicationAccessBuilderSettingsResourceModel, diags *diag.Diagnostics) {
	priorMediums := notificationMediumsFromSet(ctx, state.NotificationMediums, diags)
	mediumRefs := make([]accessRequestRef, 0, len(priorMediums))
	for _, p := range priorMediums {
		mediumRefs = append(mediumRefs, accessRequestRef{Type: "NotificationMedium", ID: p.ID, Name: p.Name})
	}
	mediums := make([]notificationMediumModel, 0, len(settings.NotificationMediums))
	for _, n := range settings.NotificationMediums {
		id, name := accessRequestRefForm("NotificationMedium", n.ID, n.Name, mediumRefs, r.notificationMediumResolver())
		mediums = append(mediums, notificationMediumModel{ID: id, Name: name})
	}
	state.NotificationMediums = notificationMediumsToSet(ctx, mediums, diags)
}

// notificationMediumResolver returns a resolver that looks up notification medium IDs by name,
// listing the mediums at most once.
func (r *ApplicationAccessBuilderSettingsResource) notificationMediumResolver() accessRequestNameResolver {
	var all []britive.NotificationMedium
	return func(_, name string) (string, error) {
		if all == nil {
			var err error
			if all, err = r.client.ListNotificationMediums(); err != nil {
				return "", err
			}
		}
		for _, n := range all {
			if n.Name == name {
				return n.ID, nil
			}
		}
		return "", errs.NewNotFoundErrorf("notification medium %q", name)
	}
}

func (r *ApplicationAccessBuilderSettingsResource) mapMemberRules(ctx context.Context, settings *britive.AccessBuilderSettings, state *ApplicationAccessBuilderSettingsResourceModel, diags *diag.Diagnostics) {
	resolve := cachedUserOrTagResolver(r.client)
	priorRules := memberRulesFromSet(ctx, state.MemberRules, diags)
	ruleRefs := make([]accessRequestRef, 0, len(priorRules))
	for _, p := range priorRules {
		ruleRefs = append(ruleRefs, accessRequestRef{Type: p.Type.ValueString(), ID: p.ID, Name: p.Name})
	}
	rules := make([]memberRuleModel, 0, len(settings.MemberRules))
	for _, m := range settings.MemberRules {
		id, name := accessRequestRefForm(m.MemberType, m.ID, m.Name, ruleRefs, resolve)
		rules = append(rules, memberRuleModel{
			Type:      types.StringValue(m.MemberType),
			Condition: types.StringValue(m.Condition),
			ID:        id,
			Name:      name,
		})
	}
	state.MemberRules = memberRulesToSet(ctx, rules, diags)
}

func (r *ApplicationAccessBuilderSettingsResource) mapAssociationApprovers(ctx context.Context, applicationID string, settings *britive.AccessBuilderSettings, state *ApplicationAccessBuilderSettingsResourceModel, diags *diag.Diagnostics) error {
	priorApprovers := associationApproversFromSet(ctx, state.AssociationApprovers, diags)
	priorByName := make(map[string][]accessRequestAssociationModel, len(priorApprovers))
	for _, p := range priorApprovers {
		priorByName[p.Name.ValueString()] = accessRequestAssociationsFromSet(ctx, p.Associations, diags)
	}

	var tree *britive.ApplicationRootEnvironmentGroup
	if len(settings.AssociationApprovers) > 0 {
		var err error
		if tree, err = r.client.GetApplicationRootEnvironmentGroup(applicationID); err != nil {
			return err
		}
	}

	approvers := make([]associationApproverModel, 0, len(settings.AssociationApprovers))
	for _, a := range settings.AssociationApprovers {
		prior := priorByName[a.Name]
		assocs := make([]accessRequestAssociationModel, 0, len(a.Associations))
		for _, as := range a.Associations {
			assocType := "Environment"
			candidates := tree.Environments
			if as.Type == associationApproverTypeEnvironmentGroup {
				assocType = "EnvironmentGroup"
				candidates = tree.EnvironmentGroups
			}

			value := as.ID
			for _, c := range candidates {
				if c.ID == as.ID {
					value = c.Name
					break
				}
			}
			for _, p := range prior {
				if strings.EqualFold(p.Type.ValueString(), assocType) && p.Value.ValueString() == as.ID {
					value = as.ID
					break
				}
			}

			typeValue := types.StringValue(assocType)
			for _, p := range prior {
				if strings.EqualFold(p.Type.ValueString(), assocType) {
					// Keep the practitioner's casing of the case-insensitive type.
					typeValue = p.Type
					break
				}
			}
			assocs = append(assocs, accessRequestAssociationModel{Type: typeValue, Value: types.StringValue(value)})
		}

		groupIDs, d := types.SetValueFrom(ctx, types.StringType, approversGroupIDs(a.ApproversGroups))
		diags.Append(d...)
		approvers = append(approvers, associationApproverModel{
			Name:              types.StringValue(a.Name),
			Associations:      accessRequestAssociationsToSet(ctx, assocs, diags),
			ApproversGroupIDs: groupIDs,
		})
	}
	state.AssociationApprovers = associationApproversToSet(ctx, approvers, diags)

	return nil
}

// parseApprovalTimeoutMinutes parses a DD:HH:MM approval timeout into minutes and checks the
// API's 5-minute to 5-day range.
func parseApprovalTimeoutMinutes(value string) (int, error) {
	if !approvalTimeoutPattern.MatchString(value) {
		return 0, fmt.Errorf("approval_timeout %q must be in DD:HH:MM format", value)
	}
	parts := strings.Split(value, ":")
	days, _ := strconv.Atoi(parts[0])
	hours, _ := strconv.Atoi(parts[1])
	minutes, _ := strconv.Atoi(parts[2])
	if hours > 23 || minutes > 59 {
		return 0, fmt.Errorf("approval_timeout %q has hours above 23 or minutes above 59", value)
	}
	total := days*24*60 + hours*60 + minutes
	if total < approvalTimeoutMinMinutes || total > approvalTimeoutMaxMinutes {
		return 0, fmt.Errorf("approval_timeout %q must be from 00:00:05 (5 minutes) to 05:00:00 (5 days)", value)
	}
	return total, nil
}

// associationKeys returns sorted "type:id" keys for comparing association lists.
func associationKeys(assocs []britive.AssociationApproverAssociation) []string {
	keys := make([]string, 0, len(assocs))
	for _, a := range assocs {
		keys = append(keys, fmt.Sprintf("%d:%s", a.Type, a.ID))
	}
	sort.Strings(keys)
	return keys
}

// approversGroupIDs returns the sorted IDs of groups.
func approversGroupIDs(groups []britive.ApproversGroup) []string {
	ids := make([]string, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	sort.Strings(ids)
	return ids
}

// approversGroupRefs converts group IDs to the API's {id} references.
func approversGroupRefs(ids []string) []britive.ApproversGroup {
	refs := make([]britive.ApproversGroup, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, britive.ApproversGroup{ID: id})
	}
	return refs
}

func sortedCopy(v []string) []string {
	c := append([]string(nil), v...)
	sort.Strings(c)
	return c
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isSubset reports whether every element of a is in b.
func isSubset(a, b []string) bool {
	in := make(map[string]bool, len(b))
	for _, v := range b {
		in[v] = true
	}
	for _, v := range a {
		if !in[v] {
			return false
		}
	}
	return true
}

// accessBuilderSettingsCompositeID builds the resource's `id` value.
func accessBuilderSettingsCompositeID(applicationID string) string {
	return fmt.Sprintf("apps/%s/access-builder-settings", applicationID)
}

// parseAccessBuilderSettingsCompositeID extracts applicationID from either the composite ID
// ("apps/{application_id}/access-builder-settings") or a bare application ID.
func parseAccessBuilderSettingsCompositeID(id string) (string, error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 3:
		if parts[0] != "apps" || parts[2] != "access-builder-settings" || parts[1] == "" {
			return "", errs.NewInvalidResourceIDError("access builder settings", id)
		}
		return parts[1], nil
	case 1:
		if parts[0] == "" {
			return "", errs.NewInvalidResourceIDError("access builder settings", id)
		}
		return parts[0], nil
	default:
		return "", errs.NewInvalidResourceIDError("access builder settings", id)
	}
}
