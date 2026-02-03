package britive_client

import "errors"

var (
	EmptyString              = ""
	ProfileLockName          = "profileLock"
	ApplicationLockName      = "applicationLock"
	AdvancedSettingLockName  = "advancedSettingLock"
	ConstraintLockName       = "constraintLock"
	IdentityProviderLockName = "identityProviderLock"
	TagLockName              = "tagLock"
	UserLockName             = "userLock"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNoContent    = errors.New("no content")
	ErrNotSupported = errors.New("not supported")
)
