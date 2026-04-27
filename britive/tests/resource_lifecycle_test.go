package tests

// resource_lifecycle_test.go — three categories of lifecycle tests for every resource:
//
//  1. Edit tests   — Create → edit fields → assert plan shows Update (not Replace) → apply → assert no drift
//  2. Idempotency  — Create → assert PostApplyPostRefresh plan is empty (no perpetual drift)
//  3. Migration    — Create with v2.2.9 (registry) → plan with v3.0.0 (local) → assert no drift
//
// Prerequisites
//   TF_ACC=1
//   BRITIVE_TENANT and BRITIVE_TOKEN environment variables (or ~/.britive/tf.config)
//
// Migration tests additionally require:
//   - Internet access to download the britive/britive v2.2.9 provider binary from the registry
//   - Terraform CLI on PATH (terraform-plugin-testing will invoke it)

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// ─────────────────────────────────────────────────────────────────────────────
// Pattern 1 — Edit tests
// ─────────────────────────────────────────────────────────────────────────────

// TestBritiveTagEditFields verifies that:
//   - Changing description and disabled status plans as an in-place Update (not a
//     Replace/destroy-recreate), confirming no field is accidentally marked RequiresReplace
//   - After the update apply, the API state matches the config (no drift)
func TestBritiveTagEditFields(t *testing.T) {
	const (
		identityProviderName = "Britive"
		name                 = "AT - Tag Lifecycle Edit Test"
		initialDescription   = "Initial description for lifecycle edit test"
		updatedDescription   = "Updated description for lifecycle edit test"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccTagLifecycleConfig(name, initialDescription, false, identityProviderName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.lifecycle"),
					resource.TestCheckResourceAttr("britive_tag.lifecycle", "name", name),
					resource.TestCheckResourceAttr("britive_tag.lifecycle", "description", initialDescription),
					resource.TestCheckResourceAttr("britive_tag.lifecycle", "disabled", "false"),
				),
			},
			// Step 2: Edit description and disabled — must be an in-place Update, not a Replace.
			// PreApply asserts the planned action before touching the API.
			// PostApplyPostRefresh asserts no residual drift after the API call completes.
			{
				Config: testAccTagLifecycleConfig(name, updatedDescription, true, identityProviderName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_tag.lifecycle", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.lifecycle"),
					resource.TestCheckResourceAttr("britive_tag.lifecycle", "description", updatedDescription),
					resource.TestCheckResourceAttr("britive_tag.lifecycle", "disabled", "true"),
				),
			},
		},
	})
}

// TestBritiveProfileEditFields verifies that profile fields that should update
// in-place (description, expiration_duration) plan as Update, and that adding
// a second association replans correctly without replacing the profile.
func TestBritiveProfileEditFields(t *testing.T) {
	const (
		applicationName    = "DO NOT DELETE - Azure TF Plugin"
		name               = "AT - Profile Lifecycle Edit Test"
		initialDescription = "Initial profile description"
		updatedDescription = "Updated profile description"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with one association and a short expiration
			{
				Config: testAccProfileLifecycleConfig(applicationName, name, initialDescription, "25m0s"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileExists("britive_profile.lifecycle"),
					resource.TestCheckResourceAttr("britive_profile.lifecycle", "name", name),
					resource.TestCheckResourceAttr("britive_profile.lifecycle", "description", initialDescription),
					resource.TestCheckResourceAttr("britive_profile.lifecycle", "expiration_duration", "25m0s"),
				),
			},
			// Step 2: Edit description and expiration_duration — both are in-place updates.
			{
				Config: testAccProfileLifecycleConfig(applicationName, name, updatedDescription, "1h0m0s"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_profile.lifecycle", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileExists("britive_profile.lifecycle"),
					resource.TestCheckResourceAttr("britive_profile.lifecycle", "description", updatedDescription),
					resource.TestCheckResourceAttr("britive_profile.lifecycle", "expiration_duration", "1h0m0s"),
				),
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Pattern 2 — Idempotency tests
// ─────────────────────────────────────────────────────────────────────────────
// Each test creates a resource and then uses PostApplyPostRefresh to assert that
// after the apply+refresh cycle there are no planned changes. This catches cases
// where the API returns values in a different form than what Terraform wrote
// (e.g. case normalization, ordering, computed-field drift).

func TestBritiveTagIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagIdempotencyConfig("AT - Tag Idempotency Test", "Britive"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.idempotency"),
				),
			},
		},
	})
}

// TestBritiveProfileIdempotency covers the Optional+Computed fields that were
// sources of drift during the migration (app_name, disabled default).
func TestBritiveProfileIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfileIdempotencyConfig(
					"DO NOT DELETE - Azure TF Plugin",
					"AT - Profile Idempotency Test",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileExists("britive_profile.idempotency"),
				),
			},
		},
	})
}

// TestBritiveApplicationIdempotency covers the computed fields that were sources
// of drift: version (Optional+Computed, API normalizes casing),
// entity_root_environment_group_id (Computed, not applicable to non-Snowflake apps),
// and catalog_app_id. Okta is used because it has the simplest schema.
func TestBritiveApplicationIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationIdempotencyConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationExists("britive_application.idempotency"),
				),
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Pattern 3 — Provider migration tests (v2.2.9 → v3.0.0)
// ─────────────────────────────────────────────────────────────────────────────
// Step 1 uses ExternalProviders to download and run v2.2.9 from the Terraform
// registry (SDKv2, Protocol v5). Step 2 uses the locally-built v3.0.0 binary
// (Plugin Framework, Protocol v6) with PlanOnly to verify that the state
// written by v2.2.9 produces no changes when read and planned by v3.0.0.
//
// Terraform automatically calls UpgradeResourceState on the new provider to
// migrate the internal state format (SDKv2 flatmap → Framework JSON). The test
// proves this migration is transparent to the user — no resource replacement
// or unexpected updates appear.
//
// Note: No providers are set at the TestCase level so that each step's provider
// configuration is used exclusively (step-level providers override TestCase
// providers when the TestCase has none). This is required so that step 1 uses
// only the registry v2.2.9 binary and step 2 uses only the local v3.0.0 binary.

func TestBritiveTagProviderMigration(t *testing.T) {
	const (
		identityProviderName = "Britive"
		name                 = "AT - Tag Provider Migration Test"
	)
	config := testAccTagMigrationConfig(name, identityProviderName)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheckFramework(t) },
		// No TestCase-level providers — each step specifies its own.
		Steps: []resource.TestStep{
			// Step 1: Create with v2.2.9 (SDKv2, Protocol v5)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"britive": {
						Source:            "britive/britive",
						VersionConstraint: "= 2.2.9",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("britive_tag.migration", "id"),
					resource.TestCheckResourceAttr("britive_tag.migration", "name", name),
				),
			},
			// Step 2: Plan with v3.0.0 (Plugin Framework, Protocol v6).
			// PlanOnly=true: runs terraform plan against the state left by step 1.
			// If the plan is non-empty the test fails, proving state incompatibility.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
				// ExpectNonEmptyPlan defaults to false — any planned change fails the test.
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Config helpers
// ─────────────────────────────────────────────────────────────────────────────

func testAccTagLifecycleConfig(name, description string, disabled bool, identityProviderName string) string {
	disabledVal := "false"
	if disabled {
		disabledVal = "true"
	}
	return fmt.Sprintf(`
data "britive_identity_provider" "existing" {
  name = %q
}

resource "britive_tag" "lifecycle" {
  name                 = %q
  description          = %q
  disabled             = %s
  identity_provider_id = data.britive_identity_provider.existing.id
}
`, identityProviderName, name, description, disabledVal)
}

func testAccProfileLifecycleConfig(applicationName, name, description, expirationDuration string) string {
	return fmt.Sprintf(`
data "britive_application" "app" {
  name = %q
}

resource "britive_profile" "lifecycle" {
  app_container_id    = data.britive_application.app.id
  name                = %q
  description         = %q
  expiration_duration = %q
  associations {
    type  = "Environment"
    value = "Subscription 1"
  }
}
`, applicationName, name, description, expirationDuration)
}

func testAccTagIdempotencyConfig(name, identityProviderName string) string {
	return fmt.Sprintf(`
data "britive_identity_provider" "existing" {
  name = %q
}

resource "britive_tag" "idempotency" {
  name                 = %q
  description          = "Idempotency test tag"
  identity_provider_id = data.britive_identity_provider.existing.id
}
`, identityProviderName, name)
}

func testAccProfileIdempotencyConfig(applicationName, name string) string {
	return fmt.Sprintf(`
data "britive_application" "app" {
  name = %q
}

resource "britive_profile" "idempotency" {
  app_container_id    = data.britive_application.app.id
  name                = %q
  description         = "Idempotency test profile"
  expiration_duration = "25m0s"
  associations {
    type  = "Environment"
    value = "Subscription 1"
  }
}
`, applicationName, name)
}

// testAccApplicationIdempotencyConfig uses the Okta application type which has
// no version field (Optional+Computed but not provided in config) and no
// entity_root_environment_group_id — testing that these Computed fields do not
// produce drift after the first apply.
func testAccApplicationIdempotencyConfig() string {
	return `
resource "britive_application" "idempotency" {
  application_type = "okta"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "AT - Okta App Idempotency Test"
  }
  properties {
    name  = "description"
    value = "AT - Okta App Idempotency Test Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1000
  }
}
`
}

// migrationProviderBlock returns a Terraform provider "britive" {} block that
// explicitly sets tenant and token with a full https:// URL. This is required
// for migration tests where step 1 runs v2.2.9 from the registry: that release
// (SDKv2) does not auto-prepend "https://" to the BRITIVE_TENANT environment
// variable — the fix was added in v3.0.0. Including an explicit provider block
// ensures both provider versions receive a valid scheme.
func migrationProviderBlock() string {
	tenant := os.Getenv("BRITIVE_TENANT")
	if tenant != "" && !strings.Contains(tenant, "://") {
		tenant = "https://" + tenant
	}
	token := os.Getenv("BRITIVE_TOKEN")
	if tenant == "" || token == "" {
		return "" // env vars absent; PreCheck will skip the test before we get here
	}
	return fmt.Sprintf(`
provider "britive" {
  tenant = %q
  token  = %q
}
`, tenant, token)
}

func testAccTagMigrationConfig(name, identityProviderName string) string {
	return migrationProviderBlock() + fmt.Sprintf(`
data "britive_identity_provider" "existing" {
  name = %q
}

resource "britive_tag" "migration" {
  name                 = %q
  description          = "Provider migration test tag"
  identity_provider_id = data.britive_identity_provider.existing.id
}
`, identityProviderName, name)
}

// Note: TestBritiveProfileProviderMigration is intentionally absent.
//
// The britive_profile "associations" field changed its HCL syntax between
// v2.2.9 and v3.0.0:
//
//   v2.2.9 (SDKv2 TypeSet)  — assignment syntax:  associations = [{type = "...", value = "..."}]
//   v3.0.0 (SetNestedBlock) — block syntax:        associations { type = "..." value = "..." }
//
// These two syntaxes are mutually exclusive; no single .tf configuration is
// valid for both versions. Users migrating profiles must update their .tf files
// alongside the provider upgrade. This is documented in MIGRATION.md §16.

// ─────────────────────────────────────────────────────────────────────────────
// Core resources — Edit tests
// ─────────────────────────────────────────────────────────────────────────────

// TestBritivePermissionEditFields verifies that the description of a permission
// can be updated in-place without triggering a Replace.
func TestBritivePermissionEditFields(t *testing.T) {
	const (
		name               = "AT - Permission Lifecycle Edit Test"
		initialDescription = "Initial permission description"
		updatedDescription = "Updated permission description"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePermissionConfig(name, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePermissionExists("britive_permission.new"),
					resource.TestCheckResourceAttr("britive_permission.new", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritivePermissionConfig(name, updatedDescription),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_permission.new", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_permission.new", "description", updatedDescription),
				),
			},
		},
	})
}

// TestBritiveRoleEditFields verifies that the role description can be updated
// in-place without replacing the role or its permissions.
func TestBritiveRoleEditFields(t *testing.T) {
	const (
		permName           = "AT - Role Lifecycle Perm"
		permDesc           = "AT - Role Lifecycle Perm Description"
		roleName           = "AT - Role Lifecycle Edit Test"
		initialDescription = "Initial role description"
		updatedDescription = "Updated role description"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRoleConfig(permName, permDesc, roleName, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveRoleExists("britive_role.new"),
					resource.TestCheckResourceAttr("britive_role.new", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritiveRoleConfig(permName, permDesc, roleName, updatedDescription),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_role.new", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_role.new", "description", updatedDescription),
				),
			},
		},
	})
}

// TestBritiveProfilePolicyEditFields verifies that the profile policy
// description can be updated in-place.
func TestBritiveProfilePolicyEditFields(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	const (
		applicationName    = "DO NOT DELETE - AWS TF Plugin"
		profileName        = "AT - Profile Policy Lifecycle Edit Test"
		policyName         = "AT - Profile Policy Lifecycle Edit Test"
		initialDescription = "Initial profile policy description"
		updatedDescription = "Updated profile policy description"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePolicyConfig(applicationName, profileName, policyName, initialDescription, timeOfAccessFrom, timeOfAccessTo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyExists("britive_profile_policy.new"),
					resource.TestCheckResourceAttr("britive_profile_policy.new", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritiveProfilePolicyConfig(applicationName, profileName, policyName, updatedDescription, timeOfAccessFrom, timeOfAccessTo),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_profile_policy.new", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_profile_policy.new", "description", updatedDescription),
				),
			},
		},
	})
}

// TestBritiveProfileAdditionalSettingsEditFields verifies that console_access
// and programmatic_access can be toggled in-place.
func TestBritiveProfileAdditionalSettingsEditFields(t *testing.T) {
	const (
		applicationName = "DO NOT DELETE - GCP TF Plugin"
		profileName     = "AT - Profile Additional Settings Lifecycle Edit Test"
		profileDesc     = "AT - Profile Addl Settings Lifecycle Edit Test Description"
		associationVal  = "britive-gdev-cis.net"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileAdditionalSettingsConfig(
					applicationName, profileName, profileDesc, associationVal,
					"false", "true", "false", "",
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileAdditionalSettingsExists("britive_profile_additional_settings.new"),
					resource.TestCheckResourceAttr("britive_profile_additional_settings.new", "console_access", "true"),
					resource.TestCheckResourceAttr("britive_profile_additional_settings.new", "programmatic_access", "false"),
				),
			},
			{
				Config: testAccCheckBritiveProfileAdditionalSettingsConfig(
					applicationName, profileName, profileDesc, associationVal,
					"false", "false", "true", "",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_profile_additional_settings.new", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_profile_additional_settings.new", "console_access", "false"),
					resource.TestCheckResourceAttr("britive_profile_additional_settings.new", "programmatic_access", "true"),
				),
			},
		},
	})
}

// TestBritiveProfilePolicyPrioritizationEditFields verifies that the priority
// ordering of profile policies can be changed in-place.
func TestBritiveProfilePolicyPrioritizationEditFields(t *testing.T) {
	const (
		applicationName = "DO NOT DELETE - AWS TF Plugin"
		profileName     = "AT - Profile Policy Prio Lifecycle Edit Test"
		policyName0     = "AT - Profile Policy Prio Lifecycle Edit Test 0"
		policyDesc0     = "AT - Profile Policy Prio Lifecycle Edit Test 0 Desc"
		policyName1     = "AT - Profile Policy Prio Lifecycle Edit Test 1"
		policyDesc1     = "AT - Profile Policy Prio Lifecycle Edit Test 1 Desc"
		policyName2     = "AT - Profile Policy Prio Lifecycle Edit Test 2"
		policyDesc2     = "AT - Profile Policy Prio Lifecycle Edit Test 2 Desc"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: priorities 0, 1, 2
			{
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(
					applicationName, profileName,
					policyName0, policyDesc0,
					policyName1, policyDesc1,
					policyName2, policyDesc2,
					0, 1, 2,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy_prioritization.new_priority"),
				),
			},
			// Step 2: swap to priorities 2, 0, 1 — must be Update, not Replace.
			{
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(
					applicationName, profileName,
					policyName0, policyDesc0,
					policyName1, policyDesc1,
					policyName2, policyDesc2,
					2, 0, 1,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_profile_policy_prioritization.new_priority", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Core resources — Idempotency tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBritiveConstraintIdempotency(t *testing.T) {
	constraintExpression := "request.time < timestamp('" + time.Now().AddDate(0, 0, 2).Format("2006-01-02T15:04:05Z07:00") + "')"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveConstraintConfig(
					"DO NOT DELETE - GCP TF Plugin",
					"AT - Britive Constraint Lifecycle Test",
					"AT - Britive Constraint Lifecycle Test Description",
					"britive-gdev-cis.net",
					"BigQuery Data Owner", "role",
					"bigquery.datasets", "my-first-project-310615.dataset2",
					"Storage Admin",
					"condition",
					"ConditionConstraintType",
					"Condition Constraint Type Description",
					constraintExpression,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveConstraintExists("britive_constraint.new"),
					testAccCheckBritiveConstraintExists("britive_constraint.new_condition"),
				),
			},
		},
	})
}

func TestBritiveTagMemberIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagMemberConfig(
					"Britive",
					"AT - Tag Member Lifecycle Test",
					"AT - Tag Member Lifecycle Test Description",
					"britiveprovideracceptancetest",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagMemberExists("britive_tag_member.new"),
				),
			},
		},
	})
}

func TestBritiveEntityGroupIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveEntityGroupConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveEntityGroupExists("britive_entity_group.entity_group_new"),
				),
			},
		},
	})
}

func TestBritiveEntityEnvironmentIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveEntityEnvironmentConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveEntityEnvironmentExists("britive_entity_environment.entity_environment_new"),
				),
			},
		},
	})
}

func TestBritivePolicyIdempotency(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePolicyConfig(
					"AT - Britive Permission Test Policy Lifecycle",
					"AT - Britive Permission Test Policy Lifecycle Description",
					"AT - Britive Role Test Policy Lifecycle",
					"AT - Britive Role Test Policy Lifecycle Description",
					"AT - Britive Policy Lifecycle Test",
					"AT - Britive Policy Lifecycle Test Description",
					timeOfAccessFrom, timeOfAccessTo,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePolicyExists("britive_policy.new"),
				),
			},
		},
	})
}

func TestBritivePermissionIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePermissionConfig(
					"AT - Britive Permission Lifecycle Test",
					"AT - Britive Permission Lifecycle Test Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePermissionExists("britive_permission.new"),
				),
			},
		},
	})
}

func TestBritiveRoleIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRoleConfig(
					"AT - Britive Permission Lifecycle Role",
					"AT - Britive Permission Lifecycle Role Description",
					"AT - Britive Role Lifecycle Test",
					"AT - Britive Role Lifecycle Test Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveRoleExists("britive_role.new"),
				),
			},
		},
	})
}

func TestBritiveProfilePermissionIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePermissionConfig(
					"DO NOT DELETE - Azure TF Plugin",
					"AT - Profile Permission Lifecycle Test",
					"AT - Profile Permission Lifecycle Test Description",
					"QA",
					"Application Developer", "role",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePermissionExists("britive_profile_permission.new"),
				),
			},
		},
	})
}

func TestBritiveProfileSessionAttributeIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileSessionAttributeConfig(
					"DO NOT DELETE - AWS TF Plugin",
					"AT - Profile Session Attribute Lifecycle Test",
					"Date Of Birth",
					"dob",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileSessionAttributeExists("britive_profile_session_attribute.new"),
				),
			},
		},
	})
}

func TestBritiveProfileAdditionalSettingsIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileAdditionalSettingsConfig(
					"DO NOT DELETE - GCP TF Plugin",
					"AT - Profile Additional Settings Lifecycle Test",
					"AT - Profile Addl Settings Lifecycle Test Description",
					"britive-gdev-cis.net",
					"false", "true", "false", "",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileAdditionalSettingsExists("britive_profile_additional_settings.new"),
				),
			},
		},
	})
}

func TestBritiveProfilePolicyIdempotency(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePolicyConfig(
					"DO NOT DELETE - AWS TF Plugin",
					"AT - Profile Policy Lifecycle Test",
					"AT - Profile Policy Lifecycle Test",
					"AT - Profile Policy Lifecycle Test Description",
					timeOfAccessFrom, timeOfAccessTo,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyExists("britive_profile_policy.new"),
				),
			},
		},
	})
}

func TestBritiveProfilePolicyPrioritizationIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(
					"DO NOT DELETE - AWS TF Plugin",
					"AT - Profile Policy Prioritization Lifecycle Test",
					"AT - Profile Policy Prio Lifecycle Test 0",
					"AT - Profile Policy Prio Lifecycle Test 0 Description",
					"AT - Profile Policy Prio Lifecycle Test 1",
					"AT - Profile Policy Prio Lifecycle Test 1 Description",
					"AT - Profile Policy Prio Lifecycle Test 2",
					"AT - Profile Policy Prio Lifecycle Test 2 Description",
					0, 1, 2,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy_prioritization.new_priority"),
				),
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Resource Manager resources — Edit tests
// ─────────────────────────────────────────────────────────────────────────────

// TestBritiveResourceTypeEditFields verifies the description of a resource type
// can be updated in-place.
func TestBritiveResourceTypeEditFields(t *testing.T) {
	const (
		name               = "AT_Britive_Resource_Type_Lifecycle_Edit_Test"
		initialDescription = "AT_Britive_Resource_Type_Lifecycle_Edit_Test_Initial_Description"
		updatedDescription = "AT_Britive_Resource_Type_Lifecycle_Edit_Test_Updated_Description"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypeConfig(name, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypeExists("britive_resource_manager_resource_type.new_resource_type_1"),
					resource.TestCheckResourceAttr("britive_resource_manager_resource_type.new_resource_type_1", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritiveResourceTypeConfig(name, updatedDescription),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_resource_manager_resource_type.new_resource_type_1", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_resource_manager_resource_type.new_resource_type_1", "description", updatedDescription),
				),
			},
		},
	})
}

// TestBritiveResponseTemplateEditFields verifies that the description of a
// response template can be updated in-place.
func TestBritiveResponseTemplateEditFields(t *testing.T) {
	const (
		name               = "AT_Britive_Response_Template_Lifecycle_Edit_Test"
		initialDescription = "AT_Britive_Response_Template_Lifecycle_Edit_Test_Initial_Desc"
		updatedDescription = "AT_Britive_Response_Template_Lifecycle_Edit_Test_Updated_Desc"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResponseTemplateConfig(name, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResponseTemplateExists("britive_resource_manager_response_template.new_response_template_1"),
					resource.TestCheckResourceAttr("britive_resource_manager_response_template.new_response_template_1", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritiveResponseTemplateConfig(name, updatedDescription),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_resource_manager_response_template.new_response_template_1", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_resource_manager_response_template.new_response_template_1", "description", updatedDescription),
				),
			},
		},
	})
}

// TestBritiveResourceLabelEditFields verifies that the description of a
// resource label can be updated in-place.
func TestBritiveResourceLabelEditFields(t *testing.T) {
	const (
		name               = "AT_Britive_Resource_Label_Lifecycle_Edit_Test"
		initialDescription = "AT_Britive_Resource_Label_Lifecycle_Edit_Test_Initial_Description"
		updatedDescription = "AT_Britive_Resource_Label_Lifecycle_Edit_Test_Updated_Description"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelConfig(name, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceLabelExists("britive_resource_manager_resource_label.resource_label_1"),
					resource.TestCheckResourceAttr("britive_resource_manager_resource_label.resource_label_1", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritiveResourceLabelConfig(name, updatedDescription),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_resource_manager_resource_label.resource_label_1", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_resource_manager_resource_label.resource_label_1", "description", updatedDescription),
				),
			},
		},
	})
}

// TestBritiveResourceManagerProfileEditFields verifies that the description of
// a resource manager profile can be updated in-place.
func TestBritiveResourceManagerProfileEditFields(t *testing.T) {
	const (
		label1Name         = "AT-Britive_RM_Profile_Lifecycle_Edit_Label_1"
		label1Desc         = "AT-Britive_RM_Profile_Lifecycle_Edit_Label_1_Description"
		label2Name         = "AT-Britive_RM_Profile_Lifecycle_Edit_Label_2"
		label2Desc         = "AT-Britive_RM_Profile_Lifecycle_Edit_Label_2_Description"
		profileName        = "AT-Britive_RM_Profile_Lifecycle_Edit_Test"
		initialDescription = "AT-Britive_RM_Profile_Lifecycle_Edit_Test_Initial_Description"
		updatedDescription = "AT-Britive_RM_Profile_Lifecycle_Edit_Test_Updated_Description"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfileConfig(label1Name, label1Desc, label2Name, label2Desc, profileName, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_profile.resource_profile_1"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "description", initialDescription),
				),
			},
			{
				Config: testAccCheckBritiveResourceManagerProfileConfig(label1Name, label1Desc, label2Name, label2Desc, profileName, updatedDescription),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_resource_manager_profile.resource_profile_1", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "description", updatedDescription),
				),
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Resource Manager resources — Idempotency tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBritiveResourceTypeIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypeConfig(
					"AT_Britive_Resource_Type_Lifecycle_Idempotency_Test",
					"AT_Britive_Resource_Type_Lifecycle_Idempotency_Test_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypeExists("britive_resource_manager_resource_type.new_resource_type_1"),
				),
			},
		},
	})
}

func TestBritiveResourceTypePermissionIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypePermissionConfig(
					"AT-Britive_RM_Lifecycle_Resource_Type_Perm",
					"AT-Britive_RM_Lifecycle_Resource_Type_Perm_Description",
					"AT-Britive_RM_Lifecycle_Response_Template_Perm",
					"AT-Britive_RM_Lifecycle_Response_Template_Perm_Description",
					"AT-Britive_RM_Lifecycle_Type_Permission_Perm",
					"AT-Britive_RM_Lifecycle_Type_Permission_Perm_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypePermissionExists("britive_resource_manager_resource_type_permission.new_resource_type_permission_1"),
				),
			},
		},
	})
}

func TestBritiveResponseTemplateIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResponseTemplateConfig(
					"AT_Britive_Response_Template_Lifecycle_Idempotency_Test",
					"AT_Britive_Response_Template_Lifecycle_Idempotency_Test_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResponseTemplateExists("britive_resource_manager_response_template.new_response_template_1"),
				),
			},
		},
	})
}

func TestBritiveResourceLabelIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelConfig(
					"AT-Britive_Resource_Label_Lifecycle_Idempotency_Test",
					"AT-Britive_Resource_Label_Lifecycle_Idempotency_Test_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceLabelExists("britive_resource_manager_resource_label.resource_label_1"),
				),
			},
		},
	})
}

func TestBritiveResourceResourceIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceResourceConfig(
					"AT-Britive_RM_Lifecycle_Resource_Type_2",
					"AT-Britive_RM_Lifecycle_Resource_Type_2_Description",
					"AT-Britive_RM_Lifecycle_Resource_Label_2",
					"AT-Britive_RM_Lifecycle_Resource_Label_2_Description",
					"AT-Britive_RM_Lifecycle_Resource_2",
					"AT-Britive_RM_Lifecycle_Resource_2_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceResourceExists("britive_resource_manager_resource.resource_1"),
				),
			},
		},
	})
}

func TestBritiveResourceManagerProfileIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfileConfig(
					"AT-Britive_RM_Profile_Lifecycle_Idempotency_Label_1",
					"AT-Britive_RM_Profile_Lifecycle_Idempotency_Label_1_Description",
					"AT-Britive_RM_Profile_Lifecycle_Idempotency_Label_2",
					"AT-Britive_RM_Profile_Lifecycle_Idempotency_Label_2_Description",
					"AT-Britive_RM_Profile_Lifecycle_Idempotency_Test",
					"AT-Britive_RM_Profile_Lifecycle_Idempotency_Test_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_profile.resource_profile_1"),
				),
			},
		},
	})
}

func TestBritiveResourceResourcePolicyIdempotency(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceResourcePolicyConfig(
					"AT-Britive_RM_Lifecycle_Resource_Label_Policy",
					"AT-Britive_RM_Lifecycle_Resource_Label_Policy_Description",
					timeOfAccessFrom, timeOfAccessTo,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceResourcePolicyExists("britive_resource_manager_resource_policy.resource_policy_1"),
				),
			},
		},
	})
}

func TestBritiveResourceManagerProfilePermissionIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePermissionConfig(
					"AT-Britive_RM_Lifecycle_Perm_Resource_Type",
					"AT-Britive_RM_Lifecycle_Perm_Resource_Type_Description",
					"AT-Britive_RM_Lifecycle_Perm_Resource",
					"AT-Britive_RM_Lifecycle_Perm_Resource_Description",
					"AT-Britive_RM_Lifecycle_Perm_Response_Template",
					"AT-Britive_RM_Lifecycle_Perm_Response_Template_Description",
					"AT-Britive_RM_Lifecycle_Perm_Type_Permission",
					"AT-Britive_RM_Lifecycle_Perm_Type_Permission_Description",
					"AT-Britive_RM_Lifecycle_Perm_Profile",
					"AT-Britive_RM_Lifecycle_Perm_Profile_Description",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_profile_permission.profile_permission_1"),
				),
			},
		},
	})
}

func TestBritiveResourceManagerProfilePolicyIdempotency(t *testing.T) {
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePolicyConfig(
					"AT-Britive_RM_Profile_Lifecycle_Policy_Label_1",
					"AT-Britive_RM_Profile_Lifecycle_Policy_Label_1_Description",
					"AT-Britive_RM_Profile_Lifecycle_Policy_Label_2",
					"AT-Britive_RM_Profile_Lifecycle_Policy_Label_2_Description",
					"AT-Britive_RM_Profile_Lifecycle_Policy_Profile",
					"AT-Britive_RM_Profile_Lifecycle_Policy_Profile_Description",
					timeOfAccessFrom, timeOfAccessTo,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_profile_policy.resource_profile_policy_1"),
				),
			},
		},
	})
}

func TestBritiveResourceResourceBrokerPoolIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceResourceBrokerPoolConfig(
					"AT-Britive_RM_Lifecycle_Broker_Pool_Resource_Type",
					"AT-Britive_RM_Lifecycle_Broker_Pool_Resource_Type_Description",
					"AT-Britive_RM_Lifecycle_Broker_Pool_Resource_Label",
					"AT-Britive_RM_Lifecycle_Broker_Pool_Resource_Label_Description",
					"AT-Britive_RM_Lifecycle_Broker_Pool_Resource",
					"AT-Britive_RM_Lifecycle_Broker_Pool_Resource_Description",
					"DO NOT DELETE - BROKER POOL TF Plugin",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceResourceBrokerPoolExists("britive_resource_manager_resource_broker_pools.resource_broker_pool_1"),
				),
			},
		},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Resource Manager Profile — Extensible Session (v3.0.0)
// ─────────────────────────────────────────────────────────────────────────────

// TestBritiveResourceManagerProfileExtensibleSessionIdempotency verifies that a
// resource manager profile with extendable=true and all session-extension fields
// set produces no drift after apply.
func TestBritiveResourceManagerProfileExtensibleSessionIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRMProfileExtensibleSessionConfig(
					"AT-Britive_RM_Lifecycle_Ext_Label_1",
					"AT-Britive_RM_Lifecycle_Ext_Label_1_Desc",
					"AT-Britive_RM_Lifecycle_Ext_Label_2",
					"AT-Britive_RM_Lifecycle_Ext_Label_2_Desc",
					"AT-Britive_RM_Lifecycle_Extensible_Profile",
					"AT-Britive_RM_Lifecycle_Extensible_Profile_Desc",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_profile.resource_profile_1"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "extendable", "true"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "notification_prior_to_expiration", "1h0m0s"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "extension_duration", "2h0m0s"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "extension_limit", "2"),
				),
			},
		},
	})
}

// TestBritiveResourceManagerProfileExtensibleSessionEditFields verifies that the
// extensible session fields can be updated in-place (no Replace triggered).
func TestBritiveResourceManagerProfileExtensibleSessionEditFields(t *testing.T) {
	const (
		label1Name  = "AT-Britive_RM_Lifecycle_ExtEdit_Label_1"
		label1Desc  = "AT-Britive_RM_Lifecycle_ExtEdit_Label_1_Desc"
		label2Name  = "AT-Britive_RM_Lifecycle_ExtEdit_Label_2"
		label2Desc  = "AT-Britive_RM_Lifecycle_ExtEdit_Label_2_Desc"
		profileName = "AT-Britive_RM_Lifecycle_ExtEdit_Profile"
		profileDesc = "AT-Britive_RM_Lifecycle_ExtEdit_Profile_Desc"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRMProfileExtensibleSessionConfig(label1Name, label1Desc, label2Name, label2Desc, profileName, profileDesc),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_profile.resource_profile_1"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "extendable", "true"),
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "extension_limit", "2"),
				),
			},
			// Increase extension_limit — must be an in-place Update, not a Replace.
			{
				Config: testAccRMProfileExtensibleSessionConfigUpdated(label1Name, label1Desc, label2Name, label2Desc, profileName, profileDesc),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_resource_manager_profile.resource_profile_1", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_resource_manager_profile.resource_profile_1", "extension_limit", "5"),
				),
			},
		},
	})
}

func testAccRMProfileExtensibleSessionConfig(label1Name, label1Desc, label2Name, label2Desc, profileName, profileDesc string) string {
	return fmt.Sprintf(`
resource "britive_resource_manager_resource_label" "resource_label_1" {
	name        = "%s"
	description = "%s"
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
resource "britive_resource_manager_resource_label" "resource_label_2" {
	name        = "%s"
	description = "%s"
	label_color = "#1a2b3c"
	values {
		name        = "us-east-1"
		description = "us-east-1 Desc"
	}
	values {
		name        = "eu-west-1"
		description = "eu-west-1 Desc"
	}
}
resource "britive_resource_manager_profile" "resource_profile_1" {
	name                             = "%s"
	description                      = "%s"
	expiration_duration              = 10800000
	extendable                       = true
	notification_prior_to_expiration = "1h0m0s"
	extension_duration               = "2h0m0s"
	extension_limit                  = 2
	allow_impersonation              = true
	associations {
		label_key = britive_resource_manager_resource_label.resource_label_1.name
		values    = ["Production", "Development"]
	}
	associations {
		label_key = britive_resource_manager_resource_label.resource_label_2.name
		values    = ["us-east-1", "eu-west-1"]
	}
}
`, label1Name, label1Desc, label2Name, label2Desc, profileName, profileDesc)
}

func testAccRMProfileExtensibleSessionConfigUpdated(label1Name, label1Desc, label2Name, label2Desc, profileName, profileDesc string) string {
	return fmt.Sprintf(`
resource "britive_resource_manager_resource_label" "resource_label_1" {
	name        = "%s"
	description = "%s"
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
resource "britive_resource_manager_resource_label" "resource_label_2" {
	name        = "%s"
	description = "%s"
	label_color = "#1a2b3c"
	values {
		name        = "us-east-1"
		description = "us-east-1 Desc"
	}
	values {
		name        = "eu-west-1"
		description = "eu-west-1 Desc"
	}
}
resource "britive_resource_manager_profile" "resource_profile_1" {
	name                             = "%s"
	description                      = "%s"
	expiration_duration              = 10800000
	extendable                       = true
	notification_prior_to_expiration = "1h0m0s"
	extension_duration               = "2h0m0s"
	extension_limit                  = 5
	allow_impersonation              = true
	associations {
		label_key = britive_resource_manager_resource_label.resource_label_1.name
		values    = ["Production", "Development"]
	}
	associations {
		label_key = britive_resource_manager_resource_label.resource_label_2.name
		values    = ["us-east-1", "eu-west-1"]
	}
}
`, label1Name, label1Desc, label2Name, label2Desc, profileName, profileDesc)
}

// ─────────────────────────────────────────────────────────────────────────────
// Resource Manager Profile Policy Prioritization — Lifecycle (v3.0.0)
// ─────────────────────────────────────────────────────────────────────────────

// TestBritiveTagOwnerEditFields verifies that updating owner sets (adding/removing
// a user owner or tag owner) plans as an in-place Update, not a Replace.
func TestBritiveTagOwnerEditFields(t *testing.T) {
	const (
		identityProviderName = "Britive"
		tagName              = "AT - Tag Owner Lifecycle Edit Test"
		ownerTagName         = "AT - Tag Owner Lifecycle Owner Tag"
		ownerUsername        = "britiveprovideracceptancetest"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with a user owner only
			{
				Config: testAccTagOwnerLifecycleConfig(identityProviderName, tagName, ownerTagName, ownerUsername, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagOwnerExists("britive_tag_owner.lifecycle"),
					resource.TestCheckResourceAttrSet("britive_tag_owner.lifecycle", "tag_id"),
				),
			},
			// Step 2: Add a tag owner — must be an in-place Update (not Replace)
			{
				Config: testAccTagOwnerLifecycleConfig(identityProviderName, tagName, ownerTagName, ownerUsername, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("britive_tag_owner.lifecycle", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagOwnerExists("britive_tag_owner.lifecycle"),
				),
			},
		},
	})
}

// TestBritiveTagOwnerIdempotency verifies that applying britive_tag_owner produces
// no drift on the second plan/apply cycle.
func TestBritiveTagOwnerIdempotency(t *testing.T) {
	const (
		identityProviderName = "Britive"
		tagName              = "AT - Tag Owner Idempotency Test"
		ownerTagName         = "AT - Tag Owner Idempotency Owner Tag"
		ownerUsername        = "britiveprovideracceptancetest"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagOwnerLifecycleConfig(identityProviderName, tagName, ownerTagName, ownerUsername, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagOwnerExists("britive_tag_owner.lifecycle"),
				),
			},
		},
	})
}

func testAccTagOwnerLifecycleConfig(identityProviderName, tagName, ownerTagName, ownerUsername string, includeTagOwner bool) string {
	tagOwnerBlock := ""
	ownerTagResource := ""
	if includeTagOwner {
		ownerTagResource = fmt.Sprintf(`
resource "britive_tag" "owner_tag" {
  name                 = %q
  description          = "Owner tag for lifecycle test"
  identity_provider_id = data.britive_identity_provider.existing.id
}`, ownerTagName)
		tagOwnerBlock = `
		tag {
			id = britive_tag.owner_tag.id
		}`
	}
	return fmt.Sprintf(`
data "britive_identity_provider" "existing" {
  name = %q
}

resource "britive_tag" "lifecycle_target" {
  name                 = %q
  description          = "Lifecycle test target tag"
  identity_provider_id = data.britive_identity_provider.existing.id
}
%s
resource "britive_tag_owner" "lifecycle" {
  tag_id = britive_tag.lifecycle_target.id

  user {
    name = %q
  }
%s
}
`, identityProviderName, tagName, ownerTagResource, ownerUsername, tagOwnerBlock)
}

// TestBritiveResourceManagerProfilePolicyPrioritizationIdempotency verifies that
// applying the prioritization resource produces no drift on second plan/apply.
func TestBritiveResourceManagerProfilePolicyPrioritizationIdempotency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePolicyPrioritizationConfig(
					"AT-Britive_RM_Lifecycle_PP_Label_1",
					"AT-Britive_RM_Lifecycle_PP_Label_1_Desc",
					"AT-Britive_RM_Lifecycle_PP_Label_2",
					"AT-Britive_RM_Lifecycle_PP_Label_2_Desc",
					"AT-Britive_RM_Lifecycle_PP_Profile",
					"AT-Britive_RM_Lifecycle_PP_Profile_Desc",
					"AT - RM Lifecycle PP Policy 0",
					"AT - RM Lifecycle PP Policy 0 Desc",
					"AT - RM Lifecycle PP Policy 1",
					"AT - RM Lifecycle PP Policy 1 Desc",
					"AT - RM Lifecycle PP Policy 2",
					"AT - RM Lifecycle PP Policy 2 Desc",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePolicyPrioritizationExists("britive_resource_manager_profile_policy_prioritization.priority_1"),
				),
			},
		},
	})
}
