package errs

import (
	"fmt"

	"github.com/britive/terraform-provider-britive/britive-client-go"
)

// NewNotFoundErrorf - godoc
func NewNotFoundErrorf(format string, a ...interface{}) error {
	return fmt.Errorf("%w %s", britive.ErrNotFound, fmt.Sprintf(format, a...))
}

// NewNotEmptyOrWhiteSpaceError - godoc
func NewNotEmptyOrWhiteSpaceError(k string) error {
	return fmt.Errorf("expected %q to not be an empty string or whitespace", k)
}

// NewInvalidResourceIDError - godoc
func NewInvalidResourceIDError(resource string, ID string) error {
	return fmt.Errorf("invalid %s id %s, please check the terraform state file", resource, ID)
}
