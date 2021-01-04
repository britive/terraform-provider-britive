package britive

import (
	"errors"
)

const (
	emptyString     = ""
	tagLockName     = "tag"
	profileLockName = "profile"
)

var (
	//ErrNotFound - godoc
	ErrNotFound = errors.New("Could not find")
)
