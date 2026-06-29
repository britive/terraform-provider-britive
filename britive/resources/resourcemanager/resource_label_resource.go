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
	CreatedBy   types.Int64  `tfsdk:"created_by"`
	UpdatedBy   types.Int64  `tfsdk:"updated_by"`
	CreatedOn   types.String `tfsdk:"created_on"`
	UpdatedOn   types.String `tfsdk:"updated_on"`
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
						"created_by": schema.Int64Attribute{
							Computed: true,
						},
						"updated_by": schema.Int64Attribute{
							Computed: true,
						},
						"created_on": schema.StringAttribute{
							Computed: true,
						},
						"updated_on": schema.StringAttribute{
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

	// GET after POST: the POST response omits server-computed audit fields.
	// A GET returns the full record so state reflects all computed values.
	createdFull, err := r.client.GetResourceLabel(created.LabelId)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Label After Create", err.Error())
		return
	}
	plan.Values = buildValuesInOrder(createdFull.Values, plan.Values)

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

	// GET after PUT: the PUT response omits server-computed audit fields
	// (updated_by, updated_on). A GET returns the full record so state
	// reflects the real post-update values rather than nulls.
	updatedFull, err := r.client.GetResourceLabel(labelID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Label After Update", err.Error())
		return
	}

	plan.Values = buildValuesInOrder(updatedFull.Values, plan.Values)

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
			CreatedBy:   optionalInt64Value(v.CreatedBy),
			UpdatedBy:   optionalInt64Value(v.UpdatedBy),
			CreatedOn:   optionalStringValue(v.CreatedOn),
			UpdatedOn:   optionalStringValue(v.UpdatedOn),
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

// ModifyPlan normalizes Optional+Computed fields and manages computed audit fields for
// list values. TPF enforces strict plan-vs-actual consistency for Computed attributes in
// ListNestedBlock (unlike SDKv2), so audit fields must be explicitly set to unknown when
// the API will change them, and copied from state by name (not index) otherwise.
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
		if config.Values[i].Description.IsNull() && !pv.Description.IsNull() {
			plan.Values[i].Description = types.StringNull()
			modified = true
		}
	}

	// Build a name-keyed map of the prior state values.
	type stateValueEntry struct {
		valueID     types.String
		createdBy   types.Int64
		updatedBy   types.Int64
		createdOn   types.String
		updatedOn   types.String
		description types.String
	}
	var state ResourceLabelResourceModel
	stateValueMap := make(map[string]stateValueEntry)
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, sv := range state.Values {
			if !sv.Name.IsNull() && !sv.Name.IsUnknown() {
				stateValueMap[sv.Name.ValueString()] = stateValueEntry{
					valueID:     sv.ValueID,
					createdBy:   sv.CreatedBy,
					updatedBy:   sv.UpdatedBy,
					createdOn:   sv.CreatedOn,
					updatedOn:   sv.UpdatedOn,
					description: sv.Description,
				}
			}
		}
	}

	// Detect whether any user-controlled attribute is changing.
	// When true, updated_by/updated_on are set to unknown because the Britive API
	// always stamps them on every write, regardless of which field changed.
	isModification := false
	if !req.State.Raw.IsNull() {
		switch {
		case plan.Name.ValueString() != state.Name.ValueString():
			isModification = true
		case plan.Description.IsNull() != state.Description.IsNull() ||
			plan.Description.ValueString() != state.Description.ValueString():
			isModification = true
		case plan.LabelColor.IsNull() != state.LabelColor.IsNull() ||
			plan.LabelColor.ValueString() != state.LabelColor.ValueString():
			isModification = true
		case len(plan.Values) != len(state.Values):
			isModification = true
		default:
			for _, pv := range plan.Values {
				sv, ok := stateValueMap[pv.Name.ValueString()]
				if !ok {
					isModification = true
					break
				}
				if pv.Description.IsNull() != sv.description.IsNull() ||
					pv.Description.ValueString() != sv.description.ValueString() {
					isModification = true
					break
				}
			}
		}
	}

	// Set computed audit fields for every planned value element.
	// TPF sets new list-element Computed attributes to null (not unknown), so we must
	// explicitly override them here to avoid "was null, but now <value>" apply errors.
	for i, pv := range plan.Values {
		if pv.Name.IsNull() || pv.Name.IsUnknown() {
			continue
		}
		sv, inState := stateValueMap[pv.Name.ValueString()]

		if !inState {
			// New element: the API will set all these; plan them as unknown.
			plan.Values[i].ValueID = types.StringUnknown()
			plan.Values[i].CreatedBy = types.Int64Unknown()
			plan.Values[i].UpdatedBy = types.Int64Unknown()
			plan.Values[i].CreatedOn = types.StringUnknown()
			plan.Values[i].UpdatedOn = types.StringUnknown()
		} else {
			// Existing element: copy value_id and immutable audit fields by name.
			if !sv.valueID.IsNull() && !sv.valueID.IsUnknown() {
				plan.Values[i].ValueID = sv.valueID
			} else {
				plan.Values[i].ValueID = types.StringUnknown()
			}
			if !sv.createdBy.IsNull() && !sv.createdBy.IsUnknown() {
				plan.Values[i].CreatedBy = sv.createdBy
			} else {
				plan.Values[i].CreatedBy = types.Int64Unknown()
			}
			if !sv.createdOn.IsNull() && !sv.createdOn.IsUnknown() {
				plan.Values[i].CreatedOn = sv.createdOn
			} else {
				plan.Values[i].CreatedOn = types.StringUnknown()
			}

			if isModification {
				// The API stamps updated_by/updated_on on every write; plan as unknown.
				plan.Values[i].UpdatedBy = types.Int64Unknown()
				plan.Values[i].UpdatedOn = types.StringUnknown()
			} else {
				// No-op plan: preserve state values to avoid spurious drift.
				plan.Values[i].UpdatedBy = sv.updatedBy
				plan.Values[i].UpdatedOn = sv.updatedOn
			}
		}
		modified = true
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
			createdBy := types.Int64Null()
			if v.CreatedBy != nil {
				createdBy = types.Int64Value(int64(*v.CreatedBy))
			}
			updatedBy := types.Int64Null()
			if v.UpdatedBy != nil {
				updatedBy = types.Int64Value(int64(*v.UpdatedBy))
			}
			createdOn := types.StringNull()
			if v.CreatedOn != nil {
				createdOn = types.StringValue(*v.CreatedOn)
			}
			updatedOn := types.StringNull()
			if v.UpdatedOn != nil {
				updatedOn = types.StringValue(*v.UpdatedOn)
			}
			newValues = append(newValues, ResourceLabelValueModel{
				ValueID:     types.StringValue(v.ValueID),
				Name:        types.StringValue(v.Name),
				Description: optionalStringValue(v.Description),
				CreatedBy:   createdBy,
				UpdatedBy:   updatedBy,
				CreatedOn:   createdOn,
				UpdatedOn:   updatedOn,
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

// optionalInt64Value returns types.Int64Null() when n is 0 (API not set), or
// types.Int64Value(n) otherwise. Used for computed integer audit fields.
func optionalInt64Value(n int) types.Int64 {
	if n == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(n))
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
		// Audit fields (created_by, updated_by, created_on, updated_on) are often
		// omitted from Create/Update API responses (returned as 0/""). Fall back to
		// the ref (plan or prior state) value so the state stays consistent with
		// the plan. The GET response used by Read always returns them, so real
		// changes surface on the next refresh.
		createdBy := optionalInt64Value(v.CreatedBy)
		if createdBy.IsNull() && !ref.CreatedBy.IsNull() && !ref.CreatedBy.IsUnknown() {
			createdBy = ref.CreatedBy
		}
		updatedBy := optionalInt64Value(v.UpdatedBy)
		if updatedBy.IsNull() && !ref.UpdatedBy.IsNull() && !ref.UpdatedBy.IsUnknown() {
			updatedBy = ref.UpdatedBy
		}
		createdOn := optionalStringValue(v.CreatedOn)
		if createdOn.IsNull() && !ref.CreatedOn.IsNull() && !ref.CreatedOn.IsUnknown() {
			createdOn = ref.CreatedOn
		}
		updatedOn := optionalStringValue(v.UpdatedOn)
		if updatedOn.IsNull() && !ref.UpdatedOn.IsNull() && !ref.UpdatedOn.IsUnknown() {
			updatedOn = ref.UpdatedOn
		}
		result = append(result, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: preserveOptionalString(v.Description, ref.Description),
			CreatedBy:   createdBy,
			UpdatedBy:   updatedBy,
			CreatedOn:   createdOn,
			UpdatedOn:   updatedOn,
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
				CreatedBy:   optionalInt64Value(v.CreatedBy),
				UpdatedBy:   optionalInt64Value(v.UpdatedBy),
				CreatedOn:   optionalStringValue(v.CreatedOn),
				UpdatedOn:   optionalStringValue(v.UpdatedOn),
			})
		}
	}

	return result
}
