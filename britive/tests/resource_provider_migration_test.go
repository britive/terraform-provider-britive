package tests

// resource_provider_migration_test.go — Provider migration tests (v2.2.9 → v3.0.0)
//
// Each test follows the 2-step pattern:
//   Step 1: Create the resource with v2.2.9 (SDKv2, Protocol v5) downloaded from the registry
//   Step 2: Plan-only with v3.0.0 (Plugin Framework, Protocol v6) using the local build
//
// Terraform calls UpgradeResourceState automatically between steps to translate
// the SDKv2 flatmap state format into Plugin Framework JSON. The PlanOnly step
// verifies the translation is transparent — no resource replacement or unexpected
// updates appear in the plan.
//
// Design notes:
//   - britive_tag is already covered by TestBritiveTagProviderMigration in
//     resource_lifecycle_test.go and is therefore excluded here.
//   - britive_profile "associations" changed HCL syntax between v2.2.9 (SDKv2
//     TypeSet assignment syntax) and v3.0.0 (SetNestedBlock syntax). All profiles
//     in this file are created WITHOUT associations to remain HCL-compatible with
//     both versions.
//   - britive_escalation_policy data source requires an IM-type connection that
//     is not guaranteed in all test environments; it is excluded.
//   - All configs are prefixed with migrationProviderBlock() (defined in
//     resource_lifecycle_test.go) to supply a full https:// tenant URL required
//     by v2.2.9.

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// migrationExternalProviders is the ExternalProvider map used for Step 1 of
// every migration test in this file.
var migrationExternalProviders = map[string]resource.ExternalProvider{
	"britive": {
		Source:            "britive/britive",
		VersionConstraint: "= 2.2.9",
	},
}

// runMigrationTest executes the canonical 2-step migration pattern:
// Step 1 creates the resource(s) using the v2.2.9 registry binary;
// Step 2 plans with the local v3.0.0 binary and asserts an empty plan.
func runMigrationTest(t *testing.T, config string) {
	t.Helper()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheckFramework(t) },
		Steps: []resource.TestStep{
			// Step 1 — create with v2.2.9 (SDKv2, Protocol v5)
			{
				ExternalProviders: migrationExternalProviders,
				Config:            config,
			},
			// Step 2 — plan-only with v3.0.0 (Plugin Framework, Protocol v6)
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Private config helpers used only in migration tests
// ─────────────────────────────────────────────────────────────────────────────

// migrationProfileNoAssocConfig produces a provider block + application data
// source + profile resource WITHOUT the "associations" field. Both v2.2.9 and
// v3.0.0 accept a profile with no associations; the HCL is identical in both.
func migrationProfileNoAssocConfig(appName, profileName string) string {
	return migrationProviderBlock() + fmt.Sprintf(`
data "britive_application" "app" {
  name = %q
}

resource "britive_profile" "migration" {
  app_container_id    = data.britive_application.app.id
  name                = %q
  expiration_duration = "25m0s"
}
`, appName, profileName)
}

// ─────────────────────────────────────────────────────────────────────────────
// Data source migration tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBritiveIdentityProviderDataSourceMigration(t *testing.T) {
	config := migrationProviderBlock() + `
data "britive_identity_provider" "migration" {
  name = "Britive"
}
`
	runMigrationTest(t, config)
}

func TestBritiveApplicationDataSourceMigration(t *testing.T) {
	config := migrationProviderBlock() + `
data "britive_application" "migration" {
  name = "DO NOT DELETE - Azure TF Plugin"
}
`
	runMigrationTest(t, config)
}

func TestBritiveSupportedConstraintsDataSourceMigration(t *testing.T) {
	// Uses a profile without associations so the HCL is compatible with both versions.
	config := migrationProfileNoAssocConfig("DO NOT DELETE - GCP TF Plugin", "AT - Migration Supported Constraints Test") + `
resource "britive_profile_permission" "migration_perm" {
  profile_id      = britive_profile.migration.id
  permission_name = "BigQuery Data Owner"
  permission_type = "role"
}

data "britive_supported_constraints" "migration" {
  profile_id      = britive_profile.migration.id
  permission_name = britive_profile_permission.migration_perm.permission_name
  permission_type = britive_profile_permission.migration_perm.permission_type
}
`
	runMigrationTest(t, config)
}

func TestBritiveConnectionDataSourceMigration(t *testing.T) {
	config := migrationProviderBlock() + `
data "britive_connection" "migration" {
  name = "TF_ACCEPTANCE_TEST_ITSM_DO_NOT_DELETE"
}
`
	runMigrationTest(t, config)
}

func TestBritiveAllConnectionsDataSourceMigration(t *testing.T) {
	config := migrationProviderBlock() + `
data "britive_all_connections" "migration" {}
`
	runMigrationTest(t, config)
}

func TestBritiveRMProfilePermissionsDataSourceMigration(t *testing.T) {
	// Use a minimal RM profile without any permission association to avoid
	// stale state from previous test runs leaving permissions associated.
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_label" "ds_migration" {
  name        = "AT-Britive_Migration_DS_RMPerms_Label"
  description = "AT-Britive_Migration_DS_RMPerms_Label_Description"
  label_color = "#aabbcc"
  values {
    name        = "Production"
    description = "Production Desc"
  }
}

resource "britive_resource_manager_profile" "ds_migration" {
  name                = "AT-Britive_Migration_DS_RMPerms_Profile"
  description         = "AT-Britive_Migration_DS_RMPerms_Profile_Description"
  expiration_duration = 10800000
  allow_impersonation = true
  associations {
    label_key = britive_resource_manager_resource_label.ds_migration.name
    values    = ["Production"]
  }
}

data "britive_resource_manager_profile_permissions" "migration" {
  profile_id = britive_resource_manager_profile.ds_migration.id
}
`
	runMigrationTest(t, config)
}

// ─────────────────────────────────────────────────────────────────────────────
// Core resource migration tests
// ─────────────────────────────────────────────────────────────────────────────

// Note: britive_tag is already covered by TestBritiveTagProviderMigration in
// resource_lifecycle_test.go.

func TestBritivePermissionMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_permission" "migration" {
  name        = "AT - Permission Migration Test"
  description = "AT - Permission Migration Test Description"
  consumer    = "secretmanager"
  resources   = ["*"]
  actions     = ["authz.policy.list", "authz.policy.read", "sm.secret.read"]
}
`
	runMigrationTest(t, config)
}

func TestBritiveRoleMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_permission" "migration_perm" {
  name        = "AT - Role Migration Test Perm"
  description = "AT - Role Migration Test Perm Description"
  consumer    = "secretmanager"
  resources   = ["*"]
  actions     = ["authz.policy.list", "sm.secret.read"]
}

resource "britive_role" "migration" {
  name        = "AT - Role Migration Test"
  description = "AT - Role Migration Test Description"
  permissions = jsonencode([{ name = britive_permission.migration_perm.name }])
}
`
	runMigrationTest(t, config)
}

func TestBritivePolicyMigration(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	config := migrationProviderBlock() + fmt.Sprintf(`
resource "britive_permission" "migration_perm" {
  name        = "AT - Policy Migration Test Perm"
  description = "AT - Policy Migration Test Perm Description"
  consumer    = "secretmanager"
  resources   = ["*"]
  actions     = ["authz.policy.list", "sm.secret.read"]
}

resource "britive_role" "migration_role" {
  name        = "AT - Policy Migration Test Role"
  description = "AT - Policy Migration Test Role Description"
  permissions = jsonencode([{ name = britive_permission.migration_perm.name }])
}

resource "britive_policy" "migration" {
  name        = "AT - Policy Migration Test"
  description = "AT - Policy Migration Test Description"
  access_type = "Allow"
  is_active   = true
  is_draft    = false
  roles       = jsonencode([{ name = britive_role.migration_role.name }])
  members     = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" },
    ]
  })
  condition = jsonencode({
    approval = {
      approvers = {
        userIds = ["britiveprovideracceptancetest"]
      }
      notificationMedium = "Email"
      timeToApprove      = 30
      isValidForInDays   = false
      validFor           = 120
    }
    ipAddress    = "192.162.0.0/16,10.10.0.10"
    timeOfAccess = {
      "dateSchedule": {
        "fromDate": %q,
        "toDate":   %q,
        "timezone": "Asia/Calcutta"
      }
    }
  })
}
`, timeOfAccessFrom, timeOfAccessTo)
	runMigrationTest(t, config)
}

func TestBritiveTagMemberMigration(t *testing.T) {
	config := migrationProviderBlock() + `
data "britive_identity_provider" "migration" {
  name = "Britive"
}

resource "britive_tag" "migration" {
  name                 = "AT - Tag Member Migration Test"
  description          = "AT - Tag Member Migration Test Description"
  identity_provider_id = data.britive_identity_provider.migration.id
}

resource "britive_tag_member" "migration" {
  tag_id   = britive_tag.migration.id
  username = "britiveprovideracceptancetest"
}
`
	runMigrationTest(t, config)
}

func TestBritiveEntityGroupMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_application" "migration" {
  application_type = "Snowflake Standalone"
  version          = "1.0"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "AT - Migration Entity Group App"
  }
  properties {
    name  = "description"
    value = "AT - Migration Entity Group App Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1500
  }
}

resource "britive_entity_group" "migration" {
  application_id     = britive_application.migration.id
  entity_name        = "AT - Entity Group Migration Test"
  entity_description = "AT - Entity Group Migration Test Description"
  parent_id          = britive_application.migration.entity_root_environment_group_id
}
`
	runMigrationTest(t, config)
}

func TestBritiveEntityEnvironmentMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_application" "migration_env" {
  application_type = "Snowflake Standalone"
  version          = "1.0"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "AT - Migration Entity Env App"
  }
  properties {
    name  = "description"
    value = "AT - Migration Entity Env App Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1500
  }
}

resource "britive_entity_environment" "migration" {
  application_id  = britive_application.migration_env.id
  parent_group_id = britive_application.migration_env.entity_root_environment_group_id
  properties {
    name  = "displayName"
    value = "AT - Entity Env Migration Test"
  }
  properties {
    name  = "description"
    value = "AT - Entity Env Migration Test Description"
  }
  properties {
    name  = "accountId"
    value = "accId"
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveApplicationMigration(t *testing.T) {
	// Tests a simple Okta application — minimal fields, compatible with both versions.
	config := migrationProviderBlock() + `
resource "britive_application" "migration" {
  application_type = "okta"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "AT - Okta Migration App"
  }
  properties {
    name  = "description"
    value = "AT - Okta Migration App Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1000
  }
}
`
	runMigrationTest(t, config)
}

// TestBritiveProfileMigration verifies that a profile created without
// associations migrates cleanly. Profiles WITH associations are excluded
// because the "associations" field changed HCL syntax between v2.2.9 and v3.0.0
// (see comment at the top of this file and MIGRATION.md §16).
func TestBritiveProfileMigration(t *testing.T) {
	config := migrationProviderBlock() + `
data "britive_application" "migration" {
  name = "DO NOT DELETE - Azure TF Plugin"
}

resource "britive_profile" "migration" {
  app_container_id    = data.britive_application.migration.id
  name                = "AT - Profile Migration Test"
  description         = "AT - Profile Migration Test Description"
  expiration_duration = "25m0s"
}
`
	runMigrationTest(t, config)
}

func TestBritiveProfilePermissionMigration(t *testing.T) {
	config := migrationProfileNoAssocConfig("DO NOT DELETE - Azure TF Plugin", "AT - Profile Permission Migration Test") + `
resource "britive_profile_permission" "migration" {
  profile_id      = britive_profile.migration.id
  permission_name = "Application Developer"
  permission_type = "role"
}
`
	runMigrationTest(t, config)
}

func TestBritiveProfileSessionAttributeMigration(t *testing.T) {
	config := migrationProfileNoAssocConfig("DO NOT DELETE - AWS TF Plugin", "AT - Profile Session Attr Migration Test") + `
resource "britive_profile_session_attribute" "migration" {
  profile_id   = britive_profile.migration.id
  attribute_name = "Date Of Birth"
  mapping_name   = "dob"
  transitive     = true
}
`
	runMigrationTest(t, config)
}

func TestBritiveProfileAdditionalSettingsMigration(t *testing.T) {
	config := migrationProfileNoAssocConfig("DO NOT DELETE - GCP TF Plugin", "AT - Profile Addl Settings Migration Test") + `
resource "britive_profile_additional_settings" "migration" {
  profile_id              = britive_profile.migration.id
  use_app_credential_type = false
  console_access          = true
  programmatic_access     = false
}
`
	runMigrationTest(t, config)
}

func TestBritiveConstraintMigration(t *testing.T) {
	// Only test the "condition" constraint type. The "bigquery.datasets" type requires
	// the profile to have scopes (environments) configured, but adding associations
	// is excluded from migration tests because the HCL syntax changed between versions.
	constraintExpression := "request.time < timestamp('" + time.Now().AddDate(0, 0, 2).Format("2006-01-02T15:04:05Z07:00") + "')"
	config := migrationProfileNoAssocConfig("DO NOT DELETE - GCP TF Plugin", "AT - Constraint Migration Test") + fmt.Sprintf(`
resource "britive_profile_permission" "migration_cond" {
  profile_id      = britive_profile.migration.id
  permission_name = "Storage Admin"
  permission_type = "role"
}

resource "britive_constraint" "migration_condition" {
  profile_id      = britive_profile.migration.id
  permission_name = britive_profile_permission.migration_cond.permission_name
  permission_type = britive_profile_permission.migration_cond.permission_type
  constraint_type = "condition"
  title           = "ConditionConstraintMigration"
  description     = "Condition Constraint Migration Description"
  expression      = %q
}
`, constraintExpression)
	runMigrationTest(t, config)
}

func TestBritiveAdvancedSettingsMigration(t *testing.T) {
	// Tests APPLICATION-level advanced settings with minimal fields only.
	// Advanced settings schema changed significantly between v2.2.9 and v3.0.0
	// (v2.2.9 had flat attributes; v3.0.0 uses nested blocks), so we use only
	// the fields common to both: resource_type and resource_id.
	config := migrationProviderBlock() + `
data "britive_application" "migration" {
  name = "DO NOT DELETE - AWS TF Plugin"
}

resource "britive_advanced_settings" "migration_app" {
  resource_type = "APPLICATION"
  resource_id   = data.britive_application.migration.id
}
`
	runMigrationTest(t, config)
}

func TestBritiveProfilePolicyMigration(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	config := migrationProfileNoAssocConfig("DO NOT DELETE - AWS TF Plugin", "AT - Profile Policy Migration Test") + fmt.Sprintf(`
resource "britive_profile_policy" "migration" {
  policy_name  = "AT - Profile Policy Migration Test Policy"
  description  = "AT - Profile Policy Migration Test Description"
  profile_id   = britive_profile.migration.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" },
    ]
  })
  condition = jsonencode({
    approval = {
      approvers = {
        userIds = ["britiveprovideracceptancetest"]
      }
      notificationMedium = "Email"
      timeToApprove      = 30
      isValidForInDays   = false
      validFor           = 120
    }
    ipAddress    = "192.162.0.0/16,10.10.0.10"
    timeOfAccess = {
      "dateSchedule": {
        "fromDate": %q,
        "toDate":   %q,
        "timezone": "Asia/Calcutta"
      }
    }
  })
}
`, timeOfAccessFrom, timeOfAccessTo)
	runMigrationTest(t, config)
}

func TestBritiveProfilePolicyPrioritizationMigration(t *testing.T) {
	config := migrationProfileNoAssocConfig("DO NOT DELETE - AWS TF Plugin", "AT - Profile Policy Prio Migration Test") + `
resource "britive_profile_policy" "migration_0" {
  policy_name  = "AT - Profile Policy Prio Migration 0"
  description  = "AT - Profile Policy Prio Migration 0 Desc"
  profile_id   = britive_profile.migration.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
}

resource "britive_profile_policy" "migration_1" {
  policy_name  = "AT - Profile Policy Prio Migration 1"
  description  = "AT - Profile Policy Prio Migration 1 Desc"
  profile_id   = britive_profile.migration.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
}

resource "britive_profile_policy" "migration_2" {
  policy_name  = "AT - Profile Policy Prio Migration 2"
  description  = "AT - Profile Policy Prio Migration 2 Desc"
  profile_id   = britive_profile.migration.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
}

resource "britive_profile_policy_prioritization" "migration" {
  profile_id = britive_profile.migration.id

  policy_priority {
    id       = britive_profile_policy.migration_0.id
    priority = 0
  }

  policy_priority {
    id       = britive_profile_policy.migration_1.id
    priority = 1
  }

  policy_priority {
    id       = britive_profile_policy.migration_2.id
    priority = 2
  }
}
`
	runMigrationTest(t, config)
}

// ─────────────────────────────────────────────────────────────────────────────
// Resource Manager resource migration tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBritiveResponseTemplateMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_response_template" "migration" {
  name                      = "AT-Britive_Migration_Response_Template"
  description               = "AT-Britive_Migration_Response_Template_Description"
  is_console_access_enabled = true
  show_on_ui                = false
  template_data             = "The user {{YS}} has the {{admin}}."
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceTypeMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_type" "migration" {
  name        = "AT-Britive_Migration_Resource_Type"
  description = "AT-Britive_Migration_Resource_Type_Description"
  parameters {
    param_name   = "testfield1"
    param_type   = "password"
    is_mandatory = true
  }
  parameters {
    param_name   = "testfield2"
    param_type   = "string"
    is_mandatory = false
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceTypePermissionMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_type" "migration" {
  name        = "AT-Britive_Migration_TypePerm_ResType"
  description = "AT-Britive_Migration_TypePerm_ResType_Description"
  parameters {
    param_name   = "testfield1"
    param_type   = "string"
    is_mandatory = true
  }
}

resource "britive_resource_manager_response_template" "migration" {
  name                      = "AT-Britive_Migration_TypePerm_Template"
  description               = "AT-Britive_Migration_TypePerm_Template_Description"
  is_console_access_enabled = true
  show_on_ui                = false
  template_data             = "The user {{YS}} has the {{admin}}."
}

resource "britive_resource_manager_resource_type_permission" "migration" {
  name                = "AT-Britive_Migration_TypePerm_Perm"
  resource_type_id    = britive_resource_manager_resource_type.migration.id
  description         = "AT-Britive_Migration_TypePerm_Perm_Description"
  checkin_time_limit  = 180
  checkout_time_limit = 360
  is_draft            = false
  show_orig_creds     = true
  variables           = ["var1", "var2"]
  code_language       = "python"
  checkin_code        = "#!/bin/bash\necho done"
  checkout_code       = "#!/bin/bash\necho start"
  response_templates  = [britive_resource_manager_response_template.migration.name]
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceLabelMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_label" "migration" {
  name        = "AT-Britive_Migration_Resource_Label"
  description = "AT-Britive_Migration_Resource_Label_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
  values {
    name        = "Development"
    description = "Development Desc"
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceResourceMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_type" "migration" {
  name        = "AT-Britive_Migration_Resource_ResType"
  description = "AT-Britive_Migration_Resource_ResType_Description"
  parameters {
    param_name   = "field1"
    param_type   = "string"
    is_mandatory = true
  }
}

resource "britive_resource_manager_resource_label" "migration" {
  name        = "AT-Britive_Migration_Resource_Label"
  description = "AT-Britive_Migration_Resource_Label_Description"
  label_color = "#1a2b3c"
  values {
    name        = "us-east-1"
    description = "US East 1"
  }
}

resource "britive_resource_manager_resource" "migration" {
  name          = "AT-Britive_Migration_Resource"
  description   = "AT-Britive_Migration_Resource_Description"
  resource_type = britive_resource_manager_resource_type.migration.name
  parameter_values = {
    "field1" = "value1"
  }
  resource_labels = {
    "${britive_resource_manager_resource_label.migration.name}" = "us-east-1"
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceResourcePolicyMigration(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	config := migrationProviderBlock() + fmt.Sprintf(`
resource "britive_resource_manager_resource_label" "migration" {
  name        = "AT-Britive_Migration_ResPolicy_Label"
  description = "AT-Britive_Migration_ResPolicy_Label_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
}

resource "britive_resource_manager_resource_policy" "migration" {
  policy_name  = "AT-Britive_Migration_Resource_Policy"
  description  = "AT-Britive_Migration_Resource_Policy_Description"
  access_type  = "Allow"
  access_level = "manage"
  consumer     = "resourcemanager"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" },
    ]
  })
  condition = jsonencode({
    ipAddress    = "192.162.0.0/16"
    timeOfAccess = {
      "dateSchedule": {
        "fromDate": %q,
        "toDate":   %q,
        "timezone": "Asia/Calcutta"
      }
    }
  })
  resource_labels {
    label_key = britive_resource_manager_resource_label.migration.name
    values    = ["Production"]
  }
}
`, timeOfAccessFrom, timeOfAccessTo)
	runMigrationTest(t, config)
}

func TestBritiveResourceManagerProfileMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_label" "migration_1" {
  name        = "AT-Britive_Migration_RMProfile_Label_1"
  description = "AT-Britive_Migration_RMProfile_Label_1_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
  values {
    name        = "Development"
    description = "Development Desc"
  }
}

resource "britive_resource_manager_resource_label" "migration_2" {
  name        = "AT-Britive_Migration_RMProfile_Label_2"
  description = "AT-Britive_Migration_RMProfile_Label_2_Description"
  label_color = "#1a2b3c"
  values {
    name        = "us-east-1"
    description = "US East 1"
  }
}

resource "britive_resource_manager_profile" "migration" {
  name                             = "AT-Britive_Migration_RM_Profile"
  description                      = "AT-Britive_Migration_RM_Profile_Description"
  expiration_duration              = 10800000
  extendable                       = true
  notification_prior_to_expiration = "1h0m0s"
  extension_duration               = "2h0m0s"
  extension_limit                  = 2
  allow_impersonation              = true
  associations {
    label_key = britive_resource_manager_resource_label.migration_1.name
    values    = ["Production", "Development"]
  }
  associations {
    label_key = britive_resource_manager_resource_label.migration_2.name
    values    = ["us-east-1"]
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceManagerProfilePermissionMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_type" "migration" {
  name        = "AT-Britive_Migration_RMPerm_ResType"
  description = "AT-Britive_Migration_RMPerm_ResType_Description"
  parameters {
    param_name   = "testfield5"
    param_type   = "string"
    is_mandatory = true
  }
}

resource "britive_resource_manager_response_template" "migration" {
  name                      = "AT-Britive_Migration_RMPerm_Template"
  description               = "AT-Britive_Migration_RMPerm_Template_Description"
  is_console_access_enabled = true
  show_on_ui                = false
  template_data             = "The user {{YS}} has the {{admin}}."
}

resource "britive_resource_manager_resource_type_permission" "migration" {
  name                = "AT-Britive_Migration_RMPerm_TypePerm"
  resource_type_id    = britive_resource_manager_resource_type.migration.id
  description         = "AT-Britive_Migration_RMPerm_TypePerm_Description"
  checkin_time_limit  = 180
  checkout_time_limit = 360
  is_draft            = false
  show_orig_creds     = true
  variables           = ["test1", "test2"]
  code_language       = "python"
  checkin_code        = "#!/bin/bash\necho done"
  checkout_code       = "#!/bin/bash\necho start"
  response_templates  = [britive_resource_manager_response_template.migration.name]
}

resource "britive_resource_manager_resource_label" "migration" {
  name        = "AT-Britive_Migration_RMPerm_Label"
  description = "AT-Britive_Migration_RMPerm_Label_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
}

resource "britive_resource_manager_resource" "migration" {
  name          = "AT-Britive_Migration_RMPerm_Resource"
  description   = "AT-Britive_Migration_RMPerm_Resource_Description"
  resource_type = britive_resource_manager_resource_type.migration.name
  parameter_values = {
    "testfield5" = "v5"
  }
}

resource "britive_resource_manager_profile" "migration" {
  name                = "AT-Britive_Migration_RMPerm_Profile"
  description         = "AT-Britive_Migration_RMPerm_Profile_Description"
  expiration_duration = 10800000
  extendable          = true
  notification_prior_to_expiration = "1h0m0s"
  extension_duration               = "2h0m0s"
  extension_limit                  = 2
  allow_impersonation = true
  associations {
    label_key = "Resource-Type"
    values    = [britive_resource_manager_resource_type.migration.name]
  }
}

resource "britive_resource_manager_profile_permission" "migration" {
  profile_id = britive_resource_manager_profile.migration.id
  name       = britive_resource_manager_resource_type_permission.migration.name
  version    = "LOCAL"
  variables {
    name              = "test1"
    value             = "val1"
    is_system_defined = false
  }
  variables {
    name              = "test2"
    value             = "val2"
    is_system_defined = false
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceManagerProfilePolicyMigration(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	config := migrationProviderBlock() + fmt.Sprintf(`
resource "britive_resource_manager_resource_label" "migration_1" {
  name        = "AT-Britive_Migration_RMPolicy_Label_1"
  description = "AT-Britive_Migration_RMPolicy_Label_1_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
  values {
    name        = "Development"
    description = "Development Desc"
  }
}

resource "britive_resource_manager_resource_label" "migration_2" {
  name        = "AT-Britive_Migration_RMPolicy_Label_2"
  description = "AT-Britive_Migration_RMPolicy_Label_2_Description"
  label_color = "#1a2b3c"
  values {
    name        = "us-east-1"
    description = "US East 1"
  }
}

resource "britive_resource_manager_profile" "migration" {
  name                             = "AT-Britive_Migration_RMPolicy_Profile"
  description                      = "AT-Britive_Migration_RMPolicy_Profile_Description"
  expiration_duration              = 10800000
  extendable                       = true
  notification_prior_to_expiration = "1h0m0s"
  extension_duration               = "2h0m0s"
  extension_limit                  = 2
  allow_impersonation              = true
  associations {
    label_key = britive_resource_manager_resource_label.migration_1.name
    values    = ["Production", "Development"]
  }
  associations {
    label_key = britive_resource_manager_resource_label.migration_2.name
    values    = ["us-east-1"]
  }
}

resource "britive_resource_manager_profile_policy" "migration" {
  profile_id   = britive_resource_manager_profile.migration.id
  policy_name  = "AT-Britive_Migration_RMPolicy_Policy"
  description  = "AT-Britive_Migration_RMPolicy_Policy_Description"
  access_type  = "Allow"
  consumer     = "resourceprofile"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" },
    ]
  })
  condition = jsonencode({
    ipAddress    = "192.162.0.0/16"
    timeOfAccess = {
      "dateSchedule": {
        "fromDate": %q,
        "toDate":   %q,
        "timezone": "Asia/Calcutta"
      }
    }
  })
  resource_labels {
    label_key = britive_resource_manager_resource_label.migration_2.name
    values    = ["us-east-1"]
  }
}
`, timeOfAccessFrom, timeOfAccessTo)
	runMigrationTest(t, config)
}

func TestBritiveResourceManagerProfilePolicyPrioritizationMigration(t *testing.T) {
	// v2.2.9 bug: britive_resource_manager_profile_policy_prioritization stores profile_id as a bare ID
	// in state, but the RM profile resource's .id evaluates to the full path "resource-manager/profile/{id}".
	// This causes the post-apply refresh plan to be non-empty (bare ID vs full path drift) after step 1,
	// which is a pre-existing v2.2.9 limitation that cannot be fixed in v3.0.0 code.
	t.Skip("Skipping migration test: v2.2.9 profile_policy_prioritization stores profile_id as bare ID " +
		"but config reference evaluates to full path, causing perpetual drift in v2.2.9 state that " +
		"cannot be resolved without a fix in the v2.2.9 provider itself.")

	// britive_resource_manager_profile_policy_prioritization was added in v2.2.9
	// and is therefore eligible for v2.2.9 → v3.0.0 migration testing.
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_label" "migration_1" {
  name        = "AT-Britive_Migration_RMPrio_Label_1"
  description = "AT-Britive_Migration_RMPrio_Label_1_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
  values {
    name        = "Development"
    description = "Development Desc"
  }
}

resource "britive_resource_manager_resource_label" "migration_2" {
  name        = "AT-Britive_Migration_RMPrio_Label_2"
  description = "AT-Britive_Migration_RMPrio_Label_2_Description"
  label_color = "#1a2b3c"
  values {
    name        = "us-east-1"
    description = "US East 1"
  }
}

resource "britive_resource_manager_profile" "migration" {
  name                             = "AT-Britive_Migration_RMPrio_Profile"
  description                      = "AT-Britive_Migration_RMPrio_Profile_Description"
  expiration_duration              = 10800000
  extendable                       = true
  notification_prior_to_expiration = "1h0m0s"
  extension_duration               = "2h0m0s"
  extension_limit                  = 2
  allow_impersonation              = true
  associations {
    label_key = britive_resource_manager_resource_label.migration_1.name
    values    = ["Production", "Development"]
  }
  associations {
    label_key = britive_resource_manager_resource_label.migration_2.name
    values    = ["us-east-1"]
  }
}

resource "britive_resource_manager_profile_policy" "migration_0" {
  profile_id   = britive_resource_manager_profile.migration.id
  policy_name  = "AT - Migration RMPrio Policy 0"
  description  = "AT - Migration RMPrio Policy 0 Desc"
  access_type  = "Allow"
  consumer     = "resourceprofile"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
  resource_labels {
    label_key = britive_resource_manager_resource_label.migration_2.name
    values    = ["us-east-1"]
  }
}

resource "britive_resource_manager_profile_policy" "migration_1" {
  profile_id   = britive_resource_manager_profile.migration.id
  policy_name  = "AT - Migration RMPrio Policy 1"
  description  = "AT - Migration RMPrio Policy 1 Desc"
  access_type  = "Allow"
  consumer     = "resourceprofile"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
  resource_labels {
    label_key = britive_resource_manager_resource_label.migration_2.name
    values    = ["us-east-1"]
  }
}

resource "britive_resource_manager_profile_policy_prioritization" "migration" {
  profile_id = britive_resource_manager_profile.migration.id
  policy_priority {
    id       = britive_resource_manager_profile_policy.migration_0.id
    priority = 0
  }
  policy_priority {
    id       = britive_resource_manager_profile_policy.migration_1.id
    priority = 1
  }
}
`
	runMigrationTest(t, config)
}

func TestBritiveResourceResourceBrokerPoolsMigration(t *testing.T) {
	config := migrationProviderBlock() + `
resource "britive_resource_manager_resource_type" "migration" {
  name        = "AT-Britive_Migration_BrokerPool_ResType"
  description = "AT-Britive_Migration_BrokerPool_ResType_Description"
  parameters {
    param_name   = "testfield1"
    param_type   = "string"
    is_mandatory = true
  }
}

resource "britive_resource_manager_resource_label" "migration" {
  name        = "AT-Britive_Migration_BrokerPool_Label"
  description = "AT-Britive_Migration_BrokerPool_Label_Description"
  label_color = "#abc123"
  values {
    name        = "Production"
    description = "Production Desc"
  }
}

resource "britive_resource_manager_resource" "migration" {
  name          = "AT-Britive_Migration_BrokerPool_Resource"
  description   = "AT-Britive_Migration_BrokerPool_Resource_Description"
  resource_type = britive_resource_manager_resource_type.migration.name
  parameter_values = {
    "testfield1" = "value1"
  }
}

resource "britive_resource_manager_resource_broker_pools" "migration" {
  resource_id  = britive_resource_manager_resource.migration.id
  broker_pools = ["DO NOT DELETE - BROKER POOL TF Plugin"]
}
`
	runMigrationTest(t, config)
}
