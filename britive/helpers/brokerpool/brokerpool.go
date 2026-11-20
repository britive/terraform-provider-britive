// Package brokerpool holds the pieces shared by the britive_resource_manager_broker_pool
// resource and data source: the read-only "brokers" and "resources" lists, the checks on token
// and label names, and the planner that changes a pool's tokens one API call at a time.
package brokerpool

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// TokenStatusActive and TokenStatusInactive are the only token statuses the API accepts.
	TokenStatusActive   = "ACTIVE"
	TokenStatusInactive = "INACTIVE"

	// DefaultTokenName is the name of the token the API creates with every pool.
	DefaultTokenName = "default"

	// KeepAliveSecondsDefault and DisconnectSecondsDefault are the values Britive gives a new
	// pool; the provider uses them when the timings are not configured.
	KeepAliveSecondsDefault  = 60
	DisconnectSecondsDefault = 86400
)

// NamePattern is what the API accepts for token names and pool label keys: letters, numbers,
// underscore and hyphen.
var NamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// BrokerObjectType is the shape of one element of the "brokers" list.
var BrokerObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"broker_id":        types.StringType,
		"broker_name":      types.StringType,
		"status":           types.StringType,
		"status_timestamp": types.StringType,
		"created":          types.StringType,
	},
}

// LinkedResourceObjectType is the shape of one element of the "resources" list.
var LinkedResourceObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"resource_id":        types.StringType,
		"name":               types.StringType,
		"resource_type_name": types.StringType,
	},
}

// BrokerModel mirrors BrokerObjectType.
type BrokerModel struct {
	BrokerID        types.String `tfsdk:"broker_id"`
	BrokerName      types.String `tfsdk:"broker_name"`
	Status          types.String `tfsdk:"status"`
	StatusTimestamp types.String `tfsdk:"status_timestamp"`
	Created         types.String `tfsdk:"created"`
}

// LinkedResourceModel mirrors LinkedResourceObjectType.
type LinkedResourceModel struct {
	ResourceID       types.String `tfsdk:"resource_id"`
	Name             types.String `tfsdk:"name"`
	ResourceTypeName types.String `tfsdk:"resource_type_name"`
}

// BrokersToList converts the API's brokers into the "brokers" list value.
func BrokersToList(ctx context.Context, brokers []britive.RemoteBroker, diags *diag.Diagnostics) types.List {
	models := make([]BrokerModel, 0, len(brokers))
	for _, b := range brokers {
		models = append(models, BrokerModel{
			BrokerID:        types.StringValue(b.BrokerID),
			BrokerName:      types.StringValue(b.BrokerName),
			Status:          types.StringValue(b.Status),
			StatusTimestamp: types.StringValue(b.StatusTimestamp),
			Created:         types.StringValue(b.Created),
		})
	}
	list, d := types.ListValueFrom(ctx, BrokerObjectType, models)
	diags.Append(d...)
	return list
}

// LinkedResourcesToList converts the API's linked resources into the "resources" list value.
func LinkedResourcesToList(ctx context.Context, resources []britive.BrokerPoolLinkedResource, diags *diag.Diagnostics) types.List {
	models := make([]LinkedResourceModel, 0, len(resources))
	for _, r := range resources {
		models = append(models, LinkedResourceModel{
			ResourceID:       types.StringValue(r.ID),
			Name:             types.StringValue(r.Name),
			ResourceTypeName: types.StringValue(r.ResourceTypeName),
		})
	}
	list, d := types.ListValueFrom(ctx, LinkedResourceObjectType, models)
	diags.Append(d...)
	return list
}

// PoolCompositeID builds the pool resource's `id` value.
func PoolCompositeID(poolID string) string {
	return "resource-manager/broker-pools/" + poolID
}

// ParseTokenID extracts the pool ID and token name from "{pool_id}/{name}" (the id of a token
// in the pool resource's tokens map) or "resource-manager/broker-pools/{pool_id}/tokens/{name}".
// It is used by the token ephemeral resource's token_id.
func ParseTokenID(id string) (string, string, error) {
	return ParseChildID(id, "tokens", "name")
}

// ParseChildID extracts the pool ID and child name from
// "resource-manager/broker-pools/{pool_id}/{kind}/{name}" or "{pool_id}/{name}". nameLabel is
// only used in the error message.
func ParseChildID(id, kind, nameLabel string) (string, string, error) {
	parts := strings.Split(id, "/")
	switch {
	case len(parts) == 2 && parts[0] != "" && parts[1] != "":
		return parts[0], parts[1], nil
	case len(parts) == 5 && parts[0] == "resource-manager" && parts[1] == "broker-pools" && parts[3] == kind && parts[2] != "" && parts[4] != "":
		return parts[2], parts[4], nil
	default:
		return "", "", fmt.Errorf("must be 'resource-manager/broker-pools/{pool_id}/%s/{%s}' or '{pool_id}/{%s}', got: %s", kind, nameLabel, nameLabel, id)
	}
}

// Int64Between returns a validator that accepts values from min to max, inclusive. The
// framework's int64validator package is not vendored in this repo, so this small one is used.
func Int64Between(min, max int64) validator.Int64 {
	return int64BetweenValidator{min: min, max: max}
}

type int64BetweenValidator struct {
	min, max int64
}

func (v int64BetweenValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be between %d and %d", v.min, v.max)
}

func (v int64BetweenValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64BetweenValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueInt64()
	if value < v.min || value > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("%s, got: %d", v.Description(ctx), value),
		)
	}
}
