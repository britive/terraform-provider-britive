package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/brokerpool"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &BrokerPoolLabelsResource{}
	_ resource.ResourceWithConfigure      = &BrokerPoolLabelsResource{}
	_ resource.ResourceWithImportState    = &BrokerPoolLabelsResource{}
	_ resource.ResourceWithValidateConfig = &BrokerPoolLabelsResource{}
)

// BrokerPoolLabelsResource manages every label of a broker pool. The API returns all of a
// pool's labels in one call, so one resource per pool holds them all; label keys that are not
// configured are removed.
type BrokerPoolLabelsResource struct {
	client *britive.Client
}

// BrokerPoolLabelsResourceModel describes the resource data model.
type BrokerPoolLabelsResourceModel struct {
	ID     types.String `tfsdk:"id"`
	PoolID types.String `tfsdk:"pool_id"`
	// A types.Set, not a plain Go slice, for the same reason as
	// ScheduleScanResourceModel.ResourceLabels: a dynamic "resource_labels" block inside a
	// for_each resource is unknown as a whole during validate/plan, which a slice can't hold.
	// Elements are ResourceLabelModel; convert with resourceLabelsFromSet/resourceLabelsToSet.
	ResourceLabels types.Set `tfsdk:"resource_labels"`
}

// NewBrokerPoolLabelsResource is a helper function to simplify the provider implementation.
func NewBrokerPoolLabelsResource() resource.Resource {
	return &BrokerPoolLabelsResource{}
}

// Metadata returns the resource type name.
func (r *BrokerPoolLabelsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_broker_pool_labels"
}

// Schema defines the schema for the resource.
func (r *BrokerPoolLabelsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages all labels of a Britive resource manager broker pool. Label keys that are not configured are removed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the pool's labels, in the form resource-manager/broker-pools/{pool_id}/labels.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.StringAttribute{
				Description: "The ID of the broker pool. Changing it creates a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"resource_labels": schema.SetNestedBlock{
				Description: "Labels assigned to the pool. One block per label key; at least one block is required.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"label_key": schema.StringAttribute{
							Required:    true,
							Description: "The label key: the name of an existing resource label. Only letters, numbers, underscore and hyphen. Each key can appear in only one block.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(brokerpool.NamePattern, "must contain only letters, numbers, underscore and hyphen"),
							},
						},
						"values": schema.SetAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "The label values: values of that resource label. Can be empty.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *BrokerPoolLabelsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig requires at least one resource_labels block and rejects two blocks with the
// same label_key. Only the resource_labels attribute is decoded, and only once it is fully
// known, so a dynamic block that is still unknown at validate time never fails here.
func (r *BrokerPoolLabelsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var labelsSet types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("resource_labels"), &labelsSet)...)
	if resp.Diagnostics.HasError() || labelsSet.IsUnknown() {
		return
	}
	if labelsSet.IsNull() || len(labelsSet.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("resource_labels"),
			"Missing resource_labels Block",
			"At least one resource_labels block is required. To remove all labels from the pool, remove this resource.",
		)
		return
	}

	labels := resourceLabelsFromSet(ctx, labelsSet, &resp.Diagnostics)
	seen := make(map[string]bool, len(labels))
	for _, label := range labels {
		if label.LabelKey.IsNull() || label.LabelKey.IsUnknown() {
			continue
		}
		key := label.LabelKey.ValueString()
		if seen[key] {
			resp.Diagnostics.AddAttributeError(
				path.Root("resource_labels"),
				"Duplicate resource_labels Block",
				fmt.Sprintf("label_key %q appears in more than one resource_labels block. Merge their values into a single block.", key),
			)
		}
		seen[key] = true
	}
}

// Create makes the configured label keys match the plan. The pool may already have labels (for
// example added in the UI), and Britive rejects a POST for a key that exists (E1005), so the
// pool's labels are read first: configured keys that already exist are updated, the others are
// added. Keys that are not in the configuration are left alone here; the next refresh reads
// them and the plan shows them being removed. Labels have no computed parts, so the plan is
// stored as-is. Calls: GET labels, then POST per new key and PUT per existing key whose values
// differ.
func (r *BrokerPoolLabelsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BrokerPoolLabelsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(plan.PoolID.ValueString())
	labels := labelsToMap(ctx, plan.ResourceLabels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetBrokerPoolLabels(poolID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Labels", err.Error())
		return
	}
	// Only configured keys go in the "old" side, so syncLabels never deletes a key here.
	current := make(map[string][]string)
	for _, label := range existing {
		if _, configured := labels[label.Key]; configured {
			current[label.Key] = nonNilStrings(label.LabelValues)
		}
	}

	log.Printf("[INFO] Setting %d labels on broker pool %s (%d already exist)", len(labels), poolID, len(current))

	if err := r.syncLabels(poolID, current, labels); err != nil {
		resp.Diagnostics.AddError("Error Adding Broker Pool Labels", err.Error())
		return
	}

	plan.ID = types.StringValue(brokerPoolLabelsCompositeID(poolID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with every label on the pool, so keys added outside
// Terraform show up as a diff. Calls: GET labels.
func (r *BrokerPoolLabelsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BrokerPoolLabelsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(state.PoolID.ValueString())
	log.Printf("[INFO] Reading labels of broker pool %s", poolID)

	labels, err := r.client.GetBrokerPoolLabels(poolID)
	if errors.Is(err, britive.ErrNotFound) {
		log.Printf("[WARN] Broker pool %s not found, removing its labels from state", poolID)
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Broker Pool Labels", err.Error())
		return
	}

	models := make([]ResourceLabelModel, 0, len(labels))
	for _, label := range labels {
		values, d := types.SetValueFrom(ctx, types.StringType, nonNilStrings(label.LabelValues))
		resp.Diagnostics.Append(d...)
		models = append(models, ResourceLabelModel{
			LabelKey: types.StringValue(label.Key),
			Values:   values,
		})
	}

	state.ID = types.StringValue(brokerPoolLabelsCompositeID(poolID))
	state.PoolID = types.StringValue(poolID)
	state.ResourceLabels = resourceLabelsToSet(ctx, models, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update makes the pool's labels match the plan. State holds what the last refresh read, so
// the difference is exactly what has to change. Calls: POST per new key, PUT per key whose
// values changed, DELETE per removed key; unchanged keys make no call.
func (r *BrokerPoolLabelsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state BrokerPoolLabelsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(state.PoolID.ValueString())
	oldLabels := labelsToMap(ctx, state.ResourceLabels, &resp.Diagnostics)
	newLabels := labelsToMap(ctx, plan.ResourceLabels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Updating labels of broker pool %s", poolID)

	if err := r.syncLabels(poolID, oldLabels, newLabels); err != nil {
		resp.Diagnostics.AddError("Error Updating Broker Pool Labels", err.Error())
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes every label key in state from the pool. Labels that are already gone (for
// example because the pool was deleted first) count as deleted. Calls: one DELETE per key.
func (r *BrokerPoolLabelsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BrokerPoolLabelsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolID := lastPathSegment(state.PoolID.ValueString())
	labels := labelsToMap(ctx, state.ResourceLabels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Removing labels from broker pool %s", poolID)

	if err := r.syncLabels(poolID, labels, map[string][]string{}); err != nil {
		resp.Diagnostics.AddError("Error Removing Broker Pool Labels", err.Error())
		return
	}
}

// ImportState accepts "resource-manager/broker-pools/{pool_id}/labels" or "{pool_id}". The
// framework calls Read right after, which reads every label of the pool and fails the import
// if the pool does not exist, so no lookup is made here.
func (r *BrokerPoolLabelsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	poolID, err := parseBrokerPoolLabelsImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), brokerPoolLabelsCompositeID(poolID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pool_id"), poolID)...)
}

// Helper functions

// syncLabels makes the pool's labels go from oldLabels to newLabels: keys no longer wanted are
// removed, new keys are added, and keys whose values changed are replaced. Keys are handled in
// sorted order so the calls are predictable.
func (r *BrokerPoolLabelsResource) syncLabels(poolID string, oldLabels, newLabels map[string][]string) error {
	for _, key := range sortedKeys(oldLabels) {
		if _, keep := newLabels[key]; !keep {
			if err := r.client.DeleteBrokerPoolLabel(poolID, key); err != nil {
				return fmt.Errorf("removing label %q: %w", key, err)
			}
		}
	}
	for _, key := range sortedKeys(newLabels) {
		oldValues, existed := oldLabels[key]
		switch {
		case !existed:
			if err := r.client.CreateBrokerPoolLabel(poolID, britive.RemoteBrokerPoolLabel{Key: key, LabelValues: newLabels[key]}); err != nil {
				return fmt.Errorf("adding label %q: %w", key, err)
			}
		case !sameStrings(oldValues, newLabels[key]):
			if err := r.client.UpdateBrokerPoolLabel(poolID, key, newLabels[key]); err != nil {
				return fmt.Errorf("updating label %q: %w", key, err)
			}
		}
	}
	return nil
}

// labelsToMap converts a resource_labels set into label key -> values. Values are never nil,
// so an empty set is sent as [].
func labelsToMap(ctx context.Context, set types.Set, diags *diag.Diagnostics) map[string][]string {
	result := make(map[string][]string)
	for _, label := range resourceLabelsFromSet(ctx, set, diags) {
		var values []string
		diags.Append(label.Values.ElementsAs(ctx, &values, false)...)
		key := label.LabelKey.ValueString()
		if _, dup := result[key]; dup {
			diags.AddError(
				"Duplicate resource_labels Block",
				fmt.Sprintf("label_key %q appears in more than one resource_labels block. Merge their values into a single block.", key),
			)
			continue
		}
		result[key] = nonNilStrings(values)
	}
	return result
}

func brokerPoolLabelsCompositeID(poolID string) string {
	return brokerpool.PoolCompositeID(poolID) + "/labels"
}

// parseBrokerPoolLabelsImportID extracts the pool ID from
// "resource-manager/broker-pools/{pool_id}/labels" or a bare "{pool_id}".
func parseBrokerPoolLabelsImportID(id string) (string, error) {
	parts := strings.Split(id, "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		return parts[0], nil
	case len(parts) == 4 && parts[0] == "resource-manager" && parts[1] == "broker-pools" && parts[2] != "" && parts[3] == "labels":
		return parts[2], nil
	default:
		return "", fmt.Errorf("import ID must be 'resource-manager/broker-pools/{pool_id}/labels' or '{pool_id}', got: %s", id)
	}
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sameStrings reports whether a and b hold the same values, ignoring order.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}
