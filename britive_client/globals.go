package britive_client

import "errors"

var (
	EmptyString             = ""
	ProfileLockName         = "profileLock"
	ApplicationLockName     = "applicationLock"
	AdvancedSettingLockName = "advancedSettingLock"
	ConstraintLockName      = "constraintLock"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNoContent    = errors.New("no content")
	ErrNotSupported = errors.New("not supported")
)
