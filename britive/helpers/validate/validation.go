package validate

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
	"unicode"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//Validation - For the custom validation functions
type Validation struct {
}

//NewValidation - Initializes new Validation
func NewValidation() *Validation {
	return &Validation{}
}

//DurationValidateFunc - To validate duration
func (v *Validation) DurationValidateFunc(val interface{}, key string) (warns []string, errs []error) {
	value := val.(string)
	_, err := time.ParseDuration(value)
	if err != nil {
		errs = append(errs, fmt.Errorf("expected %q to be duration. [e.g 1s, 10m, 2h, 2h35m0s], got: %s", key, value))
	}
	return
}

//To validate string with no white space or any special character except '_'or '-'
func (v *Validation) StringWithNoSpecialChar(val interface{}, key string) (warns []string, errs []error) {
	str, errArr := validation.StringIsNotWhiteSpace(val, key)
	if errArr != nil || str != nil {
		return str, errArr
	}
	value := val.(string)
	for i := 0; i < len(value); i++ {
		char := rune(value[i])
		if !(unicode.IsLetter(char)) && !(value[i] == '_') && !(value[i] == '-') && !(unicode.IsDigit(char)) {
			errs = append(errs, fmt.Errorf("'%s' contains invalid characters. Allowed characters are: alphanumeric and special characters:['_', '-']", value))
			break
		}
	}
	return
}

// To validate SVG
type SVG struct {
	XMLName xml.Name `xml:"svg"`
}

func (v *Validation) ValidateSVGString(val interface{}, key string) (warns []string, errs []error) {
	strVal, ok := val.(string)
	if !ok {
		errs = append(errs, fmt.Errorf("invalid type for %s: expected string", key))
		return
	}

	// Check size limit: max 400KB (400 * 1024 bytes)
	if len(strVal) > 400*1024 {
		errs = append(errs, fmt.Errorf("%s is too large: must be ≤ 400KB", key))
		return
	}

	// Check XML is well-formed and root is <svg>
	type SVG struct {
		XMLName xml.Name `xml:"svg"`
	}

	var svg SVG
	if err := xml.Unmarshal([]byte(strVal), &svg); err != nil {
		errs = append(errs, fmt.Errorf("invalid SVG XML: %s", err))
		return
	}

	if svg.XMLName.Local != "svg" {
		errs = append(errs, fmt.Errorf("invalid SVG: root element is <%s>, expected <svg>", svg.XMLName.Local))
		return
	}

	return
}

// To validate Immutable fields
func (v *Validation) ValidateImmutableFields(fields []string) schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
		for _, field := range fields {
			oldVal, newVal := d.GetChange(field)
			if d.HasChange(field) && oldVal != "" {
				return fmt.Errorf("field %q is immutable and cannot be changed (from '%v' to '%v')", field, oldVal, newVal)
			}
		}
		return nil
	}
}
