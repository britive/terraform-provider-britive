package britive_client

import "errors"

var (
	EmptyString         = ""
	ProfileLockName     = "profileLock"
	ApplicationLockName = "applicationLock"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrNoContent = errors.New("no content")
)

// package britive_cleint

// import (
// 	"errors"
// )

// const (
// 	emptyString                      = ""
// 	tagLockName                      = "tag"
// 	profileLockName                  = "profile"
// 	permissionLockName               = "permissions"
// 	roleLockName                     = "role"
// 	policyLockName                   = "policy"
// 	constraintLockName               = "constraint"
// 	applicationLockName              = "application"
// 	advancedSettingLockName          = "advancedSetting"
// 	resourceTypeLockName             = "resourceType"
// 	responseTemplateLockName         = "responseTemplate"
// 	resourceTypePermissions          = "resourceTypePermissions"
// 	resourceLabelLockName            = "resourceLabel"
// 	resourceManagerProfileLock       = "resourceManagerProfile"
// 	resourceManagerProfilePolicyLock = "resourceManagerProfilePolicy"
// 	resourceManagerProfilePermission = "resourceManagerProfilePermission"
// 	serverAccessLockName             = "serverAccess"
// 	resourceManagerResourcePolicy    = "resourceManagerResourcePolicy"
// )

// var (
// 	//ErrNotFound - godoc
// 	ErrNotFound     = errors.New("could not find")
// 	ErrNoContent    = errors.New("no content")
// 	ErrNotSupported = errors.New("not supported")
// )
