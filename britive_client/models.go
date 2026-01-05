package britive_client

// HTTPErrorResponse - godoc
type HTTPErrorResponse struct {
	Status    int64       `json:"status"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"errorCode"`
	Details   interface{} `json:"details"`
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
	PolicyOrderingEnabled         bool                 `json:"policyOrderingEnabled,omitempty"`
	AllowImpersonation            bool                 `json:"delegationEnabled,omitempty"`
}

// ProfileAssociation - godoc
type ProfileAssociation struct {
	ProfileAssociationID interface{} `json:"papScopeId,omitempty"`
	Type                 string      `json:"type"`
	AppContainerID       interface{} `json:"appContainerId,omitempty"`
	Value                string      `json:"value"`
	ProfileID            string      `json:"papId,omitempty"`
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

// ApplicationType - godoc
type ApplicationType struct {
	ApplicationType string `json:"catalogAppName,omitempty"`
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

// Application - godoc
type Application struct {
	AppContainerID        string `json:"appContainerId"`
	CatalogAppDisplayName string `json:"catalogAppDisplayName,omitempty"`
}

// ApplicationRequest - godoc
type ApplicationRequest struct {
	CatalogAppId          int64  `json:"catalogAppId"`
	CatalogAppDisplayName string `json:"catalogAppDisplayName"`
}

// ApplicationResponse - godoc
type ApplicationResponse struct {
	AppContainerId        string                           `json:"appContainerId"`
	CatalogAppId          int64                            `json:"catalogAppId"`
	CatalogAppDisplayName string                           `json:"catalogAppDisplayName"`
	CatalogAppName        string                           `json:"catalogAppName"`
	UserAccountMappings   []interface{}                    `json:"userAccountMappings,omitempty"`
	Properties            Properties                       `json:"catalogApplication,omitempty"`
	RootEnvironmentGroup  *ApplicationRootEnvironmentGroup `json:"rootEnvironmentGroup,omitempty"`
}

// Properties - godoc
type Properties struct {
	PropertyTypes []PropertyTypes `json:"propertyTypes"`
	Version       string          `json:"version"`
}

// PropertyTypes - godoc
type PropertyTypes struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"type,omitempty"`
}

type UserMappings struct {
	UserAccountMappings []UserMapping `json:"userAccountMappings"`
}

type UserMapping struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// EnvAccId - godoc
type EnvAccId struct {
	AccountId     string `json:"accountId,omitempty"`
	EnvironmentId string `json:"environmentId,omitempty"`
}

// Application Environment - godoc
type ApplicationEnvironment struct {
	EnvironmentID   string `json:"id"`
	EnvironmentName string `json:"name"`
	EnvironmentType string `json:"type"`
}

// Application Entity Group - godoc
type ApplicationEntityGroup struct {
	EntityID    string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parentId"`
}

// ApplicationRootEnvironmentGroup - godoc
type ApplicationRootEnvironmentGroup struct {
	EnvironmentGroups []Association `json:"environmentGroups,omitempty"`
	Environments      []Association `json:"environments,omitempty"`
}

// SystemAppPropertyType represents a property type in the system app catalog
type SystemAppPropertyType struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Value    interface{} `json:"value"`
	Required bool        `json:"required"`
}

// SystemApp represents an app in the system app catalog
type SystemApp struct {
	CatalogAppId  int64                   `json:"catalogAppId"`
	Name          string                  `json:"name"`
	Version       string                  `json:"version"`
	PropertyTypes []SystemAppPropertyType `json:"propertyTypes"`
}

// Advanced Settings - godoc
type AdvancedSettings struct {
	Settings []Setting `json:"settings"`
}

type Setting struct {
	SettingsType            string               `json:"settingsType"`
	ID                      string               `json:"id"`
	EntityID                string               `json:"entityId"`
	EntityType              string               `json:"entityType"`
	IsInherited             *bool                `json:"isInherited"`
	IsJustificationRequired *bool                `json:"isJustificationRequired,omitempty"`
	JustificationRegex      string               `json:"justificationRegex,omitempty"`
	ConnectionID            string               `json:"connectionId,omitempty"`
	ConnectionType          string               `json:"connectionType,omitempty"`
	IsITSMEnabled           *bool                `json:"isITSMEnabled,omitempty"`
	ItsmFilterCriterias     []ItsmFilterCriteria `json:"itsmFilterCriteria,omitempty"`
	IsAutoApprovalEnabled   *bool                `json:"isAutoApprovalEnabled,omitempty"`
	EscalationPolicies      []string             `json:"escalationPolicies,omitempty"`
}

type ItsmFilterCriteria struct {
	SupportedTicketType string                 `json:"supportedTicketType,omitempty"`
	Filter              map[string]interface{} `json:"filter,omitempty"`
}

// Connections
type Connection struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	AuthType string `json:"authType,omitempty"`
}

// EscalationPolicies
type EscalationPolicies struct {
	Policies []map[string]string `json:"escalationPolicies,omitempty"`
	Count    int                 `json:"count,omitempty"`
	Page     int                 `json:"page,omitempty"`
	Size     int                 `json:"size,omitempty"`
	More     bool                `json:"more,omitempty"`
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
