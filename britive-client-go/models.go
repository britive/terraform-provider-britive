package britive

// Config - godoc
type Config struct {
	Tenant string `json:"tenant"`
	Token  string `json:"token"`
}

// HTTPErrorResponse - godoc
type HTTPErrorResponse struct {
	Status    int64       `json:"status"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"errorCode"`
	Details   interface{} `json:"details"`
}

// Tag - godoc
type Tag struct {
	ID                       string                    `json:"userTagId,omitempty"`
	Name                     string                    `json:"name"`
	Description              string                    `json:"description"`
	Status                   string                    `json:"status,omitempty"`
	UserTagIdentityProviders []UserTagIdentityProvider `json:"userTagIdentityProviders,omitempty"`
	External                 interface{}               `json:"external,omitempty"`
}

// UserTagIdentityProvider - godoc
type UserTagIdentityProvider struct {
	IdentityProvider IdentityProvider `json:"identityProvider"`
	ExternalID       interface{}      `json:"externalId,omitempty"`
}

// IdentityProvider - godoc
type IdentityProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

// User - godoc
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

// AdminRole - godoc
type AdminRole struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// Profile - godoc
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
	DestinationUrl                string               `json:"destinationUrl,omitempty"`
}

// Application - godoc
type Application struct {
	AppContainerID        string `json:"appContainerId"`
	CatalogAppDisplayName string `json:"catalogAppDisplayName,omitempty"`
}

// Application Environment - godoc
type ApplicationEnvironment struct {
	EnvironmentID   string `json:"id"`
	EnvironmentName string `json:"name"`
	EnvironmentType string `json:"type"`
}

// ProfilePermission - godoc
type ProfilePermission struct {
	ProfileID   string      `json:"papId,omitempty"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description interface{} `json:"description,omitempty"`
	CheckStatus string      `json:"checkStatus,omitempty"`
	Message     string      `json:"message,omitempty"`
}

// ProfilePermissionRequest - godoc
type ProfilePermissionRequest struct {
	Operation  string            `json:"op"`
	Permission ProfilePermission `json:"permission"`
}

// Condition Constraint - godoc
type ConditionConstraint struct {
	Title       string `json:"title"`
	Expression  string `json:"expression"`
	Description string `json:"description"`
}

// Condition Constraint Result - godoc
type ConditionConstraintResult struct {
	Result []ConditionConstraint `json:"result"`
}

// Constraint - godoc
type Constraint struct {
	Name string `json:"name"`
}

// Constraint Result - godoc
type ConstraintResult struct {
	Result []Constraint `json:"result"`
}

// ApplicationRootEnvironmentGroup - godoc
type ApplicationRootEnvironmentGroup struct {
	EnvironmentGroups []Association `json:"environmentGroups,omitempty"`
	Environments      []Association `json:"environments,omitempty"`
}

// Association - godoc
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

// ProfileAssociation - godoc
type ProfileAssociation struct {
	ProfileAssociationID interface{} `json:"papScopeId,omitempty"`
	Type                 string      `json:"type"`
	AppContainerID       interface{} `json:"appContainerId,omitempty"`
	Value                string      `json:"value"`
	ProfileID            string      `json:"papId,omitempty"`
}

// ProfileAssociationResource - godoc
type ProfileAssociationResource struct {
	ID          int64       `json:"id,omitempty"`
	Name        string      `json:"name"`
	Description interface{} `json:"description,omitempty"`
	NativeID    string      `json:"nativeId,omitempty"`
	ParentID    string      `json:"parentId,omitempty"`
	ParentName  string      `json:"parentName,omitempty"`
	Type        string      `json:"type,omitempty"`
}

// ApplicationType - godoc
type ApplicationType struct {
	ApplicationType string `json:"catalogAppName,omitempty"`
}

// EnvAccId - godoc
type EnvAccId struct {
	AccountId     string `json:"accountId,omitempty"`
	EnvironmentId string `json:"environmentId,omitempty"`
}

// ProfilePolicy - godoc
type ProfilePolicy struct {
	ProfileID    string                     `json:"papId,omitempty"`
	PolicyID     string                     `json:"id,omitempty"`
	Name         string                     `json:"name,omitempty"`
	Description  string                     `json:"description"`
	Condition    string                     `json:"condition"`
	Members      interface{}                `json:"members"`
	Consumer     string                     `json:"consumer"`
	AccessType   string                     `json:"accessType"`
	IsActive     bool                       `json:"isActive"`
	IsDraft      bool                       `json:"isDraft"`
	IsReadOnly   bool                       `json:"isReadOnly"`
	Associations []ProfilePolicyAssociation `json:"scopes,omitempty"`
}

type ProfilePolicyAssociation struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// PaginationResponse - godoc
type PaginationResponse struct {
	Count  int           `json:"count"`
	Page   int           `json:"page"`
	Size   int           `json:"size"`
	Sort   string        `json:"sort,omitempty"`
	Filter string        `json:"filter,omitempty"`
	Data   []interface{} `json:"data"`
}

// UserAttribute - godoc
type UserAttribute struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DataType    string `json:"dataType"`
	MultiValued bool   `json:"multiValued"`
	BuiltIn     bool   `json:"builtIn"`
}

// SessionAttribute - godoc
type SessionAttribute struct {
	AttributeSchemaID    string `json:"attributeSchemaId"`
	MappingName          string `json:"mappingName"`
	Transitive           bool   `json:"transitive"`
	SessionAttributeType string `json:"sessionAttributeType"`
	AttributeValue       string `json:"attributeValue"`
	ID                   string `json:"id,omitempty"`
}

// Permission - godoc
type Permission struct {
	PermissionID string        `json:"id,omitempty"`
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	Consumer     string        `json:"consumer"`
	Resources    []interface{} `json:"resources"`
	Actions      []interface{} `json:"actions"`
}

// Resource - godoc
type Role struct {
	RoleID      string      `json:"id,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Permissions interface{} `json:"permissions"`
}

// Policy - godoc
type Policy struct {
	PolicyID    string      `json:"id,omitempty"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description"`
	Condition   string      `json:"condition"`
	Members     interface{} `json:"members"`
	Roles       interface{} `json:"roles"`
	Permissions interface{} `json:"permissions"`
	AccessType  string      `json:"accessType"`
	IsActive    bool        `json:"isActive"`
	IsDraft     bool        `json:"isDraft"`
	IsReadOnly  bool        `json:"isReadOnly"`
}
