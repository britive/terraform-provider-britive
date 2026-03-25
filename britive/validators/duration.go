package validators

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func validateDuration(value string) error {
	if _, err := time.ParseDuration(value); err != nil {
		return fmt.Errorf("expected valid duration (e.g., 1s, 10m, 2h, 2h35m0s), got: %s", value)
	}
	return nil
}

// Duration returns a validator which ensures that any configured string value
// is a valid duration format that can be parsed by time.ParseDuration.
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func Duration() validator.String {
	return StringFuncWithMarkdown(
		"value must be a valid duration (e.g., 1s, 10m, 2h)",
		"value must be a valid duration (e.g., `1s`, `10m`, `2h`)",
		validateDuration,
	)
}
