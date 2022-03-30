package britive

import (
	"errors"
)

const (
	emptyString         = ""
	tagLockName         = "tag"
	profileLockName     = "profile"
	permissionsLockName = "permissions"
)

var (
	//ErrNotFound - godoc
	ErrNotFound  = errors.New("could not find")
	ErrNoContent = errors.New("no content")
)
