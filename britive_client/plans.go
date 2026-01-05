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

// Resources
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

type ProfilePermissionPlan struct {
	ID             types.String `tfsdk:"id"`
	AppName        types.String `tfsdk:"app_name"`
	ProfileID      types.String `tfsdk:"profile_id"`
	ProfileName    types.String `tfsdk:"profile_name"`
	PermissionName types.String `tfsdk:"permission_name"`
	PermissionType types.String `tfsdk:"permission_type"`
}
