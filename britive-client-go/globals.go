package britive

import (
	"errors"
)

const (
	emptyString              = ""
	tagLockName              = "tag"
	profileLockName          = "profile"
	permissionLockName       = "permissions"
	roleLockName             = "role"
	policyLockName           = "policy"
	accountId                = "accountId"
	environmentId            = "environmentId"
	constraintLockName       = "constraint"
	resourceTypeLockName     = "resourceType"
	responseTemplateLockName = "responseTemplate"
	resourceTypePermissions  = "resourceTypePermissions"
	applicationLockName      = "application"
)

var (
	//ErrNotFound - godoc
	ErrNotFound  = errors.New("could not find")
	ErrNoContent = errors.New("no content")
)
