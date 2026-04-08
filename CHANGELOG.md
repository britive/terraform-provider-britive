## 2.3.0

FEATURES:
* **API Response Caching:** New optional `enable_cache` provider attribute to enable in-memory caching of frequently accessed API responses (root environment group, application environments, application type). Reduces redundant GET calls during plan/apply for configurations with many profiles and policies. Can be configured via provider block, `BRITIVE_CACHE` environment variable, or config file.

ENHANCEMENTS:
* **HTTP 429 Retry Handling:** The provider now automatically retries API requests that receive an HTTP 429 (Too Many Requests) response, respecting the `Retry-After` header from the Britive platform. Requests are retried up to 5 times with exponential backoff as a fallback. This is always enabled and requires no configuration.

## 2.2.9

FEATURES:
* **New Resource:**  `britive_resource_manager_profile_policy_prioritization` : Create, update, and manage resource manager profile policy prioritization

ENHANCEMENTS:
* `britive_resource_manager_profile` : Support for configuring session extensions, including the fields extendable, extension_duration, extension_limit and notification_prior_to_expiration.
