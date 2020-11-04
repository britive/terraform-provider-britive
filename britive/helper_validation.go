package britive

import (
	"fmt"
	"time"
)

//Validation - For the custom validation functions
type Validation struct {
}

//NewValidation - Initialises new Validation
func NewValidation() *Validation {
	return &Validation{}
}

//DurationValidateFunc - To validate duration
func (v *Validation) DurationValidateFunc(val interface{}, key string) (warns []string, errs []error) {
	value := val.(string)
	_, err := time.ParseDuration(value)
	if err != nil {
		errs = append(errs, fmt.Errorf("%s must be duration. [e.g 1s, 10m, 2h, 5d], got: %s", key, value))
	}
	return
}
