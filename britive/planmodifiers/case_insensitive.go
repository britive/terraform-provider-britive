package planmodifiers

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// CaseInsensitivePreserveStateModifier is a plan modifier that keeps the prior
// state value when the planned value differs from it only by letter case.
// This mirrors SDKv2's DiffSuppressFunc: strings.EqualFold(old, new), which
// suppressed case-only diffs between config and state.
type CaseInsensitivePreserveStateModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m CaseInsensitivePreserveStateModifier) Description(_ context.Context) string {
	return "Preserves the prior state value when the only difference from the planned value is letter case"
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m CaseInsensitivePreserveStateModifier) MarkdownDescription(_ context.Context) string {
	return "Preserves the prior state value when the only difference from the planned value is letter case"
}

// PlanModifyString implements the plan modification logic.
func (m CaseInsensitivePreserveStateModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	if strings.EqualFold(req.PlanValue.ValueString(), req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// CaseInsensitivePreserveState returns a new CaseInsensitivePreserveStateModifier.
func CaseInsensitivePreserveState() planmodifier.String {
	return CaseInsensitivePreserveStateModifier{}
}
