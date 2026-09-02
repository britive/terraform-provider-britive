package britive

import (
	"errors"
)

const (
	defaultMaxRetries                = 10
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
	rotationTemplateLockName         = "rotationTemplate"
	scanSettingsLockName             = "scanSettings"
	scheduleScanLockName             = "scheduleScan"
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
	// ErrScheduleScanTaskServiceNotBootstrapped - a resource type's scan task service does
	// not exist yet. Confirmed by capture: this is the API's normal response (400/E1004)
	// until the first britive_resource_manager_resource_type_schedule_scan task is created
	// for that resource type - not a real error condition, just "nothing scheduled yet".
	ErrScheduleScanTaskServiceNotBootstrapped = errors.New("resource type's scan task service has not been created yet (no schedule scan task exists for it)")
)
