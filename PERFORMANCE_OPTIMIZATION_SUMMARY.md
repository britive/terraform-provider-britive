## Terraform Provider Performance Optimization Summary

### What was accomplished

- Reduced repeated API calls in profile association logic (`britive/resources/resource_profile.go`)
  - Made app context lookups lazy so app root environment group/app type are fetched only when needed for `Environment`/`EnvironmentGroup` associations.
  - Added per-operation caching for `GetEnvId(...)` to avoid repeated env account lookups for duplicate values in a single resource operation.
  - Avoided unnecessary lookups when association types do not require that data.
  - Why this is safe: association validation/mapping behavior is unchanged; only call timing and duplicate-call elimination changed. Fallback and error paths remain intact.

- Reduced repeated API calls in profile policy association logic (`britive/resources/resource_profile_policy.go`)
  - Added optional `app_container_id` on resource.
  - Uses `profile_id -> app_container_id` fallback lookup only when `app_container_id` is not provided.
  - Added deprecation warning on fallback path.
  - Added `GetEnvId(...)` caching in association mapping loops.
  - Why this is safe: existing configs still resolve app ID through the old path, while explicit ID simply bypasses redundant lookup. No required-field change was introduced.

- Added explicit ID fields for high-traffic resources with backward-compatible fallback behavior
  - `britive/resources/resource_tag_member.go`
    - Added optional `user_id` (computed/force-new).
    - Create now uses `user_id` directly when provided.
    - Falls back to `GetUserByName(username)` with deprecation warning when missing.
    - Persists `user_id` in state on read.
  - `britive/resources/resource_profile_session_attribute.go`
    - Added optional `attribute_schema_id` (computed/force-new).
    - Identity attributes use `attribute_schema_id` directly when provided.
    - Falls back to `GetAttributeByName(attribute_name)` with deprecation warning when missing.
    - Persists `attribute_schema_id` in state.
  - Why this is safe: new fields are optional and additive; legacy name-based behavior still works via fallback. State convergence improves because resolved IDs are persisted.

- Added/expanded data sources so customers can resolve IDs once and reuse across many resources
  - Added new datasource `britive_user_attribute` (`britive/datasources/data_user_attribute.go`)
    - Lookup by `name` or `attribute_schema_id`.
  - Registered new datasource in provider (`britive/provider.go`).
  - Enhanced `britive_application` datasource (`britive/datasources/data_application.go`)
    - Added `app_container_id` input/output and supports lookup by `name` or `app_container_id`.
  - Why this is safe: datasource schema changes are additive (`ExactlyOneOf` guards invalid input), and existing name-based lookup remains supported.

- Removed redundant read-path API calls (large impact)
  - `britive/resources/resource_role.go`
  - `britive/resources/resource_permission.go`
  - `britive/resources/resource_policy.go`
  - Each previously did `GetX(id)` then `GetXByName(name)` on read; now each uses only `GetX(id)`.
  - Why this is safe: both methods hit equivalent object endpoints/models for these resources; the second call was not adding additional fields used by state mapping.

- Applied targeted lower-risk optimization
  - Removed immediate post-create `resourceRead(...)` in:
    - `britive/resources/resource_role.go`
    - `britive/resources/resource_permission.go`
    - `britive/resources/resource_policy.go`
  - Kept update-time reads unchanged.
  - Why this is safe: create responses for these resources are unmarshaled into full structs and include IDs/fields required for initial state; update/read behavior remains conservative.

- Importer optimization (ID-first import support + fallback warning)
  - `britive/resources/resource_profile.go`
    - Added import formats that accept `app_container_id` directly.
  - `britive/resources/resource_profile_permission.go`
    - Added import formats that accept `profile_id` directly.
  - `britive/resources/resource_tag_member.go`
    - Added import formats that accept `tag_id` + `user_id` directly.
  - Name-based import formats still work, with deprecation warnings encouraging migration.
  - Why this is safe: importer behavior is strictly expanded (more accepted formats), not replaced. Existing import strings continue to function.

- Fixed unrelated build/vet issues surfaced during validation
  - Corrected format strings for:
    - `britive/resources/resourcemanager/resource_resource_manager_profile.go:291`
    - `britive/resources/resourcemanager/resource_resource_manager_resource_label.go:159`
    - `britive/resources/resourcemanager/resource_resource_manager_resource_label.go:190`
  - Why this is safe: formatting-only fix to error message templates; no runtime control-flow or API behavior changes.

### Testing performed

- Ran full provider test/build in Docker:

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.22 /bin/sh -lc "/usr/local/go/bin/go test ./..."
```

- Initial run surfaced compile/vet issues; all identified issues were patched.
- Re-ran full Docker test command successfully:
  - All packages build cleanly.
  - `britive/tests` passed.

### Customer-facing Terraform definition changes

Backward compatibility is preserved. Existing configurations continue to work.

Customers can now reduce API volume by supplying explicit IDs:

- `britive_profile_policy`
  - New optional argument: `app_container_id`
  - Fallback behavior still supported when omitted (with deprecation warning).

- `britive_tag_member`
  - New optional argument: `user_id`
  - Fallback behavior still supported via `username` lookup (with deprecation warning).

- `britive_profile_session_attribute`
  - New optional argument: `attribute_schema_id`
  - Fallback behavior still supported via `attribute_name` lookup for Identity type (with deprecation warning).

Recommended lookup-once/reuse pattern:

- `data "britive_application" ...` -> `app_container_id`
- `data "britive_user" ...` -> `user_id`
- `data "britive_user_attribute" ...` -> `attribute_schema_id`

This allows customers to resolve identifiers once and feed them into multiple resources, significantly reducing repeated API calls during apply.

### Expected impact

- Fewer API calls in apply/read hot paths.
- Lower repeated name->ID and env-account lookup overhead.
- Better scaling behavior for large plans/applies.
- Safe migration path via deprecation warnings without immediate breaking changes.

### Per-operation cache explanation

The optimization includes a per-operation in-memory cache used during a single resource handler invocation (for example, one `CreateContext`, `ReadContext`, or `UpdateContext` call for one resource instance).

- What is cached:
  - Repeated `GetEnvId(appContainerID, accountID)` lookups for the same input within one association-processing loop.
- Scope/lifetime:
  - Local variable map inside the helper function.
  - Exists only for that one resource operation.
  - Discarded when the function returns.
- What it reduces:
  - Duplicate API calls for identical lookup inputs encountered multiple times in a single resource invocation.
- What it does not reduce:
  - Repeated lookups across different resources in the same apply.
  - Repeated lookups across subsequent operations (for example, a later `Read` after an `Update`).

This approach was selected because it provides immediate API-call reduction with very low correctness risk and no global cache invalidation/concurrency complexity.

Operational definition used in this work:

- Operation = one resource CRUD handler invocation for one resource instance (for example, one `CreateContext` call for one resource).

### Import format examples (old and new)

- `britive_profile`
  - Existing: `apps/<app_name>/paps/<profile_name>`
  - New (ID-first): `apps/app-container-id/<app_container_id>/paps/<profile_name>`

- `britive_profile_permission`
  - Existing: `apps/<app_name>/paps/<profile_name>/permissions/<permission_name>/type/<permission_type>`
  - New (ID-first): `paps/<profile_id>/permissions/<permission_name>/type/<permission_type>`

- `britive_tag_member`
  - Existing: `<tag_name>/<username>`
  - New (ID-first): `tags/<tag_id>/users/<user_id>`

### Release versioning guidance

- Recommended increment for this work: minor version bump (`vX.(Y+1).0`)
  - Rationale: changes are additive and backward-compatible (new optional fields, datasource addition, importer format expansion, performance improvements).

- Future major version trigger:
  - If/when deprecated fallback lookup paths are removed, use a major bump (`v(X+1).0.0`).

### Deprecation rollout plan

- Current release:
  - Fallback behavior retained when new ID fields are not provided.
  - Deprecation warnings emitted to prompt migration.

- Next phase (planned):
  - Document a timeline for fallback removal.
  - Communicate migration examples in release notes and docs.

- Breaking-change phase (future major):
  - Remove fallback lookup paths after migration window.

### Rollback and risk posture

- Risk profile:
  - Low to moderate. Changes are mostly call-pattern optimizations and additive schema updates.
  - Update-time reads were intentionally preserved in targeted areas to avoid state drift risk.

- Rollback strategy:
  - Code rollback is straightforward because no schema fields were removed.
  - Customer configs do not need emergency changes because legacy behavior paths remain available in this release.
