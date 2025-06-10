package britive

import (
	"errors"
)

const (
	emptyString          = ""
	tagLockName          = "tag"
	profileLockName      = "profile"
	permissionLockName   = "permissions"
	roleLockName         = "role"
	policyLockName       = "policy"
	accountId            = "accountId"
	environmentId        = "environmentId"
	constraintLockName   = "constraint"
	applicationLockName  = "application"
	environment          = "Environment"
	environmentGroup     = "EnvironmentGroup"
	serverAccessLockName = "serverAccess"
)

var (
	//ErrNotFound - godoc
	ErrNotFound  = errors.New("could not find")
	ErrNoContent = errors.New("no content")
)
