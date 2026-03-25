package validators

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// svgValidator validates that a string value is valid SVG XML.
type svgValidator struct{}

// Description returns a plain text description of the validator's behavior.
func (v svgValidator) Description(_ context.Context) string {
	return "value must be valid SVG XML with size <= 400KB"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior.
func (v svgValidator) MarkdownDescription(_ context.Context) string {
	return "value must be valid SVG XML with size ≤ 400KB"
}

// ValidateString performs the validation.
func (v svgValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Check size limit: max 400KB (400 * 1024 bytes)
	if len(value) > 400*1024 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"SVG Too Large",
			fmt.Sprintf("SVG content is too large: must be ≤ 400KB (got %d bytes)", len(value)),
		)
		return
	}

	// Check XML is well-formed and root is <svg>
	type SVG struct {
		XMLName xml.Name `xml:"svg"`
	}

	var svg SVG
	if err := xml.Unmarshal([]byte(value), &svg); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid SVG XML",
			fmt.Sprintf("SVG content is not valid XML: %s", err),
		)
		return
	}

	if svg.XMLName.Local != "svg" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid SVG Root Element",
			fmt.Sprintf("Invalid SVG: root element is <%s>, expected <svg>", svg.XMLName.Local),
		)
		return
	}
}

// SVG returns a validator which ensures that any configured string value
// is valid SVG XML with a root <svg> element and size ≤ 400KB.
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func SVG() validator.String {
	return svgValidator{}
}
