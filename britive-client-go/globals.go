package britive

import (
	"errors"
)

const (
	maxRetries                       = 3
	requestSleepTime                 = 180
	emptyString                      = ""
	tagLockName                      = "tag"
	profileLockName                  = "profile"
	permissionLockName               = "permissions"
	roleLockName                     = "role"
	policyLockName                   = "policy"
	accountId                        = "accountId"
	environmentId                    = "environmentId"
	constraintLockName               = "constraint"
	applicationLockName              = "application"
	advancedSettingLockName          = "advancedSetting"
	environment                      = "Environment"
	environmentGroup                 = "EnvironmentGroup"
	resourceTypeLockName             = "resourceType"
	responseTemplateLockName         = "responseTemplate"
	resourceTypePermissions          = "resourceTypePermissions"
	resourceLabelLockName            = "resourceLabel"
	resourceManagerProfileLock       = "resourceManagerProfile"
	resourceManagerProfilePolicyLock = "resourceManagerProfilePolicy"
	resourceManagerProfilePermission = "resourceManagerProfilePermission"
	serverAccessLockName             = "serverAccess"
	resourceManagerResourcePolicy    = "resourceManagerResourcePolicy"
)

var (
	//ErrNotFound - godoc
	ErrNotFound     = errors.New("could not find")
	ErrNoContent    = errors.New("no content")
	ErrNotSupported = errors.New("not supported")
)
