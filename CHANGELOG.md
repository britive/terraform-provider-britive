## 3.0.0

BREAKING CHANGES:
* Provider migrated from Terraform Plugin SDK v2 to Terraform Plugin Framework (Protocol v6). This is a major internal rewrite with no changes to the HCL resource/data source schemas. Existing Terraform state is compatible; however, the provider binary now requires Terraform >= 1.0 and uses protocol version 6.

ENHANCEMENTS:
* All 26 resources and 7 data sources fully rewritten using the Terraform Plugin Framework for improved type safety, plan modifiers, and validator support.
* `britive_application`: Added support for `GCP WIF`, `AWS`, `AWS Standalone`, `Azure`, and `Okta` application types with case-insensitive `application_type` validation.
* `britive_application`: Added `version` and `entity_root_environment_group_id` computed attributes with stable plan modifiers to eliminate drift.
* `britive_resource_manager_resource_policy`: Resource labels now use HCL block syntax (`resource_labels { }`) consistent with other block-style attributes.
* `britive_resource_manager_profile`: Added support for configuring session extensions via the fields `extendable`, `extension_duration`, `extension_limit`, and `notification_prior_to_expiration`.
* Sensitive properties (e.g. `privateKey`, `clientSecret`) continue to use argon2-based hashing as a plan modifier, matching prior SDK v2 `StateFunc` behaviour and preventing perpetual diffs.

BUG FIXES:
* Fixed perpetual plan drift for `resource_labels` in `britive_resource_manager_resource` when the Britive API returns label values in a different order than the configuration.
* Fixed "provider returned unknown value" errors for `Optional+Computed` fields (`app_name`, `profile_name`, `tag_name`) that are not returned by the API at create time.
* Fixed `attribute_value` / `attribute_name` inconsistency in `britive_profile_session_attribute` when clearing fields for Identity vs Static attribute types.
* Fixed `resource_type_id` causing perpetual plan drift after apply in `britive_resource_manager_resource`.

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
