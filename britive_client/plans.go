package britive_client

import "github.com/hashicorp/terraform-plugin-framework/types"

type BritiveProviderModel struct {
	Tenant     types.String `tfsdk:"tenant"`
	Token      types.String `tfsdk:"token"`
	ConfigPath types.String `tfsdk:"config_path"`
}

// Datasources
type DataSourceApplicationPlan struct {
	ID                       types.String                           `tfsdk:"id"`
	Name                     types.String                           `tfsdk:"name"`
	EnvironmentIDs           types.Set                              `tfsdk:"environment_ids"`
	EnvironmentGroupIDs      types.Set                              `tfsdk:"environment_group_ids"`
	EnvironmentIDsNames      []DataSourceEnvironmentIDNamePlan      `tfsdk:"environment_ids_names"`
	EnvironmentGroupIDsNames []DataSourceEnvironmentGroupIDNamePlan `tfsdk:"environment_group_ids_names"`
}

type DataSourceEnvironmentIDNamePlan struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type DataSourceEnvironmentGroupIDNamePlan struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type DataSourceAllConnectionsPlan struct {
	ID          types.String               `tfsdk:"id"`
	SettingType types.String               `tfsdk:"setting_type"`
	Connections []DataSourceConnectionPlan `tfsdk:"connections"`
}

type DataSourceConnectionPlan struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	AuthType types.String `tfsdk:"auth_type"`
}

type DataSourceSingleConnectionPlan struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	SettingType types.String `tfsdk:"setting_type"`
	AuthType    types.String `tfsdk:"auth_type"`
}

type DataSourceEscalationPolicyPlan struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ImConnectionID types.String `tfsdk:"im_connection_id"`
}

type DataSourceSupportedConstraintsPlan struct {
	ID              types.String `tfsdk:"id"`
	ProfileID       types.String `tfsdk:"profile_id"`
	PermissionName  types.String `tfsdk:"permission_name"`
	PermissionType  types.String `tfsdk:"permission_type"`
	ConstraintTypes types.Set    `tfsdk:"constraint_types"`
}

type DataSourceIdentityProviderPlan struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type DataResourceManagerProfilePermissionsPlan struct {
	ID          types.String                        `tfsdk:"id"`
	ProfileID   types.String                        `tfsdk:"profile_id"`
	Permissions []DataResourceManagerPermissionPlan `tfsdk:"permissions"`
}

type DataResourceManagerPermissionPlan struct {
	Name         types.String `tfsdk:"name"`
	PermissionID types.String `tfsdk:"permission_id"`
	Version      []string     `tfsdk:"version"`
}

// Resources
type TagPlan struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Disabled           types.Bool   `tfsdk:"disabled"`
	IdentityProviderID types.String `tfsdk:"identity_provider_id"`
	External           types.Bool   `tfsdk:"external"`
}

type TagMemberPlan struct {
	ID       types.String `tfsdk:"id"`
	TagID    types.String `tfsdk:"tag_id"`
	TagName  types.String `tfsdk:"tag_name"`
	Username types.String `tfsdk:"username"`
}

type PermissionPlan struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Consumer    types.String `tfsdk:"consumer"`
	Resources   types.Set    `tfsdk:"resources"`
	Actions     types.Set    `tfsdk:"actions"`
}

type RolePlan struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.String `tfsdk:"permissions"`
}

type PolicyPlan struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	IsDraft     types.Bool   `tfsdk:"is_draft"`
	IsReadOnly  types.Bool   `tfsdk:"is_read_only"`
	AccessType  types.String `tfsdk:"access_type"`
	Members     types.String `tfsdk:"members"`
	Condition   types.String `tfsdk:"condition"`
	Permissions types.String `tfsdk:"permissions"`
	Roles       types.String `tfsdk:"roles"`
}

type ConstraintPlan struct {
	ID             types.String `tfsdk:"id"`
	ProfileID      types.String `tfsdk:"profile_id"`
	PermissionName types.String `tfsdk:"permission_name"`
	PermissionType types.String `tfsdk:"permission_type"`
	ConstraintType types.String `tfsdk:"constraint_type"`
	Name           types.String `tfsdk:"name"`
	Title          types.String `tfsdk:"title"`
	Expression     types.String `tfsdk:"expression"`
	Description    types.String `tfsdk:"description"`
}

type ApplicationPlan struct {
	ID                           types.String `tfsdk:"id"`
	ApplicationType              types.String `tfsdk:"application_type"`
	Version                      types.String `tfsdk:"version"`
	CatalogAppID                 types.Int64  `tfsdk:"catalog_app_id"`
	EntityRootEnvironmentGroupID types.String `tfsdk:"entity_root_environment_group_id"`
	Properties                   types.Set    `tfsdk:"properties"`
	SensitiveProperties          types.Set    `tfsdk:"sensitive_properties"`
	UserAccountMappings          types.Set    `tfsdk:"user_account_mappings"`
}

type PropertyPlan struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type SensitivePropertyPlan struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type UserAccountMappingPlan struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type EntityGroupPlan struct {
	ID                types.String `tfsdk:"id"`
	EntityID          types.String `tfsdk:"entity_id"`
	ApplicationID     types.String `tfsdk:"application_id"`
	EntityName        types.String `tfsdk:"entity_name"`
	EntityDescription types.String `tfsdk:"entity_description"`
	ParentID          types.String `tfsdk:"parent_id"`
}

type EntityEnvironmentPlan struct {
	ID                  types.String `tfsdk:"id"`
	EntityID            types.String `tfsdk:"entity_id"`
	ApplicationID       types.String `tfsdk:"application_id"`
	ParentGroupID       types.String `tfsdk:"parent_group_id"`
	Properties          types.Set    `tfsdk:"properties"`
	SensitiveProperties types.Set    `tfsdk:"sensitive_properties"`
}

type AdvancedSettingsPlan struct {
	ID                    types.String `tfsdk:"id"`
	ResourceID            types.String `tfsdk:"resource_id"`
	ResourceType          types.String `tfsdk:"resource_type"`
	JustificationSettings types.Set    `tfsdk:"justification_settings"`
	Itsm                  types.Set    `tfsdk:"itsm"`
	Im                    types.Set    `tfsdk:"im"`
}

type JustificationSettingsPlan struct {
	JustificationID         types.String `tfsdk:"justification_id"`
	IsJustificationRequired types.Bool   `tfsdk:"is_justification_required"`
	JustificationRegex      types.String `tfsdk:"justification_regex"`
}

type ItsmPlan struct {
	ItsmID             types.String `tfsdk:"itsm_id"`
	ConnectionID       types.String `tfsdk:"connection_id"`
	ConnectionType     types.String `tfsdk:"connection_type"`
	IsItsmEnabled      types.Bool   `tfsdk:"is_itsm_enabled"`
	ItsmFilterCriteria types.Set    `tfsdk:"itsm_filter_criteria"`
}

type ItsmFilterCriteriaPlan struct {
	SupportedTicketType types.String `tfsdk:"supported_ticket_type"`
	Filter              types.String `tfsdk:"filter"`
}

type ImPlan struct {
	ImID                  types.String `tfsdk:"im_id"`
	ConnectionID          types.String `tfsdk:"connection_id"`
	ConnectionType        types.String `tfsdk:"connection_type"`
	IsAutoApprovalEnabled types.Bool   `tfsdk:"is_auto_approval_enabled"`
	EscalationPolicies    types.Set    `tfsdk:"escalation_policies"`
}

type ProfilePlan struct {
	ID                            types.String `tfsdk:"id"`
	AppContainerID                types.String `tfsdk:"app_container_id"`
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	Disabled                      types.Bool   `tfsdk:"disabled"`
	Associations                  types.Set    `tfsdk:"associations"`
	ExpirationDuration            types.String `tfsdk:"expiration_duration"`
	Extendable                    types.Bool   `tfsdk:"extendable"`
	NotificationPriorToExpiration types.String `tfsdk:"notification_prior_to_expiration"`
	ExtensionDuration             types.String `tfsdk:"extension_duration"`
	ExtensionLimit                types.Int64  `tfsdk:"extension_limit"`
	DestinationUrl                types.String `tfsdk:"destination_url"`
	AllowImpersonation            types.Bool   `tfsdk:"allow_impersonation"`
}

type ProfileAssociationPlan struct {
	Type       types.String `tfsdk:"type"`
	Value      types.String `tfsdk:"value"`
	ParentName types.String `tfsdk:"parent_name"`
}

type ProfilePermissionPlan struct {
	ID             types.String `tfsdk:"id"`
	AppName        types.String `tfsdk:"app_name"`
	ProfileID      types.String `tfsdk:"profile_id"`
	ProfileName    types.String `tfsdk:"profile_name"`
	PermissionName types.String `tfsdk:"permission_name"`
	PermissionType types.String `tfsdk:"permission_type"`
}

type ProfileAdditionalSettingsPlan struct {
	ID                         types.String `tfsdk:"id"`
	ProfileID                  types.String `tfsdk:"profile_id"`
	UserAppCredentialType      types.Bool   `tfsdk:"use_app_credential_type"`
	ConsoleAccess              types.Bool   `tfsdk:"console_access"`
	ProgrammaticAccess         types.Bool   `tfsdk:"programmatic_access"`
	ProjectIDForServiceAccount types.String `tfsdk:"project_id_for_service_account"`
}

type ProfilePolicyPlan struct {
	ID           types.String `tfsdk:"id"`
	ProfileID    types.String `tfsdk:"profile_id"`
	PolicyName   types.String `tfsdk:"policy_name"`
	Description  types.String `tfsdk:"description"`
	IsActive     types.Bool   `tfsdk:"is_active"`
	IsDraft      types.Bool   `tfsdk:"is_draft"`
	IsReadOnly   types.Bool   `tfsdk:"is_read_only"`
	Consumer     types.String `tfsdk:"consumer"`
	AccessType   types.String `tfsdk:"access_type"`
	Members      types.String `tfsdk:"members"`
	Condition    types.String `tfsdk:"condition"`
	Associations types.Set    `tfsdk:"associations"`
}

type PolicyAssociationPlan struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

type ProfilePolicyPrioritizationPlan struct {
	ID                    types.String `tfsdk:"id"`
	ProfileID             types.String `tfsdk:"profile_id"`
	PolicyPriorityEnabled types.Bool   `tfsdk:"policy_priority_enabled"`
	PolicyPriority        types.Set    `tfsdk:"policy_priority"`
}

type PolicyPriorityPlan struct {
	ID       types.String `tfsdk:"id"`
	Priority types.Int64  `tfsdk:"priority"`
}

type ProfileSessionAttributePlan struct {
	ID             types.String `tfsdk:"id"`
	AppName        types.String `tfsdk:"app_name"`
	ProfileID      types.String `tfsdk:"profile_id"`
	ProfileName    types.String `tfsdk:"profile_name"`
	AttributeName  types.String `tfsdk:"attribute_name"`
	AttributeType  types.String `tfsdk:"attribute_type"`
	AttributeValue types.String `tfsdk:"attribute_value"`
	MappingName    types.String `tfsdk:"mapping_name"`
	Transitive     types.Bool   `tfsdk:"transitive"`
}

type ResourceManagerResourceLabelPlan struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Internal    types.Bool   `tfsdk:"internal"`
	LabelColor  types.String `tfsdk:"label_color"`
	Values      types.Set    `tfsdk:"values"`
}

type ResourceManagerResourceLabelValuePlan struct {
	ValueID     types.String `tfsdk:"value_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type ResourceManagerResponseTemplatePlan struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	IsConsoleAccessEnabled types.Bool   `tfsdk:"is_console_access_enabled"`
	ShowOnUI               types.Bool   `tfsdk:"show_on_ui"`
	TemplateData           types.String `tfsdk:"template_data"`
	TemplateID             types.String `tfsdk:"template_id"`
}

type ResourceManagerResourceTypePlan struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Icon        types.String `tfsdk:"icon"`
	Parameters  types.Set    `tfsdk:"parameters"`
}

type ResourceManagerResourceTypeParameterPlan struct {
	Parametername types.String `tfsdk:"param_name"`
	ParameterType types.String `tfsdk:"param_type"`
	IsMandatory   types.Bool   `tfsdk:"is_mandatory"`
}

type ResourceManagerResourceTypePermissionPlan struct {
	ID                   types.String `tfsdk:"id"`
	PermissionID         types.String `tfsdk:"permission_id"`
	Name                 types.String `tfsdk:"name"`
	Version              types.String `tfsdk:"version"`
	ResourceTypeID       types.String `tfsdk:"resource_type_id"`
	Description          types.String `tfsdk:"description"`
	CheckinTimeLimit     types.Int64  `tfsdk:"checkin_time_limit"`
	CheckoutTimeLimit    types.Int64  `tfsdk:"checkout_time_limit"`
	IsDraft              types.Bool   `tfsdk:"is_draft"`
	InlineFileExists     types.Bool   `tfsdk:"inline_file_exists"`
	ResponseTemplates    types.Set    `tfsdk:"response_templates"`
	ShowOrigCreds        types.Bool   `tfsdk:"show_orig_creds"`
	CheckinCodeFile      types.String `tfsdk:"checkin_code_file"`
	CheckoutCodeFile     types.String `tfsdk:"checkout_code_file"`
	CheckinCodeFileHash  types.String `tfsdk:"checkin_code_file_hash"`
	CheckoutCodeFileHash types.String `tfsdk:"checkout_code_file_hash"`
	CheckinCode          types.String `tfsdk:"checkin_code"`
	CheckoutCode         types.String `tfsdk:"checkout_code"`
	CodeLanguage         types.String `tfsdk:"code_language"`
	CheckinFileName      types.String `tfsdk:"checkin_file_name"`
	CheckoutFileName     types.String `tfsdk:"checkout_file_name"`
	Variables            types.Set    `tfsdk:"variables"`
}

type ResourceManagerResourcePlan struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ResourceType    types.String `tfsdk:"resource_type"`
	ResourceTypeID  types.String `tfsdk:"resource_type_id"`
	ParameterValues types.Map    `tfsdk:"parameter_values"`
	ResourceLabels  types.Map    `tfsdk:"resource_labels"`
}

type ResourceManagerResourceBrokerPoolsPlan struct {
	ID          types.String `tfsdk:"id"`
	ResourceID  types.String `tfsdk:"resource_id"`
	BrokerPools types.Set    `tfsdk:"broker_pools"`
}
