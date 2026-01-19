package validate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type BoolFuncValidator struct {
	fn   func(bool) error
	desc string
}

func BoolFunc(fn func(bool) error) validator.Bool {
	return BoolFuncValidator{
		fn: fn,
	}
}

func BoolFuncWithDescription(desc string, fn func(bool) error) validator.Bool {
	return BoolFuncValidator{
		fn:   fn,
		desc: desc,
	}
}

func (v BoolFuncValidator) Description(ctx context.Context) string {
	if v.desc != "" {
		return v.desc
	}
	return "Custom bool validation."
}

func (v BoolFuncValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v BoolFuncValidator) ValidateBool(
	ctx context.Context,
	req validator.BoolRequest,
	resp *validator.BoolResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
}

func IsPolicyPriorityEnabled() func(bool) error {
	return func(s bool) error {
		if s != true {
			return fmt.Errorf("Invalid Param.")
		}
		return nil
	}
}
