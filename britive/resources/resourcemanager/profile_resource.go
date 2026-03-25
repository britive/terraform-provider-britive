package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProfileResource struct {
	client *britive.Client
}

type ProfileResourceModel struct {
	ID                             types.String       `tfsdk:"id"`
	Name                           types.String       `tfsdk:"name"`
	Description                    types.String       `tfsdk:"description"`
	ExpirationDuration             types.Int64        `tfsdk:"expiration_duration"`
	Status                         types.String       `tfsdk:"status"`
	AllowImpersonation             types.Bool         `tfsdk:"allow_impersonation"`
	Extendable                     types.Bool         `tfsdk:"extendable"`
	NotificationPriorToExpiration  types.String       `tfsdk:"notification_prior_to_expiration"`
	ExtensionDuration              types.String       `tfsdk:"extension_duration"`
	ExtensionLimit                 types.Int64        `tfsdk:"extension_limit"`
	Associations                   []AssociationModel `tfsdk:"associations"`
	ResourceLabelColorMap          types.Set          `tfsdk:"resource_label_color_map"`
}

type AssociationModel struct {
	LabelKey types.String `tfsdk:"label_key"`
	Values   types.Set    `tfsdk:"values"`
}

type ResourceLabelColorMapModel struct {
	LabelKey  types.String `tfsdk:"label_key"`
	ColorCode types.String `tfsdk:"color_code"`
}

func NewProfileResource() resource.Resource {
	return &ProfileResource{}
}

func (r *ProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_profile"
}

func (r *ProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager profile",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the britive resource manager profile",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of britive resource manager profile",
			},
			"expiration_duration": schema.Int64Attribute{
				Required:    true,
				Description: "Expiration duration of resource manager profile in milliseconds",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of resource manager profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_impersonation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Enable or disable delegation",
			},
			"extendable": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether profile expiry is extendable",
			},
			"notification_prior_to_expiration": schema.StringAttribute{
				Optional:    true,
				Description: "Duration before expiry at which to send a notification (e.g. '1h0m0s'). Required when extendable is true.",
				Validators:  []validator.String{validators.Duration()},
			},
			"extension_duration": schema.StringAttribute{
				Optional:    true,
				Description: "Duration by which the profile expiry can be extended (e.g. '2h0m0s'). Required when extendable is true.",
				Validators:  []validator.String{validators.Duration()},
			},
			"extension_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of times the profile expiry can be extended.",
			},
			"resource_label_color_map": schema.SetNestedAttribute{
				Computed:    true,
				Description: "Resource label color map",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label_key": schema.StringAttribute{
							Computed:    true,
							Description: "name of the resource label",
						},
						"color_code": schema.StringAttribute{
							Computed:    true,
							Description: "color code of resource label",
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"associations": schema.SetNestedBlock{
				Description: "Resource manager profile associations",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"label_key": schema.StringAttribute{
							Required:    true,
							Description: "Resource label name for association",
						},
						"values": schema.SetAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "Values of resource label for association",
						},
					},
				},
			},
		},
	}
}

func (r *ProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *ProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceManagerProfile, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}

	log.Printf("[INFO] Creating resource_manager_profile Resource : %#v", resourceManagerProfile)

	created, err := r.client.CreateUpdateResourceManagerProfile(resourceManagerProfile, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Manager Profile", err.Error())
		return
	}

	log.Printf("[INFO] Created Resource_Manager_Profile Resource : %#v", created)

	profileID := created.ProfileId

	log.Printf("[INFO] Adding associations to resource_manager_profile")
	resourceManagerProfile.ProfileId = profileID
	_, err = r.client.CreateUpdateResourceManagerProfileAssociations(resourceManagerProfile)
	if err != nil {
		resp.Diagnostics.AddError("Error Adding Associations", err.Error())
		// Rollback: delete the profile if associations fail
		log.Printf("[WARN] Rolling back profile creation due to error")
		delErr := r.client.DeleteResourceManagerProfile(profileID)
		if delErr != nil {
			log.Printf("[ERROR] Failed to delete profile during rollback: %v", delErr)
			resp.Diagnostics.AddError("Error Rolling Back", delErr.Error())
		} else {
			log.Printf("[INFO] Rolled back profile creation")
		}
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/profile/%s", profileID))

	log.Printf("[INFO] Added Associations to Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	// Read back to get computed values
	readResp, err := r.client.GetResourceManagerProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile", err.Error())
		return
	}

	associations, err := r.client.GetResourceManagerProfileAssociations(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Associations", err.Error())
		return
	}

	readResp.Associations = associations.Associations
	readResp.ResourceLabelColorMap = associations.ResourceLabelColorMap

	r.mapModelToResource(ctx, readResp, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Reading Resource_Manager_Profile Resource of ID : %s", profileID)

	resourceManagerProfile, err := r.client.GetResourceManagerProfile(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile", err.Error())
		return
	}

	log.Printf("[INFO] Reading Associations from Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	associations, err := r.client.GetResourceManagerProfileAssociations(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Associations", err.Error())
		return
	}

	resourceManagerProfile.Associations = associations.Associations
	resourceManagerProfile.ResourceLabelColorMap = associations.ResourceLabelColorMap

	r.mapModelToResource(ctx, resourceManagerProfile, &state, &resp.Diagnostics)

	log.Printf("[INFO] Found Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfileResourceModel
	var state ProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	resourceManagerProfile, err := r.mapResourceToModel(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Mapping Resource", err.Error())
		return
	}
	resourceManagerProfile.ProfileId = profileID

	log.Printf("[INFO] Updating Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	_, err = r.client.CreateUpdateResourceManagerProfile(resourceManagerProfile, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource Manager Profile", err.Error())
		return
	}

	log.Printf("[INFO] Updating associations to resource_manager_profile")

	_, err = r.client.CreateUpdateResourceManagerProfileAssociations(resourceManagerProfile)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Associations", err.Error())
		return
	}

	log.Printf("[INFO] Updated Resource_Manager_Profile Resource")

	// Read back to get updated values
	readResp, err := r.client.GetResourceManagerProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Manager Profile", err.Error())
		return
	}

	associations, err := r.client.GetResourceManagerProfileAssociations(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Associations", err.Error())
		return
	}

	readResp.Associations = associations.Associations
	readResp.ResourceLabelColorMap = associations.ResourceLabelColorMap

	r.mapModelToResource(ctx, readResp, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting Resource_manager_Profile Resource of ID : %s", profileID)

	err = r.client.DeleteResourceManagerProfile(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Manager Profile", err.Error())
		return
	}

	log.Printf("[INFO] Deleted Resource_Manager_Profile Resource of ID : %s", profileID)
}

func (r *ProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID

	if !strings.HasPrefix(importID, "resource-manager/profile/") {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'resource-manager/profile/{id}', got: %s", importID),
		)
		return
	}

	parts := strings.Split(importID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'resource-manager/profile/{id}', got: %s", importID),
		)
		return
	}

	profileID := parts[2]
	if strings.TrimSpace(profileID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Profile ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing Resource_Manager_Profile Resource : %s", profileID)

	resourceManagerProfile, err := r.client.GetResourceManagerProfile(profileID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Manager Profile Not Found", fmt.Sprintf("Resource manager profile %s not found", profileID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Manager Profile", err.Error())
		return
	}

	associations, err := r.client.GetResourceManagerProfileAssociations(profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Associations", err.Error())
		return
	}

	resourceManagerProfile.Associations = associations.Associations
	resourceManagerProfile.ResourceLabelColorMap = associations.ResourceLabelColorMap

	var state ProfileResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/profile/%s", resourceManagerProfile.ProfileId))

	r.mapModelToResource(ctx, resourceManagerProfile, &state, &resp.Diagnostics)

	log.Printf("[INFO] Imported Resource_Manager_Profile Resource : %s", profileID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ProfileResource) mapResourceToModel(ctx context.Context, plan *ProfileResourceModel) (britive.ResourceManagerProfile, error) {
	profile := britive.ResourceManagerProfile{
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		ExpirationDuration: int(plan.ExpirationDuration.ValueInt64()),
		DelegationEnabled:  plan.AllowImpersonation.ValueBool(),
		Extendable:         plan.Extendable.ValueBool(),
		Associations:       make(map[string][]string),
	}

	if profile.Extendable {
		if plan.NotificationPriorToExpiration.IsNull() || plan.NotificationPriorToExpiration.ValueString() == "" {
			return profile, fmt.Errorf("notification_prior_to_expiration is required when extendable is true")
		}
		d, err := time.ParseDuration(plan.NotificationPriorToExpiration.ValueString())
		if err != nil {
			return profile, fmt.Errorf("invalid notification_prior_to_expiration: %w", err)
		}
		ms := int64(d / time.Millisecond)
		profile.NotificationPriorToExpiration = &ms

		if plan.ExtensionDuration.IsNull() || plan.ExtensionDuration.ValueString() == "" {
			return profile, fmt.Errorf("extension_duration is required when extendable is true")
		}
		d2, err := time.ParseDuration(plan.ExtensionDuration.ValueString())
		if err != nil {
			return profile, fmt.Errorf("invalid extension_duration: %w", err)
		}
		ms2 := int64(d2 / time.Millisecond)
		profile.ExtensionDuration = &ms2

		profile.ExtensionLimit = int(plan.ExtensionLimit.ValueInt64())
	}

	for _, assoc := range plan.Associations {
		var values []string
		diagsVals := assoc.Values.ElementsAs(ctx, &values, false)
		if diagsVals.HasError() {
			return profile, fmt.Errorf("error parsing association values")
		}
		profile.Associations[assoc.LabelKey.ValueString()] = values
	}

	return profile, nil
}

func (r *ProfileResource) mapModelToResource(ctx context.Context, profile *britive.ResourceManagerProfile, state *ProfileResourceModel, diags *diag.Diagnostics) {
	state.Name = types.StringValue(profile.Name)
	state.Description = types.StringValue(profile.Description)
	state.ExpirationDuration = types.Int64Value(int64(profile.ExpirationDuration))
	state.Status = types.StringValue(profile.Status)
	state.AllowImpersonation = types.BoolValue(profile.DelegationEnabled)
	state.Extendable = types.BoolValue(profile.Extendable)

	if profile.Extendable {
		if profile.NotificationPriorToExpiration != nil {
			state.NotificationPriorToExpiration = types.StringValue(
				time.Duration(*profile.NotificationPriorToExpiration * int64(time.Millisecond)).String(),
			)
		}
		if profile.ExtensionDuration != nil {
			state.ExtensionDuration = types.StringValue(
				time.Duration(*profile.ExtensionDuration * int64(time.Millisecond)).String(),
			)
		}
		if profile.ExtensionLimit != nil {
			if limit, ok := profile.ExtensionLimit.(float64); ok {
				state.ExtensionLimit = types.Int64Value(int64(limit))
			} else if limit, ok := profile.ExtensionLimit.(int); ok {
				state.ExtensionLimit = types.Int64Value(int64(limit))
			}
		}
	} else {
		if state.NotificationPriorToExpiration.IsUnknown() {
			state.NotificationPriorToExpiration = types.StringNull()
		}
		if state.ExtensionDuration.IsUnknown() {
			state.ExtensionDuration = types.StringNull()
		}
		if state.ExtensionLimit.IsUnknown() {
			state.ExtensionLimit = types.Int64Null()
		}
	}

	// Map associations directly as a slice (SetNestedBlock)
	var associationsList []AssociationModel
	for labelKey, values := range profile.Associations {
		valuesSet, diagsSet := types.SetValueFrom(ctx, types.StringType, values)
		if diagsSet.HasError() {
			diags.Append(diagsSet...)
			continue
		}
		associationsList = append(associationsList, AssociationModel{
			LabelKey: types.StringValue(labelKey),
			Values:   valuesSet,
		})
	}
	state.Associations = associationsList

	// Map resource label color map
	var colorMapList []ResourceLabelColorMapModel
	for labelKey, colorCode := range profile.ResourceLabelColorMap {
		colorMapList = append(colorMapList, ResourceLabelColorMapModel{
			LabelKey:  types.StringValue(labelKey),
			ColorCode: types.StringValue(colorCode),
		})
	}

	colorMapSet, diagsColor := types.SetValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"label_key":  types.StringType,
			"color_code": types.StringType,
		},
	}, colorMapList)
	if diagsColor.HasError() {
		diags.Append(diagsColor...)
	} else {
		state.ResourceLabelColorMap = colorMapSet
	}
}

func (r *ProfileResource) parseUniqueID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid resource ID format: %s", id)
	}
	return parts[2], nil
}
