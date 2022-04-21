package britive

import (
	"errors"
)

const (
	emptyString        = ""
	tagLockName        = "tag"
	profileLockName    = "profile"
	permissionLockName = "permissions"
	roleLockName       = "role"
)

var (
	//ErrNotFound - godoc
	ErrNotFound  = errors.New("could not find")
	ErrNoContent = errors.New("no content")
)
