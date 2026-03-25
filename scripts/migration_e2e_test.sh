#!/usr/bin/env bash
# migration_e2e_test.sh — End-to-end migration test: v2.2.9 (main) → v3.0.0 (migration_framework)
#
# Strategy:
#   Phase 1  — Build provider binary from the `main` branch (v2.2.9).
#               Apply all resources + data sources. Plan again → must be empty.
#   Phase 2  — Rebuild provider binary from the `migration_framework` branch (v3.0.0).
#               The dev_overrides in ~/.terraformrc picks up the new binary automatically.
#               Plan → must be empty (migration validation).
#               Apply → confirm v3.0.0 apply works.
#               Plan again → must be empty (idempotency).
#   Cleanup  — terraform destroy using v3.0.0 binary.
#
# Prerequisites:
#   export BRITIVE_TENANT=<tenant-host-or-url>
#   export BRITIVE_TOKEN=<api-token>
#   ~/.terraformrc must have dev_overrides pointing to PROVIDER_SRC
#
# Run from any directory; PROVIDER_SRC is hard-coded below.

set -euo pipefail

# ─── Paths ───────────────────────────────────────────────────────────────────
PROVIDER_SRC="/Users/rsodhani/go/src/github.com/britive/terraform-provider-britive"
WORK_DIR="${TMPDIR:-/tmp}/britive-migration-e2e"

# ─── Colours ─────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

step()  { echo -e "\n${CYAN}══════════════════════════════════════════${NC}"; echo -e "${YELLOW}  $*${NC}"; echo -e "${CYAN}══════════════════════════════════════════${NC}"; }
ok()    { echo -e "${GREEN}  ✓ $*${NC}"; }
fail()  { echo -e "${RED}  ✗ $*${NC}"; exit 1; }
warn()  { echo -e "${YELLOW}  ⚠ $*${NC}"; }

# ─── Cleanup on EXIT ─────────────────────────────────────────────────────────
ORIGINAL_BRANCH=""

cleanup() {
  echo ""
  step "Cleanup"

  # Restore original git branch in provider source
  if [[ -n "$ORIGINAL_BRANCH" ]]; then
    cd "$PROVIDER_SRC" 2>/dev/null || true
    CURRENT=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
    if [[ "$CURRENT" != "$ORIGINAL_BRANCH" ]]; then
      warn "Restoring git branch to ${ORIGINAL_BRANCH}"
      # Re-sync vendor for the original branch so the repo isn't left in bad state
      git checkout "$ORIGINAL_BRANCH" 2>/dev/null || warn "Could not restore branch"
      chmod -R u+w vendor/ 2>/dev/null || true
      go mod vendor 2>/dev/null || warn "go mod vendor failed on cleanup"
    fi
  fi

  # Destroy if state exists (uses whatever binary is currently built = v3.0.0)
  if [[ -d "$WORK_DIR" ]] && [[ -f "$WORK_DIR/terraform.tfstate" ]]; then
    warn "Running terraform destroy for cleanup..."
    cd "$WORK_DIR"
    terraform destroy -auto-approve -no-color 2>&1 | tail -30 \
      || warn "Destroy had errors — manual cleanup may be needed"
    ok "Destroy complete"
  fi
}
trap cleanup EXIT

# ─── Preflight ───────────────────────────────────────────────────────────────
step "Preflight checks"

[[ -z "${BRITIVE_TENANT:-}" ]] && fail "BRITIVE_TENANT must be set"
[[ -z "${BRITIVE_TOKEN:-}"  ]] && fail "BRITIVE_TOKEN must be set"
command -v terraform &>/dev/null || fail "terraform not found in PATH"
command -v go &>/dev/null        || fail "go not found in PATH"

# Verify dev_overrides is configured
grep -q "dev_overrides" "${HOME}/.terraformrc" \
  || fail "~/.terraformrc does not have dev_overrides — cannot use local binaries"

TENANT="${BRITIVE_TENANT}"
[[ "$TENANT" != https://* ]] && TENANT="https://${TENANT}"

ok "BRITIVE_TENANT : ${TENANT}"
ok "Terraform      : $(terraform version -no-color | head -1)"
ok "Go             : $(go version)"
ok "dev_overrides  : $(grep -A1 'dev_overrides' "${HOME}/.terraformrc" | tail -1 | xargs)"

# Record current branch so cleanup can restore it
cd "$PROVIDER_SRC"
ORIGINAL_BRANCH=$(git rev-parse --abbrev-ref HEAD)
ok "Current branch : ${ORIGINAL_BRANCH}"

# ─── Compute dynamic values ──────────────────────────────────────────────────
DATE_FROM=$(date -v+2d '+%Y-%m-%d %H:%M:%S')
DATE_TO=$(date -v+7d '+%Y-%m-%d %H:%M:%S')
CONSTRAINT_DATE=$(date -v+2d '+%Y-%m-%dT%H:%M:%SZ')
CONSTRAINT_EXP="request.time < timestamp('${CONSTRAINT_DATE}')"

# ─── Create workspace ────────────────────────────────────────────────────────
step "Creating Terraform workspace"
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"
ok "Workspace: ${WORK_DIR}"

# versions.tf — broad constraint; dev_overrides bypasses version checking anyway
cat > "$WORK_DIR/versions.tf" << 'VEOF'
terraform {
  required_providers {
    britive = {
      source  = "britive/britive"
      version = ">= 2.0.0"
    }
  }
}
VEOF

# provider.tf — explicit tenant+token so v2.2.9 gets a valid https:// URL
cat > "$WORK_DIR/provider.tf" << PEOF
provider "britive" {
  tenant = "${TENANT}"
  token  = "${BRITIVE_TOKEN}"
}
PEOF

# main.tf — all resources and data sources
# All API-level names are prefixed with "MIGS-" to avoid collision with
# existing Go acceptance-test resources in the same environment.
cat > "$WORK_DIR/main.tf" << MEOF
# ═══════════════════════════════════════════════════════════
# DATA SOURCES
# ═══════════════════════════════════════════════════════════

# 1. identity_provider
data "britive_identity_provider" "main" {
  name = "Britive"
}

# 2. application (Azure)
data "britive_application" "azure" {
  name = "DO NOT DELETE - Azure TF Plugin"
}

# 3. application (AWS)
data "britive_application" "aws" {
  name = "DO NOT DELETE - AWS TF Plugin"
}

# 4. application (GCP)
data "britive_application" "gcp" {
  name = "DO NOT DELETE - GCP TF Plugin"
}

# 5. connection
data "britive_connection" "itsm" {
  name = "TF_ACCEPTANCE_TEST_ITSM_DO_NOT_DELETE"
}

# 6. all_connections
data "britive_all_connections" "main" {}

# ═══════════════════════════════════════════════════════════
# CORE RESOURCES
# ═══════════════════════════════════════════════════════════

# --- tag ---
resource "britive_tag" "main" {
  name                 = "MIGS - Tag"
  description          = "MIGS - Tag Description"
  identity_provider_id = data.britive_identity_provider.main.id
}

# --- tag_member ---
resource "britive_tag_member" "main" {
  tag_id   = britive_tag.main.id
  username = "britiveprovideracceptancetest"
}

# --- permission ---
resource "britive_permission" "main" {
  name        = "MIGS - Permission"
  description = "MIGS - Permission Description"
  consumer    = "secretmanager"
  resources   = ["*"]
  actions     = ["authz.policy.list", "authz.policy.read", "sm.secret.read"]
}

# --- permission (used by role) ---
resource "britive_permission" "for_role" {
  name        = "MIGS - Role Permission"
  description = "MIGS - Role Permission Description"
  consumer    = "secretmanager"
  resources   = ["*"]
  actions     = ["authz.policy.list", "sm.secret.read"]
}

# --- role ---
resource "britive_role" "main" {
  name        = "MIGS - Role"
  description = "MIGS - Role Description"
  permissions = jsonencode([{ name = britive_permission.for_role.name }])
}

# --- policy ---
resource "britive_policy" "main" {
  name        = "MIGS - Policy"
  description = "MIGS - Policy Description"
  access_type = "Allow"
  is_active   = true
  is_draft    = false
  roles       = jsonencode([{ name = britive_role.main.name }])
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
      "dateSchedule" : {
        "fromDate" : "${DATE_FROM}",
        "toDate"   : "${DATE_TO}",
        "timezone" : "Asia/Calcutta"
      }
    }
  })
}

# --- application (Okta) ---
resource "britive_application" "okta" {
  application_type = "okta"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "MIGS - Okta App"
  }
  properties {
    name  = "description"
    value = "MIGS - Okta App Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1000
  }
}

# --- application (Snowflake — for entity_group) ---
resource "britive_application" "sf_for_eg" {
  application_type = "Snowflake Standalone"
  version          = "1.0"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "MIGS - SF EG App"
  }
  properties {
    name  = "description"
    value = "MIGS - SF EG App Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1500
  }
}

# --- application (Snowflake — for entity_environment) ---
resource "britive_application" "sf_for_ee" {
  application_type = "Snowflake Standalone"
  version          = "1.0"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "MIGS - SF EE App"
  }
  properties {
    name  = "description"
    value = "MIGS - SF EE App Description"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1500
  }
}

# --- entity_group ---
resource "britive_entity_group" "main" {
  application_id     = britive_application.sf_for_eg.id
  entity_name        = "MIGS - Entity Group"
  entity_description = "MIGS - Entity Group Description"
  parent_id          = britive_application.sf_for_eg.entity_root_environment_group_id
}

# --- entity_environment ---
resource "britive_entity_environment" "main" {
  application_id  = britive_application.sf_for_ee.id
  parent_group_id = britive_application.sf_for_ee.entity_root_environment_group_id
  properties {
    name  = "displayName"
    value = "MIGS - Entity Env"
  }
  properties {
    name  = "description"
    value = "MIGS - Entity Env Description"
  }
  properties {
    name  = "accountId"
    value = "migs-acct-id"
  }
}

# --- profile (Azure — for profile_permission) ---
resource "britive_profile" "for_azure" {
  app_container_id    = data.britive_application.azure.id
  name                = "MIGS - Azure Profile"
  description         = "MIGS - Azure Profile Description"
  expiration_duration = "25m0s"
  associations {
    type  = "EnvironmentGroup"
    value = "QA"
  }
}

# --- profile (AWS — for session_attribute, profile_policy, prioritization) ---
resource "britive_profile" "for_aws" {
  app_container_id    = data.britive_application.aws.id
  name                = "MIGS - AWS Profile"
  description         = "MIGS - AWS Profile Description"
  expiration_duration = "25m0s"
  associations {
    type  = "EnvironmentGroup"
    value = "Root"
  }
}

# --- profile (GCP — for additional_settings, constraint, supported_constraints) ---
resource "britive_profile" "for_gcp" {
  app_container_id    = data.britive_application.gcp.id
  name                = "MIGS - GCP Profile"
  description         = "MIGS - GCP Profile Description"
  expiration_duration = "25m0s"
  associations {
    type  = "EnvironmentGroup"
    value = "britive-gdev-cis.net"
  }
}

# --- profile_permission (Azure) ---
resource "britive_profile_permission" "azure" {
  profile_id      = britive_profile.for_azure.id
  permission_name = "Application Developer"
  permission_type = "role"
}

# --- profile_permission (GCP Storage — for constraint) ---
resource "britive_profile_permission" "gcp_storage" {
  profile_id      = britive_profile.for_gcp.id
  permission_name = "Storage Admin"
  permission_type = "role"
}

# --- profile_permission (GCP BigQuery — for supported_constraints data source) ---
resource "britive_profile_permission" "gcp_bq" {
  profile_id      = britive_profile.for_gcp.id
  permission_name = "BigQuery Data Owner"
  permission_type = "role"
}

# 7. supported_constraints data source
data "britive_supported_constraints" "main" {
  profile_id      = britive_profile.for_gcp.id
  permission_name = britive_profile_permission.gcp_bq.permission_name
  permission_type = britive_profile_permission.gcp_bq.permission_type
}

# --- constraint ---
resource "britive_constraint" "main" {
  profile_id      = britive_profile.for_gcp.id
  permission_name = britive_profile_permission.gcp_storage.permission_name
  permission_type = britive_profile_permission.gcp_storage.permission_type
  constraint_type = "condition"
  title           = "MIGSConditionConstraint"
  description     = "MIGS Condition Constraint Description"
  expression      = "${CONSTRAINT_EXP}"
}

# --- profile_session_attribute ---
resource "britive_profile_session_attribute" "main" {
  profile_id     = britive_profile.for_aws.id
  attribute_name = "Date Of Birth"
  mapping_name   = "dob"
  transitive     = true
}

# --- profile_additional_settings ---
resource "britive_profile_additional_settings" "main" {
  profile_id              = britive_profile.for_gcp.id
  use_app_credential_type = false
  console_access          = true
  programmatic_access     = false
}

# --- advanced_settings (APPLICATION level) ---
resource "britive_advanced_settings" "main" {
  resource_type = "APPLICATION"
  resource_id   = data.britive_application.aws.id
}

# --- profile_policy (3 policies on AWS profile for prioritization) ---
resource "britive_profile_policy" "aws_0" {
  policy_name  = "MIGS - Profile Policy 0"
  description  = "MIGS - Profile Policy 0 Desc"
  profile_id   = britive_profile.for_aws.id
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
      "dateSchedule" : {
        "fromDate" : "${DATE_FROM}",
        "toDate"   : "${DATE_TO}",
        "timezone" : "Asia/Calcutta"
      }
    }
  })
}

resource "britive_profile_policy" "aws_1" {
  policy_name  = "MIGS - Profile Policy 1"
  description  = "MIGS - Profile Policy 1 Desc"
  profile_id   = britive_profile.for_aws.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
}

resource "britive_profile_policy" "aws_2" {
  policy_name  = "MIGS - Profile Policy 2"
  description  = "MIGS - Profile Policy 2 Desc"
  profile_id   = britive_profile.for_aws.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
}

# --- profile_policy_prioritization ---
resource "britive_profile_policy_prioritization" "main" {
  profile_id = britive_profile.for_aws.id

  policy_priority {
    id       = britive_profile_policy.aws_0.id
    priority = 0
  }
  policy_priority {
    id       = britive_profile_policy.aws_1.id
    priority = 1
  }
  policy_priority {
    id       = britive_profile_policy.aws_2.id
    priority = 2
  }
}

# ═══════════════════════════════════════════════════════════
# RESOURCE MANAGER RESOURCES
# ═══════════════════════════════════════════════════════════

# --- response_template ---
resource "britive_resource_manager_response_template" "main" {
  name                      = "MIGS-Response-Template"
  description               = "MIGS-Response-Template-Description"
  is_console_access_enabled = true
  show_on_ui                = false
  template_data             = "The user {{YS}} has the {{admin}}."
}

# --- resource_type ---
resource "britive_resource_manager_resource_type" "main" {
  name        = "MIGS-Resource-Type"
  description = "MIGS-Resource-Type-Description"
  parameters {
    param_name   = "field1"
    param_type   = "string"
    is_mandatory = true
  }
  parameters {
    param_name   = "field2"
    param_type   = "password"
    is_mandatory = false
  }
}

# --- resource_type_permission ---
resource "britive_resource_manager_resource_type_permission" "main" {
  name                = "MIGS-Type-Permission"
  resource_type_id    = britive_resource_manager_resource_type.main.id
  description         = "MIGS-Type-Permission-Description"
  checkin_time_limit  = 180
  checkout_time_limit = 360
  is_draft            = false
  show_orig_creds     = true
  variables           = ["var1", "var2"]
  code_language       = "python"
  checkin_code        = "#!/bin/bash\necho done"
  checkout_code       = "#!/bin/bash\necho start"
  response_templates  = [britive_resource_manager_response_template.main.name]
}

# --- resource_label (Production + Development) ---
resource "britive_resource_manager_resource_label" "main" {
  name        = "MIGS-Resource-Label"
  description = "MIGS-Resource-Label-Description"
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

# --- resource_label (us-east-1 — for rm_profile associations + rm_policy resource_labels) ---
resource "britive_resource_manager_resource_label" "us_east" {
  name        = "MIGS-Label-US-East"
  description = "MIGS-Label-US-East-Description"
  label_color = "#1a2b3c"
  values {
    name        = "us-east-1"
    description = "US East 1"
  }
}

# --- resource ---
resource "britive_resource_manager_resource" "main" {
  name          = "MIGS-Resource"
  description   = "MIGS-Resource-Description"
  resource_type = britive_resource_manager_resource_type.main.name
  parameter_values = {
    "field1" = "value1"
  }
  resource_labels = {
    (britive_resource_manager_resource_label.main.name) = "Production"
  }
}

# --- resource_broker_pools ---
resource "britive_resource_manager_resource_broker_pools" "main" {
  resource_id  = britive_resource_manager_resource.main.id
  broker_pools = ["DO NOT DELETE - BROKER POOL TF Plugin"]
}

# --- resource_policy ---
resource "britive_resource_manager_resource_policy" "main" {
  policy_name  = "MIGS-Resource-Policy"
  description  = "MIGS-Resource-Policy-Description"
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
      "dateSchedule" : {
        "fromDate" : "${DATE_FROM}",
        "toDate"   : "${DATE_TO}",
        "timezone" : "Asia/Calcutta"
      }
    }
  })
  resource_labels {
    label_key = britive_resource_manager_resource_label.main.name
    values    = ["Production"]
  }
}

# --- rm_profile (main — for rm_policy + rm_prioritization) ---
resource "britive_resource_manager_profile" "main" {
  name                             = "MIGS-RM-Profile-Main"
  description                      = "MIGS-RM-Profile-Main-Description"
  expiration_duration              = 10800000
  extendable                       = true
  notification_prior_to_expiration = "1h0m0s"
  extension_duration               = "2h0m0s"
  extension_limit                  = 2
  allow_impersonation              = true
  associations {
    label_key = britive_resource_manager_resource_label.main.name
    values    = ["Production", "Development"]
  }
  associations {
    label_key = britive_resource_manager_resource_label.us_east.name
    values    = ["us-east-1"]
  }
}

# --- rm_profile (for_perm — uses Resource-Type label so type_perm can be assigned) ---
resource "britive_resource_manager_profile" "for_perm" {
  name                             = "MIGS-RM-Profile-ForPerm"
  description                      = "MIGS-RM-Profile-ForPerm-Description"
  expiration_duration              = 10800000
  extendable                       = true
  notification_prior_to_expiration = "1h0m0s"
  extension_duration               = "2h0m0s"
  extension_limit                  = 2
  allow_impersonation              = true
  associations {
    label_key = "Resource-Type"
    values    = [britive_resource_manager_resource_type.main.name]
  }
}

# 8. resource_manager_profile_permissions data source
data "britive_resource_manager_profile_permissions" "main" {
  profile_id = britive_resource_manager_profile.for_perm.id
}

# --- rm_profile_permission ---
resource "britive_resource_manager_profile_permission" "main" {
  profile_id = britive_resource_manager_profile.for_perm.id
  name       = britive_resource_manager_resource_type_permission.main.name
  version    = "LOCAL"
  variables {
    name              = "var1"
    value             = "val1"
    is_system_defined = false
  }
  variables {
    name              = "var2"
    value             = "val2"
    is_system_defined = false
  }
}

# --- rm_profile_policy (2 policies for prioritization) ---
resource "britive_resource_manager_profile_policy" "policy_0" {
  profile_id   = britive_resource_manager_profile.main.id
  policy_name  = "MIGS - RM Policy 0"
  description  = "MIGS - RM Policy 0 Desc"
  access_type  = "Allow"
  consumer     = "resourceprofile"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
  resource_labels {
    label_key = britive_resource_manager_resource_label.us_east.name
    values    = ["us-east-1"]
  }
}

resource "britive_resource_manager_profile_policy" "policy_1" {
  profile_id   = britive_resource_manager_profile.main.id
  policy_name  = "MIGS - RM Policy 1"
  description  = "MIGS - RM Policy 1 Desc"
  access_type  = "Allow"
  consumer     = "resourceprofile"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode({ users = [{ name = "britiveprovideracceptancetest" }] })
  resource_labels {
    label_key = britive_resource_manager_resource_label.us_east.name
    values    = ["us-east-1"]
  }
}

# --- rm_profile_policy_prioritization ---
resource "britive_resource_manager_profile_policy_prioritization" "main" {
  profile_id = britive_resource_manager_profile.main.id

  policy_priority {
    id       = britive_resource_manager_profile_policy.policy_0.id
    priority = 0
  }
  policy_priority {
    id       = britive_resource_manager_profile_policy.policy_1.id
    priority = 1
  }
}
MEOF

ok "main.tf written ($(wc -l < "$WORK_DIR/main.tf") lines)"
echo ""
echo "  Resource/datasource coverage:"
echo "    Data sources  (8): identity_provider, application×3, connection, all_connections,"
echo "                        supported_constraints, resource_manager_profile_permissions"
echo "    Core         (16): tag, tag_member, permission×2, role, policy, application×3,"
echo "                        entity_group, entity_environment, profile×3, profile_permission×3,"
echo "                        constraint, profile_session_attribute, profile_additional_settings,"
echo "                        advanced_settings, profile_policy×3, profile_policy_prioritization"
echo "    Resource Mgr (10): response_template, resource_type, resource_type_permission,"
echo "                        resource_label×2, resource, resource_broker_pools, resource_policy,"
echo "                        rm_profile×2, rm_profile_permission, rm_profile_policy×2,"
echo "                        rm_profile_policy_prioritization"
echo "  (escalation_policy data source excluded — requires IM-type connection)"

# ═══════════════════════════════════════════════════════════
# PHASE 0: Build v2.2.9 binary from main branch
# ═══════════════════════════════════════════════════════════
step "Phase 0: Building v2.2.9 binary from 'main' branch"
cd "$PROVIDER_SRC"

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
  warn "Currently on '${CURRENT_BRANCH}', switching to 'main'"
  git stash 2>/dev/null || true
  git checkout main || fail "Failed to checkout main"
fi
git pull origin main --ff-only 2>&1 | tail -3 || warn "git pull: already up to date or failed"
ok "Branch: $(git rev-parse --abbrev-ref HEAD)  commit: $(git log --oneline -1)"

warn "Syncing vendor directory for main..."
chmod -R u+w vendor/ 2>/dev/null || true
go mod vendor 2>&1 | tail -3
ok "vendor synced"

go build -o terraform-provider-britive . || fail "go build failed (main)"
ok "v2.2.9 binary built: $(ls -la terraform-provider-britive | awk '{print $5, $6, $7, $8}')"

# ═══════════════════════════════════════════════════════════
# PHASE 1: Apply + plan with v2.2.9
# ═══════════════════════════════════════════════════════════
step "Phase 1: terraform init (v2.2.9 binary via dev_overrides)"
cd "$WORK_DIR"
terraform init -no-color 2>&1 | grep -v "^$" | grep -v "^Initializing" || true
ok "init complete"

step "Phase 1: terraform apply with v2.2.9 (creating all resources)"
terraform apply -auto-approve -no-color 2>&1 | tee /tmp/migs_apply_v229.log
if ! grep -q "Apply complete!" /tmp/migs_apply_v229.log; then
  tail -50 /tmp/migs_apply_v229.log
  fail "terraform apply failed with v2.2.9 — see /tmp/migs_apply_v229.log"
fi
ok "Apply complete"
grep "Apply complete" /tmp/migs_apply_v229.log | head -1 | sed 's/^/  /'

step "Phase 1: terraform plan with v2.2.9 — expecting NO changes"
set +e
terraform plan -detailed-exitcode -no-color 2>&1 | tee /tmp/migs_plan_v229.log
PLAN_EXIT=${PIPESTATUS[0]}
set -e
case $PLAN_EXIT in
  0) ok "No changes — v2.2.9 plan is clean ✓" ;;
  1) tail -40 /tmp/migs_plan_v229.log
     fail "terraform plan errored with v2.2.9" ;;
  2) echo ""
     cat /tmp/migs_plan_v229.log
     fail "DRIFT DETECTED with v2.2.9 (exit 2 = changes pending) — see /tmp/migs_plan_v229.log" ;;
esac

# ═══════════════════════════════════════════════════════════
# PHASE 2: Build v3.0.0 binary from migration_framework
# ═══════════════════════════════════════════════════════════
step "Phase 2: Building v3.0.0 binary from 'migration_framework' branch"
cd "$PROVIDER_SRC"

git checkout migration_framework || fail "Failed to checkout migration_framework"
git pull origin migration_framework --ff-only 2>&1 | tail -3 || warn "git pull: already up to date or failed"
ok "Branch: $(git rev-parse --abbrev-ref HEAD)  commit: $(git log --oneline -1)"

warn "Syncing vendor directory for migration_framework..."
chmod -R u+w vendor/ 2>/dev/null || true
go mod vendor 2>&1 | tail -3
ok "vendor synced"

go build -o terraform-provider-britive . || fail "go build failed (migration_framework)"
ok "v3.0.0 binary built: $(ls -la terraform-provider-britive | awk '{print $5, $6, $7, $8}')"

# ═══════════════════════════════════════════════════════════
# PHASE 3: Plan with v3.0.0 (migration validation)
# ═══════════════════════════════════════════════════════════
step "Phase 3: terraform plan with v3.0.0 — expecting NO changes (migration validation)"
cd "$WORK_DIR"
echo "  dev_overrides now points to v3.0.0 binary (rebuilt from migration_framework)"
set +e
terraform plan -detailed-exitcode -no-color 2>&1 | tee /tmp/migs_plan_v300.log
PLAN_EXIT=${PIPESTATUS[0]}
set -e
case $PLAN_EXIT in
  0) ok "No changes — v3.0.0 migration plan is clean ✓" ;;
  1) tail -40 /tmp/migs_plan_v300.log
     fail "terraform plan errored with v3.0.0" ;;
  2) echo ""
     cat /tmp/migs_plan_v300.log
     fail "MIGRATION DRIFT DETECTED — v3.0.0 shows changes vs v2.2.9 state — see /tmp/migs_plan_v300.log" ;;
esac

step "Phase 3: terraform apply with v3.0.0 (confirm apply works cleanly)"
terraform apply -auto-approve -no-color 2>&1 | tee /tmp/migs_apply_v300.log
if ! grep -q "Apply complete!" /tmp/migs_apply_v300.log; then
  tail -50 /tmp/migs_apply_v300.log
  fail "terraform apply failed with v3.0.0"
fi
ok "v3.0.0 apply complete"
grep "Apply complete" /tmp/migs_apply_v300.log | head -1 | sed 's/^/  /'

step "Phase 3: terraform plan post v3.0.0 apply — expecting NO changes (idempotency)"
set +e
terraform plan -detailed-exitcode -no-color 2>&1 | tee /tmp/migs_plan_v300_post.log
PLAN_EXIT=${PIPESTATUS[0]}
set -e
case $PLAN_EXIT in
  0) ok "No changes — v3.0.0 is fully idempotent ✓" ;;
  1) tail -40 /tmp/migs_plan_v300_post.log
     fail "terraform plan errored" ;;
  2) echo ""
     cat /tmp/migs_plan_v300_post.log
     fail "POST-APPLY DRIFT with v3.0.0 — see /tmp/migs_plan_v300_post.log" ;;
esac

# ═══════════════════════════════════════════════════════════
# RESULT
# ═══════════════════════════════════════════════════════════
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        ✅  MIGRATION E2E TEST PASSED                     ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║  Phase 1 (v2.2.9 / main)                                ║${NC}"
echo -e "${GREEN}║    • All resources created successfully                  ║${NC}"
echo -e "${GREEN}║    • Plan: no drift                                      ║${NC}"
echo -e "${GREEN}║  Phase 2 (v3.0.0 / migration_framework)                 ║${NC}"
echo -e "${GREEN}║    • Migration plan: no drift                            ║${NC}"
echo -e "${GREEN}║    • Apply: successful                                   ║${NC}"
echo -e "${GREEN}║    • Post-apply plan: no drift (idempotent)              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "  Log files:"
echo "    /tmp/migs_apply_v229.log      — v2.2.9 apply"
echo "    /tmp/migs_plan_v229.log       — v2.2.9 plan (clean)"
echo "    /tmp/migs_plan_v300.log       — v3.0.0 migration plan (clean)"
echo "    /tmp/migs_apply_v300.log      — v3.0.0 apply"
echo "    /tmp/migs_plan_v300_post.log  — v3.0.0 post-apply plan (clean)"
echo ""
warn "terraform destroy will run automatically during cleanup on exit..."
