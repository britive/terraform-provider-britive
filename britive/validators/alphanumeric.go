package validators

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// alphanumericValidator validates that a string value contains only alphanumeric characters,
// underscores, and dashes.
type alphanumericValidator struct{}

// Description returns a plain text description of the validator's behavior.
func (v alphanumericValidator) Description(_ context.Context) string {
	return "value must contain only alphanumeric characters, underscores, and dashes"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior.
func (v alphanumericValidator) MarkdownDescription(_ context.Context) string {
	return "value must contain only alphanumeric characters, underscores (`_`), and dashes (`-`)"
}

// ValidateString performs the validation.
func (v alphanumericValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Check if string is empty or only whitespace
	if strings.TrimSpace(value) == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			"value must not be empty or whitespace",
		)
		return
	}

	// Check if string contains only allowed characters
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' && char != '-' {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Characters",
				fmt.Sprintf("'%s' contains invalid characters. Allowed characters are: alphanumeric and special characters: ['_', '-']", value),
			)
			return
		}
	}
}

// Alphanumeric returns a validator which ensures that any configured string value
// contains only alphanumeric characters, underscores, and dashes.
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func Alphanumeric() validator.String {
	return alphanumericValidator{}
}
