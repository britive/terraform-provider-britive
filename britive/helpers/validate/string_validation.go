package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

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
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
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

func IsValidJSON() func(string) error {
	return func(s string) error {
		var js interface{}
		if err := json.Unmarshal([]byte(s), &js); err != nil {
			return fmt.Errorf("invalid JSON: %v", err)
		}
		return nil
	}
}

func StringIsNotWhiteSpace() func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("expected %q to not be an empty string or whitespace", s)
		}
		return nil
	}
}

func IsTimeMissinUnit() func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "0" {
			return fmt.Errorf("time: missing unit in duration '0'")
		}
		return nil
	}
}

func StringWithNoSpecialChar() func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("expected %q to not be an empty string or whitespace", s)
		}

		for i := 0; i < len(s); i++ {
			char := rune(s[i])
			if !(unicode.IsLetter(char)) && !(s[i] == '_') && !(s[i] == '-') && !(unicode.IsDigit(char)) {
				return fmt.Errorf("'%s' contains invalid characters. Allowed characters are: alphanumeric and special characters:['_', '-']", s)
			}
		}
		return nil
	}
}
