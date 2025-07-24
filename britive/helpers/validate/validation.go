package validate

import (
	"fmt"
	"time"
	"unicode"

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
