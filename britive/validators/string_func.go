package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// stringFuncValidator adapts a plain Go validation function into a Framework validator.String.
// It calls the provided function with the string value and maps any returned error to an
// attribute diagnostic. Null and unknown values are skipped.
type stringFuncValidator struct {
	description         string
	markdownDescription string
	validate            func(value string) error
}

// Description returns a plain text description.
func (v stringFuncValidator) Description(_ context.Context) string {
	return v.description
}

// MarkdownDescription returns a markdown description.
func (v stringFuncValidator) MarkdownDescription(_ context.Context) string {
	if v.markdownDescription != "" {
		return v.markdownDescription
	}
	return v.description
}

// ValidateString performs the validation by calling the wrapped function.
func (v stringFuncValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := v.validate(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			fmt.Sprintf("%s", err),
		)
	}
}

// StringFunc returns a validator.String that delegates to the provided validation function.
// The description is used for both plain-text and markdown descriptions unless
// markdownDescription is also supplied via StringFuncWithMarkdown.
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func StringFunc(description string, fn func(string) error) validator.String {
	return stringFuncValidator{
		description: description,
		validate:    fn,
	}
}

// StringFuncWithMarkdown is like StringFunc but accepts separate plain-text and markdown descriptions.
func StringFuncWithMarkdown(description, markdownDescription string, fn func(string) error) validator.String {
	return stringFuncValidator{
		description:         description,
		markdownDescription: markdownDescription,
		validate:            fn,
	}
}
