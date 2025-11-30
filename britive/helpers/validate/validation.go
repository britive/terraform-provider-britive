package validate

import (
	"context"
	"strings"

	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// CaseInsensitiveOneOf validates string attribute with allowed values ignoring case.
type CaseInsensitiveOneOf struct {
	Allowed []string
}

func (v CaseInsensitiveOneOf) Description(ctx context.Context) string {
	return "String must be one of the allowed values (case-insensitive)."
}

func (v CaseInsensitiveOneOf) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v CaseInsensitiveOneOf) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() || req.ConfigValue.ValueString() == britive_client.EmptyString {
		return
	}

	value := req.ConfigValue.ValueString()
	lowerValue := strings.ToLower(value)

	for _, allowed := range v.Allowed {
		if lowerValue == strings.ToLower(allowed) {
			return // valid
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid application_type",
		"value must be one of (case-insensitive): "+strings.Join(v.Allowed, ", "),
	)
}

func CaseInsensitiveOneOfValidator(allowed ...string) validator.String {
	return CaseInsensitiveOneOf{Allowed: allowed}
}
