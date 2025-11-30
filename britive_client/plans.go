package britive_client

import "github.com/hashicorp/terraform-plugin-framework/types"

type BritiveProviderModel struct {
	Tenant types.String `tfsdk:"tenant"`
	Token  types.String `tfsdk:"token"`
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
