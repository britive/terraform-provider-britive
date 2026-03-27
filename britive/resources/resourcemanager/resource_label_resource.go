package resourcemanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceLabelResource struct {
	client *britive.Client
}

type ResourceLabelResourceModel struct {
	ID          types.String              `tfsdk:"id"`
	Name        types.String              `tfsdk:"name"`
	Description types.String              `tfsdk:"description"`
	Internal    types.Bool                `tfsdk:"internal"`
	LabelColor  types.String              `tfsdk:"label_color"`
	Values      []ResourceLabelValueModel `tfsdk:"values"`
}

type ResourceLabelValueModel struct {
	ValueID     types.String `tfsdk:"value_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewResourceLabelResource() resource.Resource {
	return &ResourceLabelResource{}
}

func (r *ResourceLabelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_label"
}

func (r *ResourceLabelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a Britive resource manager resource label",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validators.Alphanumeric(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"internal": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"label_color": schema.StringAttribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"values": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value_id": schema.StringAttribute{
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (r *ResourceLabelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	label := britive.ResourceLabel{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		LabelColor:  plan.LabelColor.ValueString(),
		Values:      make([]britive.ResourceLabelValue, 0),
	}

	for _, v := range plan.Values {
		label.Values = append(label.Values, britive.ResourceLabelValue{
			Name:        v.Name.ValueString(),
			Description: v.Description.ValueString(),
		})
	}

	log.Printf("[INFO] Creating resource label: %#v", label)

	created, err := r.client.CreateUpdateResourceLabel(label, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Label", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/labels/%s", created.LabelId))
	plan.Internal = types.BoolValue(created.Internal)

	var values []ResourceLabelValueModel
	for _, v := range created.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	plan.Values = values

	log.Printf("[INFO] Created resource label: %s", created.LabelId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := parseLabelID(state.ID.ValueString())
	label, err := r.client.GetResourceLabel(labelID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Label", err.Error())
		return
	}

	state.Name = types.StringValue(label.Name)
	state.Description = types.StringValue(label.Description)
	state.Internal = types.BoolValue(label.Internal)
	state.LabelColor = types.StringValue(label.LabelColor)

	var values []ResourceLabelValueModel
	for _, v := range label.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	state.Values = values

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceLabelResourceModel
	var state ResourceLabelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := parseLabelID(state.ID.ValueString())

	label := britive.ResourceLabel{
		LabelId:     labelID,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		LabelColor:  plan.LabelColor.ValueString(),
		Values:      make([]britive.ResourceLabelValue, 0),
	}

	for _, v := range plan.Values {
		labelValue := britive.ResourceLabelValue{
			Name:        v.Name.ValueString(),
			Description: v.Description.ValueString(),
		}
		if !v.ValueID.IsNull() {
			labelValue.ValueId = v.ValueID.ValueString()
		}
		label.Values = append(label.Values, labelValue)
	}

	log.Printf("[INFO] Updating resource label: %s", labelID)

	updated, err := r.client.CreateUpdateResourceLabel(label, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource Label", err.Error())
		return
	}

	plan.Internal = types.BoolValue(updated.Internal)

	var values []ResourceLabelValueModel
	for _, v := range updated.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	plan.Values = values

	log.Printf("[INFO] Updated resource label: %s", labelID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := parseLabelID(state.ID.ValueString())

	log.Printf("[INFO] Deleting resource label: %s", labelID)

	err := r.client.DeleteResourceLabel(labelID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Label", err.Error())
		return
	}

	log.Printf("[INFO] Deleted resource label: %s", labelID)
}

func (r *ResourceLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var labelID string

	if strings.HasPrefix(importID, "resource-manager/labels/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/labels/{id}' or '{id}', got: %s", importID))
			return
		}
		labelID = parts[2]
	} else {
		labelID = importID
	}

	label, err := r.client.GetResourceLabel(labelID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Label Not Found", fmt.Sprintf("Label %s not found", labelID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Label", err.Error())
		return
	}

	var state ResourceLabelResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/labels/%s", label.LabelId))
	state.Name = types.StringValue(label.Name)
	state.Description = types.StringValue(label.Description)
	state.Internal = types.BoolValue(label.Internal)
	state.LabelColor = types.StringValue(label.LabelColor)

	var values []ResourceLabelValueModel
	for _, v := range label.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	state.Values = values

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func parseLabelID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return id
}

// ModifyPlan copies value_id from state to plan by matching on `name`, bypassing the set element
// hashing problem: UseStateForUnknown can't match elements when value_id (computed) is unknown.
func (r *ResourceLabelResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip create and delete
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan ResourceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build name → state value map
	type stateVal struct {
		valueID     string
		description types.String
	}
	stateValues := make(map[string]stateVal)
	for _, sv := range state.Values {
		if !sv.Name.IsNull() && !sv.Name.IsUnknown() {
			stateValues[sv.Name.ValueString()] = stateVal{
				valueID:     sv.ValueID.ValueString(),
				description: sv.Description,
			}
		}
	}

	// Fix plan values by matching on name:
	// 1. Always copy value_id from state (it's purely Computed; UseStateForUnknown can assign
	//    the wrong ID when set element hashes mismatch due to description="" vs null).
	// 2. When plan description is null and state description is "" (SDKv2 empty string),
	//    treat them as equivalent to prevent spurious set element churn.
	modified := false
	for i, pv := range plan.Values {
		if pv.Name.IsUnknown() || pv.Name.IsNull() {
			continue
		}
		sv, ok := stateValues[pv.Name.ValueString()]
		if !ok {
			continue
		}
		if pv.ValueID.ValueString() != sv.valueID {
			plan.Values[i].ValueID = types.StringValue(sv.valueID)
			modified = true
		}
		// Normalize: null in plan vs "" in state are equivalent for Optional description
		if plan.Values[i].Description.IsNull() && !sv.description.IsNull() && sv.description.ValueString() == "" {
			plan.Values[i].Description = sv.description
			modified = true
		}
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// UpgradeState handles migration from SDKv2 state (version 0) to Plugin Framework state (version 1).
// SDKv2 stored extra fields in each `values` set element: created_by (int), updated_by (int),
// created_on (string), updated_on (string). The Framework's automatic upgrade cannot handle the
// TypeInt → missing field mapping, causing set element matching to scramble. This upgrader
// explicitly drops those extra fields.
func (r *ResourceLabelResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// Version 0 = SDKv2 state format
		0: {
			PriorSchema: nil, // use raw JSON upgrade
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Parse raw JSON from SDKv2 state
				var rawState map[string]json.RawMessage
				if err := json.Unmarshal(req.RawState.JSON, &rawState); err != nil {
					resp.Diagnostics.AddError("Error upgrading resource label state", fmt.Sprintf("Could not parse raw state JSON: %s", err))
					return
				}

				// Parse the values array — each element may have extra SDKv2 fields
				type sdkV2LabelValue struct {
					ValueID     string  `json:"value_id"`
					Name        string  `json:"name"`
					Description string  `json:"description"`
					// Extra SDKv2 fields we drop
					CreatedBy  *float64 `json:"created_by,omitempty"`
					UpdatedBy  *float64 `json:"updated_by,omitempty"`
					CreatedOn  *string  `json:"created_on,omitempty"`
					UpdatedOn  *string  `json:"updated_on,omitempty"`
				}

				var sdkValues []sdkV2LabelValue
				if rawValues, ok := rawState["values"]; ok {
					if err := json.Unmarshal(rawValues, &sdkValues); err != nil {
						resp.Diagnostics.AddError("Error upgrading resource label state", fmt.Sprintf("Could not parse values: %s", err))
						return
					}
				}

				// Helper to extract string field from raw state
				getString := func(key string) string {
					if raw, ok := rawState[key]; ok {
						var s string
						if err := json.Unmarshal(raw, &s); err == nil {
							return s
						}
					}
					return ""
				}
				getBool := func(key string) bool {
					if raw, ok := rawState[key]; ok {
						var b bool
						if err := json.Unmarshal(raw, &b); err == nil {
							return b
						}
					}
					return false
				}

				// Build new state with only the fields the Framework schema expects
				newValues := make([]ResourceLabelValueModel, 0, len(sdkValues))
				for _, v := range sdkValues {
					newValues = append(newValues, ResourceLabelValueModel{
						ValueID:     types.StringValue(v.ValueID),
						Name:        types.StringValue(v.Name),
						Description: types.StringValue(v.Description),
					})
				}

				newState := ResourceLabelResourceModel{
					ID:          types.StringValue(getString("id")),
					Name:        types.StringValue(getString("name")),
					Description: types.StringValue(getString("description")),
					Internal:    types.BoolValue(getBool("internal")),
					LabelColor:  types.StringValue(getString("label_color")),
					Values:      newValues,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
			},
		},
	}
}
