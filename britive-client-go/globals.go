package britive

import (
	"errors"
)

const (
	emptyString             = ""
	tagLockName             = "tag"
	profileLockName         = "profile"
	permissionLockName      = "permissions"
	roleLockName            = "role"
	policyLockName          = "policy"
	accountId               = "accountId"
	environmentId           = "environmentId"
	constraintLockName      = "constraint"
	applicationLockName     = "application"
	advancedSettingLockName = "advancedSetting"
	environment             = "environment"
	environmentGroup        = "environmentGroup"
)

var (
	//ErrNotFound - godoc
	ErrNotFound     = errors.New("could not find")
	ErrNoContent    = errors.New("no content")
	ErrNotSupported = errors.New("not supported")
)
