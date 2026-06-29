
## 2.3.4

ENHANCEMENTS:
* **Client:** Application API calls now use `?view=minimized` to reduce response payload size. `GetApplicationRootEnvironmentGroup` and environment lookups are consolidated into a single `GetApplication` call, reducing unnecessary API round trips.
* **Provider:** Added `max_retries` (default: `10`), `retry_wait_min` (default: `1` s), and `retry_wait_max` (default: `600` s) arguments to configure client-side retry behavior for rate-limited (HTTP 429) API requests. The client retries with exponential backoff and full jitter, honoring the `Retry-After` response header when present.

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
