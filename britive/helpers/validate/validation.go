package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Validation - For the custom validation functions
type Validation struct {
}

// NewValidation - Initializes new Validation
func NewValidation() *Validation {
	return &Validation{}
}

// DurationValidateFunc - To validate duration
func (v *Validation) DurationValidateFunc(val interface{}, key string) (warns []string, errs []error) {
	value := val.(string)
	_, err := time.ParseDuration(value)
	if err != nil {
		errs = append(errs, fmt.Errorf("expected %q to be duration. [e.g 1s, 10m, 2h, 2h35m0s], got: %s", key, value))
	}
	return
}

func (v *Validation) ValidateFileExists(val interface{}, key string) (warns []string, errs []error) {
	path := val.(string)

	absPath, err := filepath.Abs(path)
	if err != nil {
		errs = append(errs, fmt.Errorf("%q: invalid path: %s", key, err))
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("%q: file does not exist at path: %s", key, absPath))
	}

	return
}
