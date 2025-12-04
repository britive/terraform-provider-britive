package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type StringFuncValidator struct {
	attrName string
	fn       func(string) error
	desc     string
}

func StringFunc(attrName string, fn func(string) error) validator.String {
	return StringFuncValidator{
		attrName: attrName,
		fn:       fn,
	}
}

func StringFuncWithDescription(attrName, desc string, fn func(string) error) validator.String {
	return StringFuncValidator{
		attrName: attrName,
		fn:       fn,
		desc:     desc,
	}
}

func (v StringFuncValidator) Description(ctx context.Context) string {
	if v.desc != "" {
		return v.desc
	}
	if v.attrName != "" {
		return fmt.Sprintf("Custom validation for %s.", v.attrName)
	}
	return "Custom string validation."
}

func (v StringFuncValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v StringFuncValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() || req.ConfigValue.ValueString() == britive_client.EmptyString {
		return
	}

	val := req.ConfigValue.ValueString()

	if err := v.fn(val); err != nil {
		title := "Invalid value"
		if v.attrName != "" {
			title = "Invalid " + v.attrName
		}

		resp.Diagnostics.AddAttributeError(
			req.Path,
			title,
			err.Error(),
		)
	}
}

func CaseInsensitiveOneOf(allowed ...string) func(string) error {
	allowedLower := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedLower[strings.ToLower(a)] = struct{}{}
	}

	return func(s string) error {
		if _, ok := allowedLower[strings.ToLower(s)]; ok {
			return nil
		}
		return fmt.Errorf("value must be one of (case-insensitive): %s", strings.Join(allowed, ", "))
	}
}
