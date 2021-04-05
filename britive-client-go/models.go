package britive

import (
	"time"
)

//Config - godoc
type Config struct {
	Tenant string `json:"tenant"`
	Token  string `json:"token"`
}

//HTTPErrorResponse - godoc
type HTTPErrorResponse struct {
	Status    int64       `json:"status"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"errorCode"`
	Details   interface{} `json:"details"`
}

//Tag - godoc
type Tag struct {
	ID                       string                    `json:"userTagId,omitempty"`
	Name                     string                    `json:"name"`
	Description              string                    `json:"description"`
	Status                   string                    `json:"status,omitempty"`
	UserTagIdentityProviders []UserTagIdentityProvider `json:"userTagIdentityProviders,omitempty"`
	External                 interface{}               `json:"external,omitempty"`
}

//UserTagIdentityProvider - godoc
type UserTagIdentityProvider struct {
	IdentityProvider IdentityProvider `json:"identityProvider"`
	ExternalID       interface{}      `json:"externalId,omitempty"`
}

//IdentityProvider - godoc
type IdentityProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

//User - godoc
type User struct {
	AdminRoles       []AdminRole      `json:"adminRoles,omitempty"`
	Type             string           `json:"type,omitempty"`
	Email            string           `json:"email,omitempty"`
	Username         string           `json:"username,omitempty"`
	FirstName        string           `json:"firstName,omitempty"`
	LastName         string           `json:"lastName,omitempty"`
	Name             string           `json:"name,omitempty"`
	ExternalID       interface{}      `json:"externalId,omitempty"`
	Mobile           interface{}      `json:"mobile,omitempty"`
	IdentityProvider IdentityProvider `json:"identityProvider,omitempty"`
	MappedAccounts   []interface{}    `json:"mappedAccounts,omitempty"`
	External         bool             `json:"external,omitempty"`
	Status           string           `json:"status,omitempty"`
	UserID           string           `json:"userId,omitempty"`
}

//AdminRole - godoc
type AdminRole struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

//Profile - godoc
type Profile struct {
	ProfileID                     string               `json:"papId,omitempty"`
	AppContainerID                string               `json:"appContainerId"`
	Name                          string               `json:"name"`
	Description                   string               `json:"description"`
	Status                        string               `json:"status,omitempty"`
	Associations                  []ProfileAssociation `json:"scope,omitempty"`
	ExpirationDuration            int64                `json:"expirationDuration,omitempty"`
	Extendable                    bool                 `json:"extendable"`
	NotificationPriorToExpiration *int64               `json:"notificationPriorToExpiration,omitempty"`
	ExtensionDuration             *int64               `json:"extensionDuration,omitempty"`
	ExtensionLimit                interface{}          `json:"extensionLimit,omitempty"`
}

//Application - godoc
type Application struct {
	AppContainerID        string `json:"appContainerId"`
	CatalogAppDisplayName string `json:"catalogAppDisplayName,omitempty"`
}

//ProfilePermission - godoc
type ProfilePermission struct {
	ProfileID   string      `json:"papId,omitempty"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description interface{} `json:"description,omitempty"`
	CheckStatus string      `json:"checkStatus,omitempty"`
	Message     string      `json:"message,omitempty"`
}

//ProfilePermissionRequest - godoc
type ProfilePermissionRequest struct {
	Operation  string            `json:"op"`
	Permission ProfilePermission `json:"permission"`
}

//ApplicationRootEnvironmentGroup - godoc
type ApplicationRootEnvironmentGroup struct {
	EnvironmentGroups []Association `json:"environmentGroups,omitempty"`
	Environments      []Association `json:"environments,omitempty"`
}

//Association - godoc
type Association struct {
	ID               string      `json:"id,omitempty"`
	Name             string      `json:"name"`
	Description      interface{} `json:"description,omitempty"`
	ParentID         string      `json:"parentId,omitempty"`
	ParentGroupID    string      `json:"parentGroupId,omitempty"`
	InternalParentID string      `json:"internalParentId,omitempty"`
	Type             string      `json:"type,omitempty"`
	Status           string      `json:"status,omitempty"`
}

//ProfileAssociation - godoc
type ProfileAssociation struct {
	ProfileAssociationID interface{} `json:"papScopeId,omitempty"`
	Type                 string      `json:"type"`
	AppContainerID       interface{} `json:"appContainerId,omitempty"`
	Value                string      `json:"value"`
	ProfileID            string      `json:"papId,omitempty"`
}

//ProfileAssociationResource - godoc
type ProfileAssociationResource struct {
	ID          int64       `json:"id,omitempty"`
	Name        string      `json:"name"`
	Description interface{} `json:"description,omitempty"`
	NativeID    string      `json:"nativeId,omitempty"`
	ParentID    string      `json:"parentId,omitempty"`
	ParentName  string      `json:"parentName,omitempty"`
	Type        string      `json:"type,omitempty"`
}

//TimePeriod - godoc
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

//ProfileTag - godoc
type ProfileTag struct {
	ProfileID    string      `json:"papId,omitempty"`
	TagID        string      `json:"userTagId,omitempty"`
	Name         string      `json:"name,omitempty"`
	Description  string      `json:"description,omitempty"`
	Status       string      `json:"status,omitempty"`
	UserCount    interface{} `json:"userCount,omitempty"`
	AccessPeriod *TimePeriod `json:"accessPeriod,omitempty"`
	CheckStatus  string      `json:"checkStatus,omitempty"`
	Message      string      `json:"message,omitempty"`
}

//ProfileIdentity - godoc
type ProfileIdentity struct {
	ProfileID    string      `json:"papId,omitempty"`
	UserID       string      `json:"userId,omitempty"`
	Name         string      `json:"name,omitempty"`
	Username     string      `json:"username,omitempty"`
	UserType     string      `json:"userType,omitempty"`
	Status       string      `json:"status,omitempty"`
	AccessPeriod *TimePeriod `json:"accessPeriod,omitempty"`
	CheckStatus  string      `json:"checkStatus,omitempty"`
	Message      string      `json:"message,omitempty"`
}

//PaginationResponse - godoc
type PaginationResponse struct {
	Count  int           `json:"count"`
	Page   int           `json:"page"`
	Size   int           `json:"size"`
	Sort   string        `json:"sort,omitempty"`
	Filter string        `json:"filter,omitempty"`
	Data   []interface{} `json:"data"`
}

//UserAttribute - godoc
type UserAttribute struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DataType    string `json:"dataType"`
	MultiValued bool   `json:"multiValued"`
	BuiltIn     bool   `json:"builtIn"`
}

//SessionAttribute - godoc
type SessionAttribute struct {
	AttributeSchemaID string `json:"attributeSchemaId"`
	MappingName       string `json:"mappingName"`
	Transitive        bool   `json:"transitive"`
	ID                string `json:"id,omitempty"`
}
