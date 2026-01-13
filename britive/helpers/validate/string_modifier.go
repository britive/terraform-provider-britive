package validate

import (
	"context"
	"fmt"

	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type DefaultStringPlanModifier struct {
	defaultString string
	fn            func() string
	desc          string
}

func StringModifierFunc(attrName string, fn func() string) planmodifier.String {
	return DefaultStringPlanModifier{
		defaultString: attrName,
		fn:            fn,
	}
}

func StringModifierFuncWithDescription(attrName, desc string, fn func() string) planmodifier.String {
	return DefaultStringPlanModifier{
		defaultString: attrName,
		fn:            fn,
		desc:          desc,
	}
}

func (m DefaultStringPlanModifier) Description(ctx context.Context) string {
	if m.desc != "" {
		return m.desc
	}
	if m.defaultString != "" {
		return fmt.Sprintf("Custom modifier for %s.", m.defaultString)
	}
	return "Custom string modification."
}

func (m DefaultStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m DefaultStringPlanModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() || req.ConfigValue.ValueString() == britive_client.EmptyString {
		return
	}
	// if req.PlanValue.IsNull() {
	// 	resp.PlanValue = types.StringValue(m.Default)
	// }
}

func DefaultConstraintPermissionType(defaultVal string) func() string {
	return func() string {
		return defaultVal
	}
}
