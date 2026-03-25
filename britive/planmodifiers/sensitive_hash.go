package planmodifiers

import (
	"context"
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/argon2"
)

// SensitiveHashModifier is a plan modifier that uses argon2 hashing to compare sensitive values
type SensitiveHashModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m SensitiveHashModifier) Description(_ context.Context) string {
	return "Uses argon2 hash comparison for sensitive properties to detect changes without storing plaintext"
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m SensitiveHashModifier) MarkdownDescription(_ context.Context) string {
	return "Uses argon2 hash comparison for sensitive properties to detect changes without storing plaintext"
}

// PlanModifyString implements the plan modification logic.
func (m SensitiveHashModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If plan value is null or unknown, nothing to do
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	planValue := req.PlanValue.ValueString()

	// If state value is null or unknown (first apply), hash the plan value now.
	// This matches SDK v2 StateFunc behavior: always store hash in state, never plaintext.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		resp.PlanValue = types.StringValue(getHash(planValue))
		return
	}

	// State exists - compare plan value with stored hash
	stateHash := req.StateValue.ValueString()
	planHash := getHash(planValue)

	// If hashes match, use the state value (preserves exact hash)
	if stateHash == planHash {
		resp.PlanValue = req.StateValue
	} else {
		// Value has changed, store the hash of the new value
		resp.PlanValue = types.StringValue(planHash)
	}
}

// getHash computes an argon2 hash of the input string
func getHash(val string) string {
	hash := argon2.IDKey([]byte(val), []byte{}, 1, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(hash)
}

// GetHash is the exported version for use by other packages
func GetHash(val string) string {
	return getHash(val)
}

// IsHashValue checks if stateValue is a hash of plainValue
func IsHashValue(stateValue, plainValue string) bool {
	return stateValue == getHash(plainValue)
}

// SensitiveHash returns a new SensitiveHashModifier
func SensitiveHash() planmodifier.String {
	return SensitiveHashModifier{}
}
