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
		Version:     2,
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
				Computed: true,
			},
			"internal": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"label_color": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"values": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value_id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
							Computed: true,
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

	plan.Values = buildValuesInOrder(created.Values, plan.Values)

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
	state.Description = preserveOptionalString(label.Description, state.Description)
	state.Internal = types.BoolValue(label.Internal)
	state.LabelColor = preserveOptionalString(label.LabelColor, state.LabelColor)

	// Reorder API values to match prior state order to avoid spurious list-ordering diffs.
	state.Values = buildValuesInOrder(label.Values, state.Values)

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
		if !v.ValueID.IsNull() && !v.ValueID.IsUnknown() {
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

	plan.Values = buildValuesInOrder(updated.Values, plan.Values)

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
	state.Description = optionalStringValue(label.Description)
	state.Internal = types.BoolValue(label.Internal)
	state.LabelColor = optionalStringValue(label.LabelColor)

	var values []ResourceLabelValueModel
	for _, v := range label.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: optionalStringValue(v.Description),
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

// ModifyPlan normalizes Optional+Computed fields (null/empty config → null plan) and copies
// value_id from state by name-matching to handle reordering and additions safely.
func (r *ResourceLabelResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ResourceLabelResourceModel
	var config ResourceLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modified := false

	// Normalize top-level description and label_color: null config → null plan.
	// Needed because Optional+Computed carries prior state when config is null; this resets it.
	// When config is explicitly "" (user wants to clear the field), keep plan as "" — do NOT
	// normalize to null, as that would violate the framework rule that planned values must match
	// non-null config values.
	if config.Description.IsNull() {
		if !plan.Description.IsNull() {
			plan.Description = types.StringNull()
			modified = true
		}
	}
	if config.LabelColor.IsNull() {
		if !plan.LabelColor.IsNull() {
			plan.LabelColor = types.StringNull()
			modified = true
		}
	}

	// Normalize value descriptions: null config → null plan.
	for i, pv := range plan.Values {
		if i >= len(config.Values) {
			break
		}
		cv := config.Values[i]
		if cv.Description.IsNull() {
			if !pv.Description.IsNull() {
				plan.Values[i].Description = types.StringNull()
				modified = true
			}
		}
	}

	// On Update (state exists): copy value_id from state by name to correctly handle
	// additions, removals, and reordering (index-based UseStateForUnknown is insufficient).
	if !req.State.Raw.IsNull() {
		var state ResourceLabelResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}

		stateValueMap := make(map[string]types.String)
		for _, sv := range state.Values {
			if !sv.Name.IsNull() && !sv.Name.IsUnknown() {
				stateValueMap[sv.Name.ValueString()] = sv.ValueID
			}
		}

		for i, pv := range plan.Values {
			if pv.Name.IsUnknown() || pv.Name.IsNull() {
				continue
			}
			stateID, ok := stateValueMap[pv.Name.ValueString()]
			if !ok || stateID.IsNull() || stateID.IsUnknown() {
				continue
			}
			if pv.ValueID.IsUnknown() || pv.ValueID.ValueString() != stateID.ValueString() {
				plan.Values[i].ValueID = stateID
				modified = true
			}
		}
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// UpgradeState handles state migrations:
//   - v0 (SDKv2): had extra fields per value (created_by, updated_by, etc.) and values as a set.
//   - v1 (Framework): values as SetNestedBlock, description/label_color Optional only.
//   - v2 (current): values as ListNestedBlock, description/label_color/value-description Optional+Computed.
func (r *ResourceLabelResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// rawLabelUpgrader parses the raw JSON and builds a ResourceLabelResourceModel.
	// Used by both v0→v2 and v1→v2 paths since the JSON wire format is compatible
	// (both SetNestedBlock and ListNestedBlock are stored as JSON arrays).
	rawLabelUpgrader := func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
		var rawState map[string]json.RawMessage
		if err := json.Unmarshal(req.RawState.JSON, &rawState); err != nil {
			resp.Diagnostics.AddError("Error upgrading resource label state", fmt.Sprintf("Could not parse raw state JSON: %s", err))
			return
		}

		type rawValue struct {
			ValueID     string   `json:"value_id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			CreatedBy   *float64 `json:"created_by,omitempty"`
			UpdatedBy   *float64 `json:"updated_by,omitempty"`
			CreatedOn   *string  `json:"created_on,omitempty"`
			UpdatedOn   *string  `json:"updated_on,omitempty"`
		}

		var rawValues []rawValue
		if rawVals, ok := rawState["values"]; ok {
			if err := json.Unmarshal(rawVals, &rawValues); err != nil {
				resp.Diagnostics.AddError("Error upgrading resource label state", fmt.Sprintf("Could not parse values: %s", err))
				return
			}
		}

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

		newValues := make([]ResourceLabelValueModel, 0, len(rawValues))
		for _, v := range rawValues {
			newValues = append(newValues, ResourceLabelValueModel{
				ValueID:     types.StringValue(v.ValueID),
				Name:        types.StringValue(v.Name),
				Description: optionalStringValue(v.Description),
			})
		}

		newState := ResourceLabelResourceModel{
			ID:          types.StringValue(getString("id")),
			Name:        types.StringValue(getString("name")),
			Description: optionalStringValue(getString("description")),
			Internal:    types.BoolValue(getBool("internal")),
			LabelColor:  optionalStringValue(getString("label_color")),
			Values:      newValues,
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	}

	return map[int64]resource.StateUpgrader{
		// v0: SDKv2 state — values set had extra int fields (created_by, updated_by, etc.)
		0: {StateUpgrader: rawLabelUpgrader},
		// v1: Framework state with SetNestedBlock — JSON array format is identical to ListNestedBlock
		1: {StateUpgrader: rawLabelUpgrader},
	}
}

// optionalStringValue returns types.StringNull() for an empty string, or
// types.StringValue(s) otherwise. Used for Optional-only string fields where
// the API returns "" for an unset value but the plan expects null.
func optionalStringValue(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// preserveOptionalString handles the case where the API returns "" for an
// Optional+Computed string field. When the API value is empty:
//   - if the prior state was null (user never set it), keep null
//   - if the prior state was non-null (user explicitly set "" or a value that was cleared), return ""
//
// This prevents a persistent plan diff when the user has explicitly set the
// field to "" while still correctly returning null when the user never touched it.
func preserveOptionalString(apiValue string, priorState types.String) types.String {
	if apiValue == "" {
		if priorState.IsNull() {
			return types.StringNull()
		}
		return types.StringValue("")
	}
	return types.StringValue(apiValue)
}

// buildValuesInOrder builds a []ResourceLabelValueModel from the API response, ordered to
// match referenceOrder (prior state in Read; plan in Create/Update). This prevents spurious
// list-ordering diffs when the API returns values in a different sequence than the config.
//
//   - Values present in referenceOrder are emitted first, in referenceOrder sequence.
//   - Values returned by the API but absent from referenceOrder (newly added) are appended.
//   - Values in referenceOrder that the API no longer returns (deleted externally) are omitted.
//   - description uses preserveOptionalString so an explicit "" is kept as "" rather than null.
func buildValuesInOrder(apiValues []britive.ResourceLabelValue, referenceOrder []ResourceLabelValueModel) []ResourceLabelValueModel {
	apiByName := make(map[string]britive.ResourceLabelValue, len(apiValues))
	for _, v := range apiValues {
		apiByName[v.Name] = v
	}

	seen := make(map[string]bool, len(referenceOrder))
	result := make([]ResourceLabelValueModel, 0, len(apiValues))

	for _, ref := range referenceOrder {
		name := ref.Name.ValueString()
		v, ok := apiByName[name]
		if !ok {
			continue // value was removed externally; omit from state
		}
		result = append(result, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: preserveOptionalString(v.Description, ref.Description),
		})
		seen[name] = true
	}

	// Append values returned by API that were not in referenceOrder (newly created).
	for _, v := range apiValues {
		if !seen[v.Name] {
			result = append(result, ResourceLabelValueModel{
				ValueID:     types.StringValue(v.ValueId),
				Name:        types.StringValue(v.Name),
				Description: preserveOptionalString(v.Description, types.StringNull()),
			})
		}
	}

	return result
}
