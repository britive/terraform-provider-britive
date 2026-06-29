
## 2.3.4

ENHANCEMENTS:
* **Resource/Data Source:** `britive_application`, `britive_profile`, `britive_profile_policy`, `britive_profile_permission`, `britive_profile_session_attribute`, `britive_entity_group`, `britive_entity_environment` : Application API calls now use `?view=minimized` for smaller response payloads. Environment and root group data is now fetched from the main application response, eliminating separate calls to the `/root-environment-group` endpoint.
* **Provider:** Added client-side handling for API rate limiting (HTTP 429): retry with backoff, honoring `Retry-After`, configurable via `max_retries` / `retry_wait_min` / `retry_wait_max`. Dormant until rate limiting is enabled server-side by Britive; no behavioral change on upgrade.

=======

## 2.3.3

ENHANCEMENTS:
* **Resource:** `britive_profile` : Added `tag_associations` argument to associate tag-based scope filters with a profile. At least one of `associations` or `tag_associations` must be specified (previously `associations` was required).
* **Resource:** `britive_profile_policy` : Added `tag_associations` argument to associate tag-based scope filters with a profile policy.
* **Resource:** `britive_application` : Added support for `Britive` as a new `application_type`.

=======

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

=======

## 2.3.1

ENHANCEMENTS:
* **Resource:** `britive_tag` : Added `requestable` argument to control whether the tag is requestable
* **Resource:** `britive_tag` : Added `attributes` argument to associate one or more name/value attribute pairs with a tag, with support for multi-valued attributes

=======

## 2.3.0

FEATURES:
* **New Resource:** `britive_tag_owner` : Create, update, and manage owners (users and tags) of a Britive tag
* **New Data Source:** `britive_tag` : Look up a Britive tag by name or id
* **New Data Source:** `britive_user` : Look up a Britive user by name or id
