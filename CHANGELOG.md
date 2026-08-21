## 3.0.1

BUG FIXES:
* **Resource:** `britive_profile`, `britive_tag_owner`, `britive_resource_manager_resource_type_permission`, `britive_resource_manager_profile_permission`, `britive_resource_manager_resource_label` : Fixed a "Value Conversion Error ... Received unknown value, however the target type cannot handle unknown values" failure during `terraform validate` and `terraform plan`. It affected any resource declared with `for_each` whose nested blocks are populated from `each.value` (e.g. a `dynamic "associations"` block on `britive_profile`), and any nested block whose `for_each` is not resolved until apply. Terraform evaluates such configurations without instance-expansion context, so the block legitimately arrives as an unknown value; the provider now tolerates it and defers those checks to a later plan instead of erroring. No configuration or state changes are required.

## 3.0.0

BREAKING CHANGES:
* Provider migrated from Terraform Plugin SDK v2 to Terraform Plugin Framework (Protocol v6). This is a major internal rewrite with no changes to the HCL resource/data source schemas. Existing Terraform state is compatible; however, the provider binary now requires Terraform >= 1.0 and uses protocol version 6.

ENHANCEMENTS:
* All 28 resources and 10 data sources fully rewritten using the Terraform Plugin Framework for improved type safety, plan modifiers, and validator support.
* `britive_resource_manager_profile_permission`: Added `prompt_at_checkout` argument to a permission's `variables`, so a variable's value can be supplied by the user at checkout time instead of being configured in Terraform.
* `britive_resource_manager_resource_type_permission`: Added support for declaring a password-type permission variable by suffixing its name with `:password` in `variables` (e.g. `variables = ["test1", "test2:password"]`); `britive_resource_manager_profile_permission` then references it by its base name and exposes the resolved type via the variable's `type` attribute.
* `britive_permission`: Added `permission_scopes` argument to restrict a permission to one or more application types (e.g. `AWS`, `Azure`, `GCP`). Only valid when `consumer` is `apps` and `resources` is `["*"]`.

BUG FIXES:
* `britive_resource_manager_response_template`: Fixed `template_id` never being populated in state (it was declared in the schema but not set on read since v2.3.x); this will appear as a one-time value population on the first plan/refresh after upgrading.

## 2.3.6

DEPRECATIONS:
* **Resource:** `britive_application` : The `scanOrganization` and `scanProjectsOnly` properties are no longer supported for `GCP WIF` applications and must be removed from the `properties` block in existing configurations. These properties remain supported for the standard `GCP` application type.

## 2.3.5

ENHANCEMENTS:
* **Resource:** `britive_application` : Added support for `Azure WIF` and `Oracle WIF`, enabling Workload Identity Federation app management for Azure and Oracle Cloud environments.
* **Resource:** `britive_resource_manager_profile` : Added `exclusive_checkout` argument to control exclusive checkout behavior for resource manager profiles.

## 2.3.4

ENHANCEMENTS:
* **Resource/Data Source:** `britive_application`, `britive_profile`, `britive_profile_policy`, `britive_profile_permission`, `britive_profile_session_attribute`, `britive_entity_group`, `britive_entity_environment` : Application API calls now use `?view=minimized` for smaller response payloads. Environment and root group data is now fetched from the main application response, eliminating separate calls to the `/root-environment-group` endpoint.
* **Provider:** Added client-side handling for API rate limiting (HTTP 429): retry with backoff, honoring `Retry-After`, configurable via `max_retries` / `retry_wait_min` / `retry_wait_max`. Dormant until rate limiting is enabled server-side by Britive; no behavioral change on upgrade.

## 2.3.3

ENHANCEMENTS:
* **Resource:** `britive_profile` : Added `tag_associations` argument to associate tag-based scope filters with a profile. At least one of `associations` or `tag_associations` must be specified (previously `associations` was required).
* **Resource:** `britive_profile_policy` : Added `tag_associations` argument to associate tag-based scope filters with a profile policy.
* **Resource:** `britive_application` : Added support for `Britive` as a new `application_type`.

## 2.3.2

FEATURES:
* **New Data Source:** `britive_user_attribute` : Look up a Britive user attribute by name or attribute schema ID

ENHANCEMENTS:
* `britive_application` data source now supports lookup by either `name` or `app_container_id`
* `britive_profile_policy` adds optional `app_container_id` to avoid profile->application lookup calls
* `britive_tag_member` adds optional `user_id` to avoid username->user_id lookup calls
* `britive_profile_session_attribute` adds optional `attribute_schema_id` to avoid attribute-name lookup calls for identity attributes
* Added ID-first import formats for `britive_profile`, `britive_profile_permission`, and `britive_tag_member`

DEPRECATIONS:
* Name-based fallback lookups remain supported for backward compatibility but now emit warnings encouraging ID-first configuration
  * `britive_profile_policy`: fallback from `profile_id` to resolve `app_container_id`
  * `britive_tag_member`: fallback from `username` to resolve `user_id`
  * `britive_profile_session_attribute`: fallback from `attribute_name` to resolve `attribute_schema_id`
  * Legacy name-based import formats are still accepted where documented

## 2.3.1

ENHANCEMENTS:
* **Resource:** `britive_tag` : Added `requestable` argument to control whether the tag is requestable
* **Resource:** `britive_tag` : Added `attributes` argument to associate one or more name/value attribute pairs with a tag, with support for multi-valued attributes

## 2.3.0

FEATURES:
* **New Resource:** `britive_tag_owner` : Create, update, and manage owners (users and tags) of a Britive tag
* **New Data Source:** `britive_tag` : Look up a Britive tag by name or id
* **New Data Source:** `britive_user` : Look up a Britive user by name or id

## 2.2.9

FEATURES:
* **New Resource:** `britive_resource_manager_profile_policy_prioritization`: Create, update, and manage resource manager profile policy prioritization.

ENHANCEMENTS:
* `britive_resource_manager_profile`: Support for configuring session extensions, including the fields `extendable`, `extension_duration`, `extension_limit` and `notification_prior_to_expiration`.

## 2.2.8

ENHANCEMENTS:
* `britive_resource_manager_resource_type`: Extended support for creating and managing dynamic resource types.

BUG FIXES:
* `britive_resource_manager_resource_type`: Fixed issues with resource type imports.

## 2.2.7

ENHANCEMENTS:
* `britive_application`: Support extended to create and manage applications of type GCP WIF.

## 2.2.3

FEATURES:
* **New Resource:** `britive_profile_policy_prioritization`: Create, update, and manage profile policy prioritization.

## 2.2.2

ENHANCEMENTS:
* Documentation restructure.

## 2.2.0

FEATURES:
* **New Resource:** `britive_resource_manager_response_template`: Create, update, and manage resource manager response templates.
* **New Resource:** `britive_resource_manager_resource_type`: Create, update, and manage resource manager resource types.
* **New Resource:** `britive_resource_manager_resource_type_permission`: Create, update, and manage resource manager resource type permissions.
* **New Resource:** `britive_resource_manager_resource_label`: Create, update, and manage resource manager resource labels.
* **New Resource:** `britive_resource_manager_resource`: Create, update, and manage resource manager resources.
* **New Resource:** `britive_resource_manager_resource_policy`: Create, update, and manage resource manager resource policies.
* **New Resource:** `britive_resource_manager_profile`: Create, update, and manage resource manager profiles.
* **New Resource:** `britive_resource_manager_profile_permission`: Create, update, and manage resource manager profile permissions.
* **New Resource:** `britive_resource_manager_profile_policy`: Create, update, and manage resource manager profile policies.
* **New Resource:** `britive_resource_manager_resource_broker_pools`: Create, update, and manage resource manager broker pools.
* **New Data Source:** `britive_escalation_policy`: Retrieve information about a specific escalation policy required for configuring IM settings.
* **New Data Source:** `britive_resource_manager_profile_permissions`: Retrieve the permissions available for a specific profile.

ENHANCEMENTS:
* `britive_advanced_settings`: Support for IM settings. Allowing configuration of advanced settings for `RESOURCE_MANAGER_PROFILE` and `RESOURCE_MANAGER_PROFILE_POLICY`.
* `britive_connection` (data source): Support to fetch IM settings.
* `britive_all_connections` (data source): Support to fetch IM settings.

## 2.1.8

ENHANCEMENTS:
* `britive_profile_policy`: Support for `managerApproval` in approval config.

BUG FIXES:
* `britive_advanced_settings`: Fix to enable clearing of `justification_regex`.
* `britive_advanced_settings`: Re-create the advanced settings when the `resource_type` changes.

## 2.1.3

FEATURES:
* **New Resource:** `britive_profile_additional_settings`: Configure the additional settings (console and programmatic access) associated with a profile.

## 2.1.2

ENHANCEMENTS:
* `britive_profile_policy`: Terraform support for profile optimization.

## 2.1.1

ENHANCEMENTS:
* `britive_profile_policy`: Support for `slackAppChannels` and `teamsAppChannels` in profile policy.

BUG FIXES:
* `britive_constraint`, `britive_supported_constraints` (data source): Documentation hyperlink fix for TF Plugin resource/data source name for constraints.

## 2.0.9

ENHANCEMENTS:
* `britive_profile`: Include `AccountID` as an association value for AWS Standalone applications.

BUG FIXES:
* `britive_profile_permission`: Terraform plan fails after creation of a profile permission with a `/` in its name.

## 2.0.8

BUG FIXES:
* `britive_policy`, `britive_profile_policy`, `britive_role`: Preserve the order as provided in the configuration.
* `britive_profile`: Consistency of root value of `EnvironmentGroup` in AWS Standalone and AWS Org apps.
* `britive_policy`, `britive_profile_policy`: Avoid diff seen in IP address list having space after comma.
* `britive_policy`, `britive_profile_policy`: Approval block removal not reflecting in the application.
* `britive_profile_permission`: Documentation update — profile permission type restriction.

## 2.0.6

BUG FIXES:
* `britive_profile`: Diff shown for all associations when one association is added or removed.
* `britive_profile`: Documentation update specifying not to use extendable properties for AWS profiles.

## 2.0.5

BUG FIXES:
* `britive_profile_policy`: Unable to change profile policy name from Terraform.

## 2.0.4

BUG FIXES:
* provider: Terraform Plan/Apply always shows changes.
* `britive_policy`, `britive_profile_policy`: Terraform provider gives whitespace-change diffs for conditions under policy/profile-policy.

## 2.0.3

ENHANCEMENTS:
* `britive_policy`: Update `timeOfAccess` to include `dateSchedule` and `daysSchedule`.
* `britive_profile_policy`: Update `timeOfAccess` to include `dateSchedule` and `daysSchedule`.

## 2.0.2

ENHANCEMENTS:
* `britive_policy`: Added variable `isValidForInDays` to support approval validity in days.
* `britive_profile_policy`: Added variable `isValidForInDays` to support approval validity in days.

## 2.0.1

BUG FIXES:
* provider: Terraform Plan/Apply gives intermittent time-outs.

## 2.0.0

FEATURES:
* **New Resource:** `britive_permission`
* **New Resource:** `britive_role`
* **New Resource:** `britive_policy`
* **New Resource:** `britive_profile_policy`
