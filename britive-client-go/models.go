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
	Requestable              bool                      `json:"requestable,omitempty"`
	Attributes               []TagAttribute            `json:"attributes,omitempty"`
}

// TagAttribute - godoc
type TagAttribute struct {
	AttributeName  string `json:"attributeName"`
	AttributeValue string `json:"attributeValue"`
	AttributeID    string `json:"attributeId,omitempty"`
}

// TagAttributesUpdateRequest - Request body for PATCH /user-tags (attributes + requestable update)
type TagAttributesUpdateRequest struct {
	UserTagID   string         `json:"userTagId"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Requestable *bool          `json:"requestable,omitempty"`
	Attributes  []TagAttribute `json:"attributes"`
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
	PolicyOrderingEnabled         bool                 `json:"policyOrderingEnabled,omitempty"`
	// DelegationEnabled must NOT use omitempty: Go omits false for bool with omitempty,
	// which would prevent the PATCH from ever setting delegation to false.
	DelegationEnabled             bool                 `json:"delegationEnabled"`
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
	Associations []ProfilePolicyAssociation `json:"scopes"`
	ScopeTags    []ScopeTag                 `json:"scopeTags,omitempty"`
	Order        int                        `json:"order,omitempty"`
}

type ProfilePolicyAssociation struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ScopeTag - represents a tag-based scope filter
type ScopeTag struct {
	TagKey    string   `json:"tagKey"`
	TagValues []string `json:"tagValues"`
}

type ProfileSummary struct {
	AppContainerID                string `json:"appContainerId,omitempty"`
	PapId                         string `json:"papId,omitempty"`
	Name                          string `json:"name"`
	Description                   string `json:"description"`
	DestinationUrl                string `json:"destinationUrl,omitempty"`
	ExpirationDuration            int    `json:"expirationDuration,omitempty"`
	ExtensionDuration             int    `json:"extensionDuration,omitempty"`
	Extendable                    bool   `json:"extendable"`
	ExtensionLimit                int    `json:"extensionLimit,omitempty"`
	NotificationPriorToExpiration int    `json:"notificationPriorToExpiration"`
	PolicyOrderingEnabled         bool   `json:"policyOrderingEnabled"`
	UseDefaultAppUrl              bool   `json:"useDefaultAppUrl,omitempty"`
}

type ProfilePolicyPriority struct {
	ProfileID             string `json:"papId"`
	PolicyOrderingEnabled bool   `json:"policyOrderingEnabled"`
	PolicyOrder           []PolicyOrder
}

type PolicyOrder struct {
	Id    string `json:"id"`
	Order int    `json:"order"`
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

// Profile Additional Settings - godoc
type ProfileAdditionalSettings struct {
	ProfileID                    string `json:"papId"`
	UseApplicationCredentialType bool   `json:"useApplicationCredentialType"`
	ConsoleAccess                bool   `json:"consoleAccess"`
	ProgrammaticAccess           bool   `json:"programmaticAccess"`
	ProjectIdForServiceAccount   string `json:"projectIdForServiceAccount"`
}

// Permission - godoc
type Permission struct {
	PermissionID     string        `json:"id,omitempty"`
	Name             string        `json:"name"`
	Description      *string       `json:"description"`
	Consumer         string        `json:"consumer"`
	Resources        []interface{} `json:"resources"`
	Actions          []interface{} `json:"actions"`
	PermissionScopes []interface{} `json:"permissionScopes"`
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

type ApplicationRequest struct {
	CatalogAppId          int    `json:"catalogAppId"`
	CatalogAppDisplayName string `json:"catalogAppDisplayName"`
}

type ApplicationResponse struct {
	AppContainerId        string                           `json:"appContainerId"`
	CatalogAppId          int                              `json:"catalogAppId"`
	CatalogAppDisplayName string                           `json:"catalogAppDisplayName"`
	CatalogAppName        string                           `json:"catalogAppName"`
	UserAccountMappings   []interface{}                    `json:"userAccountMappings,omitempty"`
	Properties            Properties                       `json:"catalogApplication,omitempty"`
	RootEnvironmentGroup  *ApplicationRootEnvironmentGroup `json:"rootEnvironmentGroup,omitempty"`
}

type Properties struct {
	PropertyTypes []PropertyTypes `json:"propertyTypes"`
	Version       string          `json:"version"`
}

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

// Application Entity Environment - godoc
type ApplicationEntityEnvironment struct {
	EntityID      string `json:"id,omitempty"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ParentGroupID string `json:"parentGroupId"`
}

// Application Entity Group - godoc
type ApplicationEntityGroup struct {
	EntityID    string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parentId"`
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

// ResourceType - godoc
type ResourceType struct {
	ResourceTypeID string      `json:"resourceTypeId,omitempty"`
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	Parameters     []Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	ParamName   string `json:"name"`
	ParamType   string `json:"paramType"`
	IsMandatory bool   `json:"isMandatory"`
}

type ResponseTemplate struct {
	TemplateID             string `json:"templateId,omitempty"`
	Name                   string `json:"name"`
	Description            string `json:"description,omitempty"`
	IsConsoleAccessEnabled bool   `json:"isConsoleAccessEnabled"`
	ShowOnUI               bool   `json:"show_on_ui"`
	TemplateData           string `json:"template_data"`
}

type AllResponseTemplates struct {
	Count             int                `json:"count,omitempty"`
	ResponseTemplates []ResponseTemplate `json:"data,omitempty"`
}

// RotationTemplateCreateRequest - request body for creating a rotation template stub.
// The mode (local/inline-code/file), time limit, and variables are configured afterwards
// via UpdateRotationTemplate; the create call only accepts name/description.
type RotationTemplateCreateRequest struct {
	Name        string `json:"rotationTemplateName"`
	Description string `json:"rotationTemplateDesc,omitempty"`
}

// RotationTemplateSummary - shape returned by rotation template creation and by the
// paginated rotation-templates list endpoint (a thinner shape than RotationTemplate's
// detail/update representation - e.g. "templateId"/"templateName" instead of "id"/"rotationTemplateName").
type RotationTemplateSummary struct {
	TemplateID   string `json:"templateId,omitempty"`
	TemplateName string `json:"templateName,omitempty"`
	Description  string `json:"description,omitempty"`
	CreatedOn    string `json:"createdOn,omitempty"`
	CreatedBy    string `json:"createdBy,omitempty"`
}

// RotationTemplateVariable - a single variable exposed to a rotation template's script.
type RotationTemplateVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	MultiValued bool   `json:"multivalued"`
}

// RotationTemplate - full detail of a rotation template, and the request/response shape
// for the metadata update call. Note: rotationTemplateName/rotationTemplateDesc are never
// accepted by the update call (confirmed by capture - present on GET, absent from every PUT
// body observed), so callers building an update payload must leave Name/Description unset.
type RotationTemplate struct {
	ID             string                     `json:"id,omitempty"`
	ResourceTypeID string                     `json:"resourceTypeId,omitempty"`
	ResourceType   string                     `json:"resourceType,omitempty"`
	Name           string                     `json:"rotationTemplateName,omitempty"`
	Description    string                     `json:"rotationTemplateDesc,omitempty"`
	TimeoutLimit   int                        `json:"timeoutLimit"`
	IsLocal        bool                       `json:"isLocal"`
	InlineFile     bool                       `json:"inlineFile"`
	EditorType     string                     `json:"editorType,omitempty"`
	ScriptName     string                     `json:"scriptName,omitempty"`
	Variables      []RotationTemplateVariable `json:"variables"`
	PresignedURL   string                     `json:"presignedUrl,omitempty"`
	CreatedOn      string                     `json:"createdOn,omitempty"`
	CreatedBy      string                     `json:"createdBy,omitempty"`
	UpdatedOn      string                     `json:"updatedOn,omitempty"`
	UpdatedBy      string                     `json:"updatedBy,omitempty"`
}

// PresignedURLResponse - response body of a presigned-url endpoint (shared shape between
// rotation templates' and scan settings' presigned-url endpoints).
type PresignedURLResponse struct {
	PresignedURL string `json:"presignedUrl"`
}

// ScanSettings - a resource type's scan settings. Unlike RotationTemplate, this is a
// singleton per resource type (no name/description, created via an idempotent PUT rather
// than POST-then-PUT) - see resource_manager_resource_type_scan_settings.go's doc comment.
// ScriptName intentionally has no `omitempty`: confirmed by capture that the API honors an
// explicit "" (clearing a previously-set script name on switching to Local), unlike
// RotationTemplate where the field is only ever added, never explicitly cleared.
type ScanSettings struct {
	ID             string                     `json:"id,omitempty"`
	ResourceTypeID string                     `json:"resourceTypeId,omitempty"`
	ScriptName     string                     `json:"scriptName"`
	TimeoutLimit   int                        `json:"timeoutLimit"`
	IsLocal        bool                       `json:"isLocal"`
	InlineFile     bool                       `json:"inlineFile"`
	EditorType     string                     `json:"editorType,omitempty"`
	Variables      []RotationTemplateVariable `json:"variables"`
	PresignedURL   string                     `json:"presignedUrl,omitempty"`
	CreatedOn      string                     `json:"createdOn,omitempty"`
	CreatedBy      string                     `json:"createdBy,omitempty"`
	UpdatedOn      string                     `json:"updatedOn,omitempty"`
	UpdatedBy      string                     `json:"updatedBy,omitempty"`
}

// ScheduleScanTaskService - a resource type's scan task-service record. Unlike
// RotationTemplate/ScanSettings, this is auto-created by the API on the first
// ScheduleScanTask created for a resource type - GetScheduleScanTaskService returns a
// distinguishable "not yet created" error (confirmed by capture: 400/E1004) before that
// point, see resource_manager_resource_type_schedule_scan.go's doc comment.
type ScheduleScanTaskService struct {
	TaskServiceID   string `json:"taskServiceId,omitempty"`
	Name            string `json:"name,omitempty"`
	TenantNamespace string `json:"tenantNamespace,omitempty"`
	AppID           string `json:"appId,omitempty"`
	ScanSourceType  string `json:"scanSourceType,omitempty"`
	Enabled         bool   `json:"enabled"`
	QueueID         string `json:"queueId,omitempty"`
	TaskType        string `json:"taskType,omitempty"`
}

// ScheduleScanTaskServiceStub - the static taskService object bundled into every
// create-task call. Every field here is a hardcoded constant confirmed by capture (not
// derived from user input) - see newScheduleScanTaskServiceStub.
type ScheduleScanTaskServiceStub struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	QueueID string `json:"queueId"`
}

// ScheduleScanTask - the create/update request shape for a single scheduled scan ("task")
// under a resource type's scan task-service. FrequencyInterval has no `omitempty`: an
// explicit `null` must be sent for Daily (confirmed by capture), so a nil pointer here is
// meaningful and must be marshaled as JSON null, not omitted. Properties has no `omitempty`
// either: an explicit `{}` clears all resource label filters (confirmed by capture) - a nil
// map would marshal as JSON null instead, which the API would treat as "no change".
type ScheduleScanTask struct {
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Properties        map[string][]string `json:"properties"`
	FrequencyType     string              `json:"frequencyType"`
	FrequencyInterval *int                `json:"frequencyInterval"`
	StartTime         string              `json:"startTime"`
}

// ScheduleScanTaskCreateRequest - the create call's body: the static taskService stub plus
// the task payload, bundled together. Confirmed by capture: the API (re)creates the
// taskService idempotently (a no-op after the first call) and always creates a new task in
// the same call - there's no separate "just bootstrap the service" endpoint.
type ScheduleScanTaskCreateRequest struct {
	TaskService ScheduleScanTaskServiceStub `json:"taskService"`
	Task        ScheduleScanTask            `json:"task"`
}

// ScheduleScanTaskDetail - the response/list shape for a scheduled scan task. StartTime
// here is an [hour, minute] pair, unlike the "HH:MM" string ScheduleScanTask sends on
// write. Modified is 0 (JSON null) until the task's first update.
type ScheduleScanTaskDetail struct {
	TaskID            string              `json:"taskId,omitempty"`
	TaskServiceID     string              `json:"taskServiceId,omitempty"`
	TenantNamespace   string              `json:"tenantNamespace,omitempty"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Properties        map[string][]string `json:"properties"`
	StartTime         []int               `json:"startTime"`
	FrequencyType     string              `json:"frequencyType"`
	FrequencyInterval *int                `json:"frequencyInterval"`
	CreatedBy         string              `json:"createdBy,omitempty"`
	Created           int64               `json:"created,omitempty"`
	Modified          int64               `json:"modified,omitempty"`
	ModifiedBy        string              `json:"modifiedBy,omitempty"`
	NextRun           int64               `json:"nextRun,omitempty"`
}

// ResourceTypePermission - Model for resource type permissions
type ResourceTypePermission struct {
	PermissionID      string        `json:"permissionId,omitempty"`
	Name              string        `json:"name"`
	Description       string        `json:"description,omitempty"`
	ResourceTypeID    string        `json:"resourceTypeId"`
	ResourceTypeName  string        `json:"resourceTypeName,omitempty"`
	IsDraft           bool          `json:"isDraft"`
	Version           string        `json:"version,omitempty"`
	CheckinTimeLimit  int           `json:"checkinTimeLimit,omitempty"`
	CheckoutTimeLimit int           `json:"checkoutTimeLimit,omitempty"`
	ShowOrigCreds     bool          `json:"showOrigCreds,omitempty"`
	InlineFileExists  bool          `json:"inlineFileExists,omitempty"`
	ResponseTemplates []interface{} `json:"responseTemplates,omitempty"`
	CheckinFileName   string        `json:"checkinFileName,omitempty"`
	CheckoutFileName  string        `json:"checkoutFileName,omitempty"`
	Variables         []interface{} `json:"variables,omitempty"`
}

type ResourceTypePermissiosUploadUrls struct {
	CheckInUrl  string `json:"checkinURL,omitempty"`
	CheckOutUrl string `json:"checkoutURL,omitempty"`
}

// Resource Label Resource - godoc
type ResourceLabel struct {
	LabelId     string               `json:"keyId,omitempty"`
	Name        string               `json:"keyName,omitempty"`
	Description string               `json:"description,omitempty"`
	Internal    bool                 `json:"internal,omitempty"`
	LabelColor  string               `json:"labelColor,omitempty"`
	Values      []ResourceLabelValue `json:"values,omitempty"`
}

type ResourceLabelValue struct {
	ValueId     string `json:"valueId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedBy   int    `json:"createdBy,omitempty"`
	UpdatedBy   int    `json:"updatedBy,omitempty"`
	CreatedOn   string `json:"createdOn,omitempty"`
	UpdatedOn   string `json:"updatedOn,omitempty"`
}

// ResourceManagerProfile - godoc
type ResourceManagerProfile struct {
	ProfileId                     string              `json:"profileId,omitempty"`
	Name                          string              `json:"name,omitempty"`
	Description                   *string             `json:"description,omitempty"`
	ExpirationDuration            int                 `json:"expirationDuration,omitempty"`
	Status                        string              `json:"status,omitempty"`
	Associations                  map[string][]string `json:"associations,omitempty"`
	ResourceLabelColorMap         map[string]string   `json:"resourceLabelColorMap,omitempty"`
	// DelegationEnabled must NOT use omitempty: Go omits false for bool with omitempty,
	// which would prevent the PATCH from ever setting delegation to false.
	DelegationEnabled             bool                `json:"delegationEnabled"`
	PolicyOrderingEnabled         bool                `json:"policyOrderingEnabled,omitempty"`
	ExclusiveCheckout             bool                `json:"exclusiveCheckout"`
	Extendable                    bool                `json:"extendable"`
	NotificationPriorToExpiration *int64              `json:"notificationPriorToExpiration,omitempty"`
	ExtensionDuration             *int64              `json:"extensionDuration,omitempty"`
	ExtensionLimit                interface{}         `json:"extensionLimit,omitempty"`
}

// ResourceManagerProfilePolicy - godoc
type ResourceManagerProfilePolicy struct {
	ProfileID      string              `json:"profileId,omitempty"`
	PolicyID       string              `json:"id,omitempty"`
	Name           string              `json:"name,omitempty"`
	Description    string              `json:"description"`
	Condition      string              `json:"condition"`
	Members        interface{}         `json:"members"`
	Consumer       string              `json:"consumer"`
	AccessType     string              `json:"accessType"`
	IsActive       bool                `json:"isActive"`
	IsDraft        bool                `json:"isDraft"`
	IsReadOnly     bool                `json:"isReadOnly"`
	ResourceLabels map[string][]string `json:"resourceLabels"`
	Order          int                 `json:"order,omitempty"`
}

// Resource Manager Profile Permission - godoc
type ResourceManagerProfilePermission struct {
	ProfilID         string                   `json:"profileID,omitempty"`
	PermissionID     string                   `json:"permissionId,omitempty"`
	PermissionName   string                   `json:"permissionName,omitempty"`
	Description      string                   `json:"description,omitempty"`
	Version          string                   `json:"version,omitempty"`
	ResourceTypeId   string                   `json:"resourceTypeId,omitempty"`
	ResourceTypeName string                   `json:"resourceTypeName,omitempty"`
	Variables        []map[string]interface{} `json:"variables,omitempty"`
}

// Resource Manager Permissions - godoc
type ResourceManagerPermissions struct {
	Permissions []map[string]interface{} `json:"data"`
}

// TagOwnerEntity - an entity (user or tag) that owns a tag
type TagOwnerEntity struct {
	RelatedEntityID   string `json:"relatedEntityId,omitempty"`
	RelatedEntityName string `json:"relatedEntityName,omitempty"`
	RelatedEntityType string `json:"relatedEntityType"`
}

// TagOwnerRelationships - the relationships block containing owners
type TagOwnerRelationships struct {
	Owners []TagOwnerEntity `json:"owners"`
}

// TagWithOwners - tag with owner relationships; used for GET /api/user-tags/{id} and PATCH /api/user-tags
type TagWithOwners struct {
	TagID         string                `json:"userTagId"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	Relationships TagOwnerRelationships `json:"relationships"`
}

// Server Access Resource - godoc
type ServerAccessResource struct {
	ResourceID                  string                   `json:"resourceId,omitempty"`
	Name                        string                   `json:"name"`
	Description                 string                   `json:"description"`
	ResourceType                ServerAccessResourceType `json:"resourceType"`
	ResourceTypeParameterValues map[string]string        `json:"paramValues"`
	ResourceLabels              map[string][]string      `json:"resourceLabels"`
}

// Server Access Resource Type - godoc
type ServerAccessResourceType struct {
	ResourceTypeID string `json:"id"`
	Name           string `json:"name"`
}

// Broker Pool - godoc
type BrokerPool struct {
	BrokerPoolID string `json:"brokerPoolId,omitempty"`
	Name         string `json:"brokerPoolName"`
	Description  string `json:"brokerPoolDesc,omitempty"`
	Count        int    `json:"brokerCount,omitempty"`
}

// Resource Manager Resource-Policy - godoc
type ResourceManagerResourcePolicy struct {
	PolicyID       string              `json:"id,omitempty"`
	Name           string              `json:"name,omitempty"`
	Description    string              `json:"description"`
	Condition      string              `json:"condition"`
	Members        interface{}         `json:"members"`
	Consumer       string              `json:"consumer"`
	AccessType     string              `json:"accessType"`
	AccessLevel    string              `json:"accessLevel"`
	IsActive       bool                `json:"isActive"`
	IsDraft        bool                `json:"isDraft"`
	IsReadOnly     bool                `json:"isReadOnly"`
	ResourceLabels map[string][]string `json:"resourceLabels"`
}
