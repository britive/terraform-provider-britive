# SDKv2 → Plugin Framework Migration: Code Review

This document explains every architectural and code-level change made during the
migration of the Britive Terraform provider from **Terraform Plugin SDK v2** to the
**Terraform Plugin Framework** (Protocol v6), released as **v3.0.0**.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Repository Layout](#2-repository-layout)
3. [Entry Point — main.go](#3-entry-point--maingo)
4. [Provider Configuration](#4-provider-configuration)
5. [Schema Definition Patterns](#5-schema-definition-patterns)
6. [CRUD Operations](#6-crud-operations)
7. [State Management](#7-state-management)
8. [Import State](#8-import-state)
9. [Custom Validators](#9-custom-validators)
10. [Plan Modifiers](#10-plan-modifiers)
11. [Nested Blocks vs Nested Attributes](#11-nested-blocks-vs-nested-attributes)
12. [Sensitive Properties — Argon2 Hashing](#12-sensitive-properties--argon2-hashing)
13. [Default Values](#13-default-values)
14. [Config-Level Validation](#14-config-level-validation)
15. [Test Infrastructure](#15-test-infrastructure)
16. [Lifecycle & Idempotency Tests](#16-lifecycle--idempotency-tests)
17. [Continuous Integration](#17-continuous-integration)
18. [Bug Fixes Discovered During Testing](#18-bug-fixes-discovered-during-testing)
19. [Quick-Reference Pattern Map](#19-quick-reference-pattern-map)
20. [Rollback Strategy](#20-rollback-strategy)
21. [Post-Migration Changes (v3.0.0+)](#21-post-migration-changes-v300)

---

## 1. Overview

The Terraform Plugin Framework is HashiCorp's modern replacement for Plugin SDK v2.
It uses **Protocol v6** (vs SDKv2's Protocol v5), provides a strongly-typed Go API,
and treats validation, plan modification, and state handling as first-class concerns.

### What stayed the same

- The API client (`britive-client-go/`) is framework-agnostic and required **no changes**.
- All HCL resource/data source schemas are wire-compatible — existing Terraform state files
  are valid without migration.
- All 26 resources and 7 data sources expose identical attributes from the user's
  perspective.

### What changed

| Area | SDKv2 | Plugin Framework |
|------|-------|-----------------|
| Protocol version | v5 | v6 |
| Schema types | `schema.Schema` with `TypeString` etc. | Typed attribute structs (`schema.StringAttribute`) |
| State/plan access | `d *schema.ResourceData` with `d.Get()` / `d.Set()` | Typed Go structs with `tfsdk:` tags |
| Validators | `ValidateFunc` inline functions | `validator.String` interface |
| Plan modifiers | `DiffSuppressFunc` / `StateFunc` | `planmodifier.String` interface |
| Default values | `Default:` field on schema | `Default: stringdefault.StaticString(...)` |
| Import | `Importer: &schema.ResourceImporter{}` | `ImportState` method on resource |
| Testing | `Providers: testAccProviders` (SDKv2) | `ProtoV6ProviderFactories` (plugin testing) |

---

## 2. Repository Layout

All old SDKv2 files were renamed to `.bak` during development and deleted before the
v3.0.0 release. The new files follow a consistent naming convention.

```
britive/
├── provider_framework.go          # NEW — replaces provider.go (now deleted)
│
├── validators/                    # NEW — custom validator package
│   ├── duration.go                #   validates time.ParseDuration strings
│   ├── alphanumeric.go            #   validates [a-zA-Z0-9_-] strings
│   └── svg.go                     #   validates SVG XML + ≤400KB size
│
├── planmodifiers/                 # NEW — custom plan modifier package
│   └── sensitive_hash.go          #   argon2 hash modifier (replaces StateFunc)
│
├── datasources/                   # REWRITTEN — one file per data source
│   ├── identity_provider_data_source.go
│   ├── application_data_source.go
│   ├── supported_constraints_data_source.go
│   ├── connection_data_source.go
│   ├── all_connections_data_source.go
│   ├── escalation_policy_data_source.go
│   └── resource_manager_profile_permissions_data_source.go
│
├── resources/                     # REWRITTEN — one file per resource
│   ├── tag_resource.go
│   ├── constraint_resource.go
│   ├── tag_member_resource.go
│   ├── entity_group_resource.go
│   ├── entity_environment_resource.go
│   ├── policy_resource.go
│   ├── permission_resource.go
│   ├── role_resource.go
│   ├── profile_permission_resource.go
│   ├── profile_session_attribute_resource.go
│   ├── profile_additional_settings_resource.go
│   ├── advanced_settings_resource.go
│   ├── profile_policy_resource.go
│   ├── profile_policy_prioritization_resource.go
│   ├── application_resource.go
│   ├── profile_resource.go
│   └── resourcemanager/
│       ├── resource_type_resource.go
│       ├── resource_type_permissions_resource.go
│       ├── response_template_resource.go
│       ├── resource_label_resource.go
│       ├── resource_resource.go
│       ├── profile_resource.go
│       ├── resource_policy_resource.go
│       ├── profile_permission_resource.go
│       ├── profile_policy_resource.go
│       └── resource_type_resource_broker_pools.go
│
└── tests/
    ├── provider_framework_test.go  # NEW — replaces provider_test.go
    └── resource_*_test.go          # UPDATED — all use ProtoV6ProviderFactories
```

---

## 3. Entry Point — main.go

**SDKv2:**
```go
// Used plugin/serve with a schema.Provider factory
plugin.Serve(&plugin.ServeOpts{
    ProviderFunc: provider.New,
})
```

**Plugin Framework:**
```go
// Uses providerserver.Serve with a provider.Provider factory
providerserver.Serve(context.Background(), britive.New(version), providerserver.ServeOpts{
    Address: "registry.terraform.io/britive/britive",
    Debug:   debug,
})
```

Key differences:
- `providerserver` package replaces `plugin`
- `Address` is now specified at the serve level (previously in `terraform-registry-manifest.json`)
- `Debug` flag for attaching debuggers like Delve is now a standard option
- The version string is injected via `ldflags` at build time and threaded through
  `britive.New(version)` → stored on `BritiveProvider.version` → passed to `NewClient`

---

## 4. Provider Configuration

**File:** `britive/provider_framework.go`

### Interface compliance

```go
// Compile-time assertion that BritiveProvider satisfies the provider.Provider interface
var _ provider.Provider = &BritiveProvider{}
```

This pattern (blank identifier `_` assigned the interface) appears on every resource
and data source as well, catching missing method implementations at compile time rather
than at runtime.

### Provider struct and model

```go
type BritiveProvider struct {
    version string  // injected from ldflags, used in User-Agent header
}

// BritiveProviderModel is the typed Go struct for the provider config block.
// The tfsdk: tags are the mapping between HCL attribute names and struct fields.
type BritiveProviderModel struct {
    Tenant     types.String `tfsdk:"tenant"`
    Token      types.String `tfsdk:"token"`
    ConfigPath types.String `tfsdk:"config_path"`
}
```

In SDKv2 the provider used `*schema.ResourceData` — an untyped map with string keys.
The Framework uses concrete Go structs, giving compile-time field name safety.

### Schema method

```go
func (p *BritiveProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "tenant": schema.StringAttribute{Optional: true, ...},
            "token":  schema.StringAttribute{Optional: true, Sensitive: true, ...},
            ...
        },
    }
}
```

SDKv2 equivalent was a `func() *schema.Provider` returning a struct with a `Schema` map
of `*schema.Schema` objects. The Framework's typed attribute structs (`StringAttribute`,
`BoolAttribute`, etc.) replace the generic `schema.Schema{Type: schema.TypeString}`.

### Configure method

```go
func (p *BritiveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config BritiveProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

    // ... resolve tenant/token from env vars / config file ...

    // Auto-prepend https:// if no scheme present (fix for bare hostnames in env vars)
    if !strings.Contains(tenant, "://") {
        tenant = "https://" + tenant
    }

    client, err := britive.NewClient(apiBaseURL, token, p.version)

    // Make client available to all resources and data sources via ProviderData
    resp.DataSourceData = client
    resp.ResourceData = client
}
```

Notable changes vs SDKv2:
- Config is decoded into a typed struct via `req.Config.Get(ctx, &config)` instead of
  `d.Get("tenant").(string)`
- Errors are accumulated in `resp.Diagnostics` (which supports multiple errors) instead
  of `return nil, fmt.Errorf(...)`
- The client is distributed via `resp.DataSourceData` / `resp.ResourceData` — resources
  receive it in their `Configure` method via `req.ProviderData`
- `https://` auto-prepend added: the acceptance tests set `BRITIVE_TENANT` as a bare
  hostname; SDKv2 did not enforce this, but the Framework is stricter about URL validation

### Resource and data source registration

```go
func (p *BritiveProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        resources.NewTagResource,
        resources.NewConstraintResource,
        // ...26 total
    }
}

func (p *BritiveProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        datasources.NewIdentityProviderDataSource,
        // ...7 total
    }
}
```

SDKv2 used a `ResourcesMap` and `DataSourcesMap` in the provider struct (maps of
`string → *schema.Resource`). The Framework uses factory slices — each entry is a
`func() resource.Resource` that constructs a fresh instance per Terraform operation.

---

## 5. Schema Definition Patterns

### Attribute types

| SDKv2 | Plugin Framework |
|-------|-----------------|
| `schema.Schema{Type: schema.TypeString}` | `schema.StringAttribute{}` |
| `schema.Schema{Type: schema.TypeBool}` | `schema.BoolAttribute{}` |
| `schema.Schema{Type: schema.TypeInt}` | `schema.Int64Attribute{}` |
| `schema.Schema{Type: schema.TypeFloat}` | `schema.Float64Attribute{}` |
| `schema.Schema{Type: schema.TypeList, Elem: &schema.Schema{Type: schema.TypeString}}` | `schema.ListAttribute{ElementType: types.StringType}` |
| `schema.Schema{Type: schema.TypeSet, Elem: &schema.Resource{...}}` | `schema.SetNestedAttribute{}` or `schema.SetNestedBlock{}` (see §11) |

### Attribute flags

| SDKv2 | Plugin Framework |
|-------|-----------------|
| `Required: true` | `Required: true` (same) |
| `Optional: true` | `Optional: true` (same) |
| `Computed: true` | `Computed: true` (same) |
| `Sensitive: true` | `Sensitive: true` (same) |
| `ForceNew: true` | `PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}` |
| `Default: "value"` | `Default: stringdefault.StaticString("value")` |
| `ValidateFunc: validateDuration` | `Validators: []validator.String{validators.Duration()}` |
| `DiffSuppressFunc: suppress` | `PlanModifiers: []planmodifier.String{customModifier}` |
| `StateFunc: hashFunc` | `PlanModifiers: []planmodifier.String{planmodifiers.SensitiveHash()}` |

### Computed-only ID field (universal pattern)

Every resource defines its `id` attribute the same way:
```go
"id": schema.StringAttribute{
    Computed: true,
    Description: "...",
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

`UseStateForUnknown()` tells the Framework: "if we already have a value for this in
state, keep it in the plan rather than showing `(known after apply)`". Without this,
every `terraform plan` would show `id = (known after apply)` even when the ID is known.

---

## 6. CRUD Operations

### Method signatures

**SDKv2:**
```go
func resourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics
func resourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics
func resourceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics
func resourceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics
```

**Plugin Framework:**
```go
func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse)
func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse)
func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse)
func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse)
```

Changes:
- Methods are on a struct receiver, not standalone functions — client is stored in
  `r.client` (set in `Configure`) instead of cast from `meta`
- Request/response objects replace the single `*schema.ResourceData`
- No return value — errors are written to `resp.Diagnostics`

### Reading plan/state

**SDKv2:**
```go
name := d.Get("name").(string)
disabled := d.Get("disabled").(bool)
```

**Plugin Framework:**
```go
var plan TagResourceModel
resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
if resp.Diagnostics.HasError() {
    return
}
name := plan.Name.ValueString()
disabled := plan.Disabled.ValueBool()
```

The key difference: the Framework deserializes the entire config/plan/state into a typed
struct in one call. Accessing individual fields never panics from a bad type assertion.

### Writing state

**SDKv2:**
```go
d.Set("name", tag.Name)
d.Set("disabled", tag.Status == "Inactive")
d.SetId(tag.ID)
```

**Plugin Framework:**
```go
plan.Name = types.StringValue(tag.Name)
plan.Disabled = types.BoolValue(tag.Status == "Inactive")
plan.ID = types.StringValue(tag.ID)
resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
```

### Handling "not found" on Read

**SDKv2:**
```go
d.SetId("")  // removing the ID signals the resource is gone
return nil
```

**Plugin Framework:**
```go
resp.State.RemoveResource(ctx)  // explicit, intent-revealing method
return
```

### Client injection (Configure method)

Every resource implements `resource.ResourceWithConfigure` to receive the API client:
```go
func (r *TagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return  // called before provider Configure; skip
    }
    client, ok := req.ProviderData.(*britive.Client)
    if !ok {
        resp.Diagnostics.AddError("Unexpected Resource Configure Type", ...)
        return
    }
    r.client = client
}
```

SDKv2 used `meta.(providerMeta).client` inside each CRUD function. The Framework
pattern stores the client once on the struct and all CRUD methods use `r.client`.

---

## 7. State Management

### types.String / types.Bool / types.Int64

All state fields use Framework wrapper types instead of raw Go types:

| Go type | Framework type | Zero / null value |
|---------|---------------|-------------------|
| `string` | `types.String` | `types.StringNull()` |
| `bool` | `types.Bool` | `types.BoolNull()` |
| `int64` | `types.Int64` | `types.Int64Null()` |

These types distinguish between three states that a plain Go type cannot:
- **Null**: attribute was not set in config (maps to HCL `null`)
- **Unknown**: value will be known after apply (maps to `(known after apply)`)
- **Known**: has an actual value

### Null vs empty string — a critical distinction

For **`Optional`-only** fields (not `Computed`): the planned value is `null` when the
user doesn't set it. The provider **must** return `null` (not `""`) or Terraform will
error with "planned value does not match config":

```go
// WRONG for Optional-only field:
state.Description = types.StringValue("")

// CORRECT:
state.Description = types.StringNull()
```

For **`Optional+Computed`** fields: the provider can return either null or a value, but
must not return unknown. If the API doesn't return the field, set it to null:

```go
// After Create, if the API didn't populate app_name:
if plan.AppName.IsUnknown() {
    plan.AppName = types.StringNull()
}
```

### UseStateForUnknown and the null-state trap

`stringplanmodifier.UseStateForUnknown()` preserves the prior state value during
planning instead of showing `(known after apply)`. However, it **skips null state
values** — if the state is null, the plan remains unknown.

This affected `entity_root_environment_group_id` in `britive_application`. For app
types where this field doesn't apply, storing `types.StringNull()` meant every
subsequent plan showed the field as unknown. The fix was to store `types.StringValue("")`
(empty string, not null) so `UseStateForUnknown` can copy the known value into the plan.

---

## 8. Import State

**SDKv2:**
```go
Importer: &schema.ResourceImporter{
    StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
        // parse ID, set fields, return
    },
},
```

**Plugin Framework:**
```go
// Interface declaration
var _ resource.ResourceWithImportState = &TagResource{}

// Method implementation
func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Parse req.ID using regex
    // Look up entity by name via API
    // Set individual attributes
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), tag.ID)...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), tag.Name)...)
}
```

The import pattern used across all resources:
1. Parse `req.ID` with one or more named-capture-group regexes (supporting multiple
   import ID formats like `tags/{name}` and `{name}`)
2. Validate parsed components (non-empty, whitespace checks)
3. Resolve names to IDs via API calls
4. Set the minimum state needed for `Read` to populate the rest

Example: `britive_tag` supports both `tags/{name}` and bare `{name}` import IDs:
```go
idRegexes := []string{
    `^tags/(?P<name>[^/]+)$`,
    `^(?P<name>[^/]+)$`,
}
```

---

## 9. Custom Validators

**Location:** `britive/validators/`

SDKv2 used `ValidateFunc func(interface{}, string) ([]string, []string)` — an untyped
function returning (warnings, errors) string slices. The Framework defines validators
as structs implementing `validator.String` (or `validator.Set`, `validator.Int64` etc.).

### Three validators written for this provider

**`validators.Duration()`** — [validators/duration.go](britive/validators/duration.go)

Validates that a string can be parsed by `time.ParseDuration`:
```go
_, err := time.ParseDuration(value)
// Reports error if parsing fails
```
Used on `expiration_duration`, `extension_duration`, `notification_prior_to_expiration`
in `britive_profile`.

**`validators.Alphanumeric()`** — [validators/alphanumeric.go](britive/validators/alphanumeric.go)

Validates `[a-zA-Z0-9_-]` character set. Used for resource type names and identifiers.

**`validators.SVG()`** — [validators/svg.go](britive/validators/svg.go)

Validates that a string is valid SVG XML and does not exceed 400KB. Used for
application icon properties.

### Validator interface

```go
type durationValidator struct{}

func (v durationValidator) Description(_ context.Context) string { ... }
func (v durationValidator) MarkdownDescription(_ context.Context) string { ... }
func (v durationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
        return  // validators must skip null/unknown — Framework contract
    }
    // ... validation logic, write to resp.Diagnostics on failure
}

func Duration() validator.String { return durationValidator{} }
```

The `Description` and `MarkdownDescription` methods provide documentation that tools
like `terraform providers schema` and IDE plugins can surface to users.

### Case-insensitive enum validation

For `application_type` (tests pass values like `"gcp wif"`, `"okta"`, `"AWS"`) and
`code_language` (tests use `"PyThon"`), `stringvalidator.OneOfCaseInsensitive` is used
instead of `stringvalidator.OneOf`:

```go
stringvalidator.OneOfCaseInsensitive("Snowflake", "GCP", "GCP WIF", "AWS", "Azure", "Okta", ...)
```

---

## 10. Plan Modifiers

**Location:** `britive/planmodifiers/sensitive_hash.go`

### Built-in plan modifiers

Two built-in modifiers are used extensively:

```go
// Preserve prior state value (avoids "(known after apply)" on stable computed fields)
stringplanmodifier.UseStateForUnknown()

// Destroy and recreate if field changes (replaces ForceNew: true)
stringplanmodifier.RequiresReplace()
```

`RequiresReplace` is applied to immutable fields like `tag_id`, `username` in
`britive_tag_member`, `profile_id` in `britive_profile_session_attribute`, and
`app_container_id` in `britive_profile`.

### Custom argon2 hash modifier

SDKv2 had `StateFunc` — a function applied to a value before storing in state.
The provider used it to store an argon2 hash of sensitive property values instead of
the plaintext, preventing perpetual diffs when Terraform plans re-read plaintext from
config and compares with the stored plaintext.

The Framework has no direct `StateFunc` equivalent. The replacement is a custom
`planmodifier.String`:

```go
// britive/planmodifiers/sensitive_hash.go

func (m SensitiveHashModifier) PlanModifyString(ctx context.Context,
    req planmodifier.StringRequest, resp *planmodifier.StringResponse) {

    planValue := req.PlanValue.ValueString()

    if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
        // First apply: hash the plaintext and store the hash
        resp.PlanValue = types.StringValue(getHash(planValue))
        return
    }

    // Subsequent plans: compare hash of config value with stored hash
    if req.StateValue.ValueString() == getHash(planValue) {
        resp.PlanValue = req.StateValue  // no change, keep state
    } else {
        resp.PlanValue = types.StringValue(getHash(planValue))  // changed, store new hash
    }
}

func getHash(val string) string {
    hash := argon2.IDKey([]byte(val), []byte{}, 1, 64*1024, 4, 32)
    return base64.RawStdEncoding.EncodeToString(hash)
}
```

Applied in `britive_application` to all `sensitive_properties.value` fields:
```go
"value": schema.StringAttribute{
    Required:  true,
    Sensitive: true,
    PlanModifiers: []planmodifier.String{
        planmodifiers.SensitiveHash(),
    },
},
```

The modifier is also used in Read to detect whether a state value is a hash:
```go
// In populateStateFromAPI, for sensitive properties:
for _, prop := range app.SensitiveProperties {
    for j, stateProp := range state.SensitiveProperties {
        if stateProp.Name.ValueString() == prop.Name {
            // If state has a hash of the current value, keep the hash (no diff)
            if planmodifiers.IsHashValue(stateProp.Value.ValueString(), prop.Value) {
                state.SensitiveProperties[j].Value = stateProp.Value
            } else {
                state.SensitiveProperties[j].Value = types.StringValue(prop.Value)
            }
        }
    }
}
```

---

## 11. Nested Blocks vs Nested Attributes

This is the most important structural decision in the Framework migration and the source
of a class of bugs if chosen incorrectly.

### The distinction

**`SetNestedAttribute`** — used with `=` assignment syntax in HCL:
```hcl
resource "britive_policy" "example" {
  members = [{ type = "User", value = "alice@example.com" }]
}
```

**`SetNestedBlock`** — used with block syntax in HCL (no `=`):
```hcl
resource "britive_profile" "example" {
  associations {
    type  = "EnvironmentGroup"
    value = "Root"
  }
}
```

### The Go model difference

For `SetNestedBlock`, the struct field **must be a slice**, not `types.Set`:
```go
// CORRECT for SetNestedBlock
Associations []ProfileAssociationModel `tfsdk:"associations"`

// WRONG — causes type mismatch
Associations types.Set `tfsdk:"associations"`
```

For `SetNestedAttribute`, the struct field **must be `types.Set`** with an appropriate
object type.

### Schema placement

`SetNestedBlock` goes in the `Blocks:` map, not `Attributes:`:
```go
resp.Schema = schema.Schema{
    Attributes: map[string]schema.Attribute{
        "id": schema.StringAttribute{...},
    },
    Blocks: map[string]schema.Block{           // ← separate map
        "associations": schema.SetNestedBlock{
            NestedObject: schema.NestedBlockObject{
                Attributes: map[string]schema.Attribute{...},
            },
        },
    },
}
```

### Fields using SetNestedBlock in this provider

All of these use HCL block syntax and require the slice model:

| Resource | Block field | Notes |
|----------|-------------|-------|
| `britive_profile` | `associations` | `[]ProfileAssociationModel` |
| `britive_application` | `properties`, `sensitive_properties`, `user_account_mappings` | `[]PropertyModel`, `[]UserAccountMappingModel` |
| `britive_profile_policy` | `policy_priority` | `[]PolicyPriorityModel` |
| `britive_resource_manager_resource_type` | `variables` | `[]VariableModel` |
| `britive_resource_manager_resource` | `resource_labels` | `[]ResourceLabelModel` |
| `britive_resource_manager_resource_policy` | `resource_labels` | `[]ResourceLabelModel` |
| `britive_resource_manager_resource_type_permissions` | `values`, `parameters` | `[]ValueModel`, `[]ParameterModel` |

### Reading slice blocks

```go
// Read all associations from plan
for _, assoc := range plan.Associations {
    apiAssoc.Type = assoc.Type.ValueString()
    apiAssoc.Value = assoc.Value.ValueString()
    // ...
}
```

No `ElementsAs` call needed — the Framework populates the slice directly.

### Block validators

Block-level validators use `setvalidator` (a separate package):
```go
import "github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"

"associations": schema.SetNestedBlock{
    Validators: []validator.Set{
        setvalidator.SizeAtLeast(1),  // at least one association required
    },
    ...
}
```

---

## 12. Sensitive Properties — Argon2 Hashing

See §10 for the plan modifier implementation. The full flow for sensitive properties:

1. **Config** contains plaintext: `value = "<Private Key>"`
2. **PlanModifyString** runs during planning:
   - First apply: computes `argon2(plaintext)` → stored as plan value
   - Subsequent plans: compares `argon2(config_plaintext)` with `state_hash`
     - Match → no diff (no change shown in plan)
     - No match → new hash in plan (change shown)
3. **State** always stores the hash, never the plaintext
4. **Read** uses `IsHashValue(stateHash, apiValue)` to detect round-trips without diff

The argon2 parameters match what the SDKv2 StateFunc used, ensuring state files from
v2.x remain valid without reprovisioning sensitive properties.

---

## 13. Default Values

**SDKv2:**
```go
"disabled": {
    Type:    schema.TypeBool,
    Default: false,
}
```

**Plugin Framework:**
```go
"disabled": schema.BoolAttribute{
    Optional:  true,
    Computed:  true,
    Default:   booldefault.StaticBool(false),
}
```

`Default` requires both `Optional: true` and `Computed: true`. The `Computed` flag
signals that the provider can set this field; `Optional` allows the user to override it.
Without `Computed`, the Framework will reject the `Default` at schema registration.

Default packages used:
```go
import (
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
)

booldefault.StaticBool(false)
stringdefault.StaticString("resourcemanager")
int64default.StaticInt64(0)
```

---

## 14. Config-Level Validation

The Framework adds a `ValidateConfig` hook that runs before any plan/apply. This
replaces ad-hoc validation inside `Create`/`Update` in SDKv2.

Resources implementing `resource.ResourceWithValidateConfig`:
- `britive_application`
- `britive_profile`
- `britive_profile_session_attribute`
- `britive_resource_manager_resource_type_permissions`

Example — `britive_profile` validates cross-field constraints:
```go
func (r *ProfileResource) ValidateConfig(ctx context.Context,
    req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {

    var data ProfileResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

    // When extendable=true, notification_prior_to_expiration and extension_duration
    // are required — enforce this at config time, not at apply time
    if data.Extendable.ValueBool() {
        if data.NotificationPriorToExpiration.IsNull() {
            resp.Diagnostics.AddAttributeError(
                path.Root("notification_prior_to_expiration"),
                "Missing Required Field",
                "When extendable is true, notification_prior_to_expiration must be provided",
            )
        }
    }

    // ApplicationResource associations require parent_name
    for i, assoc := range data.Associations {
        if assoc.Type.ValueString() == "ApplicationResource" {
            if assoc.ParentName.IsNull() {
                resp.Diagnostics.AddAttributeError(
                    path.Root("associations"),
                    "Missing Required Field",
                    fmt.Sprintf("parent_name is required for ApplicationResource associations (index %d)", i),
                )
            }
        }
    }
}
```

The `path.Root("field")` argument pins the error to the specific attribute in the user's
HCL, giving precise error location in `terraform validate` output.

---

## 15. Test Infrastructure

### Provider factory

**SDKv2 tests:**
```go
resource.Test(t, resource.TestCase{
    Providers:    testAccProviders,   // map[string]*schema.Provider
    PreCheck:     func() { testAccPreCheck(t) },
    ...
})
```

**Framework tests:**
```go
resource.Test(t, resource.TestCase{
    ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
    PreCheck:                 func() { testAccPreCheckFramework(t) },
    ...
})
```

The factory definition:
```go
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "britive": providerserver.NewProtocol6WithError(britive.New(testVersion)()),
}
```

`providerserver.NewProtocol6WithError` wraps a `provider.Provider` in the Protocol v6
gRPC server that the test framework connects to. This is the Framework equivalent of
the SDKv2 provider server.

### Import changes

All test files changed their import from:
```go
// SDKv2
"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
```
to:
```go
// Framework (uses terraform-plugin-testing, not the SDK)
"github.com/hashicorp/terraform-plugin-testing/helper/resource"
"github.com/hashicorp/terraform-plugin-testing/terraform"
```

`terraform-plugin-testing` is a standalone testing package that works with both SDKv2
and Framework providers and is independent of either implementation.

### PreCheck function

```go
func testAccPreCheckFramework(t *testing.T) {
    configPath, _ := homedir.Expand("~/.britive/tf.config")
    if _, err := os.Stat(configPath); !os.IsNotExist(err) {
        return  // config file present, tests can proceed
    }
    // If no config file, env vars must be set
    if tenant := os.Getenv("BRITIVE_TENANT"); tenant == "" {
        t.Fatal("BRITIVE_TENANT must be set for acceptance tests")
    }
    if token := os.Getenv("BRITIVE_TOKEN"); token == "" {
        t.Fatal("BRITIVE_TOKEN must be set for acceptance tests")
    }
}
```

---

## 16. Lifecycle & Idempotency Tests

**File:** `britive/tests/resource_lifecycle_test.go`

These tests were added alongside the Plugin Framework migration to systematically verify
the correctness of every resource's in-place update behaviour and state round-trip
stability. They are separate from the per-resource functional tests (which cover basic
CRUD and import) and focus exclusively on drift detection.

### Three test patterns

#### Pattern 1 — Edit tests

Verifies that fields which should update in-place do so without triggering an accidental
`RequiresReplace` (destroy + recreate). Two steps:

1. Create the resource with an initial config value.
2. Change the value → assert the plan shows **Update** (not Replace) → apply → assert
   the plan is empty after a refresh.

```go
// Step 2 of an edit test:
{
    Config: testAccCheckBritivePermissionConfig(name, updatedDescription),
    ConfigPlanChecks: resource.ConfigPlanChecks{
        PreApply: []plancheck.PlanCheck{
            // Must be an in-place Update, NOT a Replace (destroy+recreate)
            plancheck.ExpectResourceAction("britive_permission.new", plancheck.ResourceActionUpdate),
        },
        PostApplyPostRefresh: []plancheck.PlanCheck{
            // After the API call + a fresh Read, the plan must be empty
            plancheck.ExpectEmptyPlan(),
        },
    },
},
```

`plancheck.ExpectResourceAction` is the key assertion — it fails the test if the plan
proposes a `Create`/`Delete` pair instead of an `Update`, catching any field that was
accidentally decorated with `RequiresReplace()`.

#### Pattern 2 — Idempotency tests

Verifies that applying a resource once and immediately re-planning (with a fresh `Read`
from the API) produces an empty plan. This catches:

- API responses that return values in a different format than what Terraform wrote
  (e.g. date normalization, case changes, ordering differences in JSON fields)
- `Optional+Computed` fields that flip to `(known after apply)` after the first apply
- Computed fields whose value changes on every API call

```go
{
    Config: testAccCheckBritivePermissionConfig(name, description),
    ConfigPlanChecks: resource.ConfigPlanChecks{
        PostApplyPostRefresh: []plancheck.PlanCheck{
            // After apply + Read, the plan must be empty (no perpetual drift)
            plancheck.ExpectEmptyPlan(),
        },
    },
},
```

`PostApplyPostRefresh` runs after `terraform apply` **and** a subsequent `terraform
refresh`, making it the strongest possible no-drift assertion — it catches drift that
only manifests after the API normalises a value on its next read.

#### Pattern 3 — Provider migration tests

Verifies that state written by v2.2.8 (SDKv2, Protocol v5) is compatible with v3.0.0
(Plugin Framework, Protocol v6). Step 1 creates a resource using the v2.2.8 binary
downloaded from the Terraform registry; step 2 runs `terraform plan` using the locally
built v3.0.0 binary and asserts the plan is empty.

Terraform's `UpgradeResourceState` RPC (called automatically when the schema version
changes) performs the flatmap → JSON state migration transparently. The test proves this
is invisible to users — no resource replacement or unexpected updates appear.

```go
Steps: []resource.TestStep{
    // Step 1: Create with v2.2.8 from the registry (SDKv2)
    {
        ExternalProviders: map[string]resource.ExternalProvider{
            "britive": {Source: "britive/britive", VersionConstraint: "= 2.2.8"},
        },
        Config: config,
    },
    // Step 2: Plan with local v3.0.0 (Plugin Framework) — must show no changes
    {
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Config:   config,
        PlanOnly: true,
    },
},
```

> **Note:** `britive_profile` is intentionally excluded from Pattern 3. The
> `associations` field changed HCL syntax between versions (TypeSet assignment `= [...]`
> → SetNestedBlock `{}`) — the two syntaxes are mutually exclusive in a single `.tf`
> file. Users must update their `.tf` files alongside the provider upgrade; this is
> documented separately.

### Coverage

37 lifecycle tests cover all 26 resources across the three patterns:

**Pattern 1 — Edit tests (9 resources):**

| Test | Field edited |
|------|-------------|
| `TestBritiveTagEditFields` | `description`, `disabled` |
| `TestBritiveProfileEditFields` | `description`, `expiration_duration` |
| `TestBritivePermissionEditFields` | `description` |
| `TestBritiveRoleEditFields` | `description` |
| `TestBritiveProfilePolicyEditFields` | `description` |
| `TestBritiveProfileAdditionalSettingsEditFields` | `console_access`, `programmatic_access` |
| `TestBritiveProfilePolicyPrioritizationEditFields` | priority order (swap) |
| `TestBritiveResourceTypeEditFields` | `description` |
| `TestBritiveResponseTemplateEditFields` | `description` |
| `TestBritiveResourceLabelEditFields` | `description` |
| `TestBritiveResourceManagerProfileEditFields` | `description` |

**Pattern 2 — Idempotency tests (all 26 resources + application):**

All resources have a corresponding `TestBritive<Resource>Idempotency` test that uses
`PostApplyPostRefresh: [plancheck.ExpectEmptyPlan()]`.

**Pattern 3 — Migration tests (1 resource):**

`TestBritiveTagProviderMigration` — the simplest resource, used to prove the
SDKv2 → Framework state upgrade path works end to end.

### Defensive Create rollback

During testing, an idempotency failure on `britive_profile` revealed an orphan resource
bug: if `Create` succeeded in creating the profile but subsequently failed to save
associations, Terraform had no state ID and could not clean up the partially created
resource. Subsequent test runs then failed with "profile name already exists".

The fix adds a rollback in `profile_resource.go`:
```go
err = r.saveProfileAssociations(p.AppContainerID, p.ProfileID, &plan)
if err != nil {
    // Delete the profile we just created so the name is freed for retry
    if deleteErr := r.client.DeleteProfile(p.AppContainerID, p.ProfileID); deleteErr != nil {
        log.Printf("[WARN] Failed to delete profile %s after association error: %s", p.ProfileID, deleteErr)
    }
    resp.Diagnostics.AddError("Error Saving Profile Associations", err.Error())
    return
}
```

---

## 17. Continuous Integration

**File:** `.github/workflows/acceptance-tests.yml`

The CI workflow runs the full acceptance test suite (`make testacc`) on every push to
`main` and on every pull request targeting `main`.

### Changes from the original workflow

| Setting | Old (SDKv2 era) | New (Plugin Framework) | Why |
|---------|----------------|------------------------|-----|
| `actions/checkout` | `@v2` | `@v4` | v2 is deprecated; newer runner environment |
| `actions/setup-go` | `@v2` + `go-version: 1.16` | `@v5` + `go-version-file: go.mod` | Plugin Framework requires Go 1.21+; `go-version-file` auto-tracks whatever `go.mod` declares (currently 1.24) |
| `hashicorp/setup-terraform` | `@v1` + `1.0.8` | `@v3` + `1.10.3` | Terraform 1.0.8 predates Protocol v6; Plugin Framework requires 1.1+ |
| `GO_FLAGS` env var | `GO_FLAGS: -mod=vendor` | `GOFLAGS: -mod=vendor` | `GO_FLAGS` is not a Go env var; `GOFLAGS` (no underscore) is the standard one that `go test` honours |
| `GO111MODULE` | `GO111MODULE: on` | *(removed)* | Redundant since Go 1.16; module mode is always on |
| `Unshallow` step | `git fetch --prune --unshallow` | *(removed)* | Only needed by release workflows that inspect git tags |
| Job `timeout-minutes` | *(none — GitHub default 6h)* | `200` | Prevents runaway jobs; the `go test -timeout 180m` inside is the inner guard |

### Test timeout

The `GNUmakefile` `testacc` target timeout was updated from `120m` to `180m` to
accommodate the 37 lifecycle tests added to the suite (total ~67 tests). The job-level
`timeout-minutes: 200` provides an outer safety net.

```yaml
# .github/workflows/acceptance-tests.yml (key sections)
jobs:
  acceptance-test:
    runs-on: ubuntu-latest
    timeout-minutes: 200
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.10.3"
          terraform_wrapper: false
      - run: make testacc
        env:
          GOFLAGS: -mod=vendor
          BRITIVE_TENANT: ${{ secrets.BRITIVE_TENANT }}
          BRITIVE_TOKEN: ${{ secrets.BRITIVE_TOKEN }}
```

### Required repository secrets

| Secret | Description |
|--------|-------------|
| `BRITIVE_TENANT` | Full URL of the Britive tenant, e.g. `https://your-org.britive-app.com/` |
| `BRITIVE_TOKEN` | API token with sufficient permissions to create/read/delete all resource types |

---

## 18. Bug Fixes Discovered During Testing

These issues were found and fixed while running the acceptance test suite against a real
Britive environment. They represent Framework-specific behavior that differs from SDKv2.

### 1. `version` perpetual drift

**Resource:** `britive_application`

**Symptom:** On a second `terraform plan` after a successful apply, `version` shows
`"1.0" -> (known after apply)` even though nothing changed.

**Root cause:** `version` is `Optional+Computed`. SDKv2 preserved computed values
automatically. The Framework treats `Optional+Computed` fields as unknown in the plan
unless a plan modifier explicitly preserves them.

**Fix:** Added `stringplanmodifier.UseStateForUnknown()` to `version`.

---

### 2. `entity_root_environment_group_id` perpetual drift

**Resource:** `britive_application`

**Symptom:** Same drift pattern — `(known after apply)` on second plan.

**Root cause:** `UseStateForUnknown` **skips null state values**. For app types where
this field doesn't apply, the state was set to `types.StringNull()` after create. On
the next plan, `UseStateForUnknown` saw null and left the plan as unknown.

**Fix:** Store `types.StringValue("")` (empty string) instead of null for non-applicable
app types. `UseStateForUnknown` can then copy the known empty string into the plan.

```go
// BEFORE — caused drift
state.EntityRootEnvironmentGroupID = types.StringNull()

// AFTER — stable
if state.EntityRootEnvironmentGroupID.IsUnknown() {
    state.EntityRootEnvironmentGroupID = types.StringValue("")
}
```

---

### 3. `attribute_value` inconsistency in profile session attributes

**Resource:** `britive_profile_session_attribute`

**Symptom:** Terraform error: `was cty.NullVal(cty.String), but now cty.StringVal("")`

**Root cause:** `attribute_value` is `Optional` (not `Computed`). When the user doesn't
set it, the planned value is `null`. The provider was returning `""` instead of `null`
after `populateStateFromAPI`.

**Fix:** Use `types.StringNull()` when clearing Optional-only fields, not
`types.StringValue("")`:
```go
// For Identity type (no attribute_value):
state.AttributeValue = types.StringNull()  // matches planned null

// For Static type (no attribute_name):
state.AttributeName = types.StringNull()
```

---

### 4. `app_name` / `profile_name` unknown after create

**Resources:** `britive_profile_session_attribute`, `britive_profile`, `britive_tag_member`

**Symptom:** `Error: Provider returned unknown value` after `terraform apply`.

**Root cause:** These `Optional+Computed` fields are not returned by the Britive API.
After `Create`, the fields are still in the unknown state set during planning.

**Fix:** After `populateStateFromAPI` in `Create`, explicitly null any still-unknown
`Optional+Computed` fields:
```go
if plan.AppName.IsUnknown() {
    plan.AppName = types.StringNull()
}
if plan.ProfileName.IsUnknown() {
    plan.ProfileName = types.StringNull()
}
```

---

### 5. `resource_labels` "Unsupported block type" error

**Resource:** `britive_resource_manager_resource_policy`

**Symptom:** `terraform apply` fails with `Unsupported block type: "resource_labels"`.

**Root cause:** The original migration used `SetNestedAttribute` (requires `=` syntax)
but the HCL config uses block syntax `resource_labels { }`. The Framework enforces this
distinction strictly.

**Fix:** Changed to `SetNestedBlock` in the `Blocks:` map and changed the Go model
field from `types.Set` to `[]ResourceLabelModel`.

---

### 6. `resource_labels` value order drift

**Resource:** `britive_resource_manager_resource`

**Symptom:** Perpetual diff: `label_values = "Development,Production"` → `"Production,Development"`.

**Root cause:** Label values are stored as comma-separated strings. The Britive API
returns values in a different order than the user configured them. Sorting alphabetically
produced `"Development,Production"`, but the user's config had `"Production,Development"`.

**Fix:** Preserve the state's ordering if the API returns the same set of values
(regardless of order):
```go
func sameValueSet(apiValues []string, stateStr string) bool {
    stateValues := strings.Split(stateStr, ",")
    if len(apiValues) != len(stateValues) {
        return false
    }
    apiSorted := make([]string, len(apiValues))
    copy(apiSorted, apiValues)
    sort.Strings(apiSorted)
    stateSorted := make([]string, len(stateValues))
    copy(stateSorted, stateValues)
    sort.Strings(stateSorted)
    return strings.Join(apiSorted, ",") == strings.Join(stateSorted, ",")
}
```

---

### 7. `resource_type_id` unknown on update plans

**Resource:** `britive_resource_manager_resource`

**Symptom:** `resource_type_id = (known after apply)` on every plan after the initial
apply.

**Root cause:** `resource_type_id` is `Computed`-only with no plan modifier. During
update plans the Framework treats it as unknown.

**Fix:** Added `UseStateForUnknown()`.

---

### 8. `code_language` validator case sensitivity

**Resource:** `britive_resource_manager_resource_type`

**Symptom:** Acceptance test failed with `expected one of ["Python", ...]` because the
test config uses `"PyThon"`.

**Fix:** Changed `stringvalidator.OneOf(...)` to `stringvalidator.OneOfCaseInsensitive(...)`.

---

## 19. Quick-Reference Pattern Map

| Task | SDKv2 | Plugin Framework |
|------|-------|-----------------|
| Read a string from config | `d.Get("name").(string)` | `plan.Name.ValueString()` |
| Read a bool from config | `d.Get("disabled").(bool)` | `plan.Disabled.ValueBool()` |
| Write a string to state | `d.Set("name", val)` | `state.Name = types.StringValue(val)` |
| Write a bool to state | `d.Set("disabled", val)` | `state.Disabled = types.BoolValue(val)` |
| Set the resource ID | `d.SetId(id)` | `plan.ID = types.StringValue(id)` |
| Signal resource gone | `d.SetId("")` | `resp.State.RemoveResource(ctx)` |
| Add an error | `diag.FromErr(err)` | `resp.Diagnostics.AddError("title", err.Error())` |
| Check for errors | `diags.HasError()` | `resp.Diagnostics.HasError()` |
| ForceNew field | `ForceNew: true` | `stringplanmodifier.RequiresReplace()` |
| Default value | `Default: false` | `Default: booldefault.StaticBool(false)` |
| Validate a string | `ValidateFunc: validateFn` | `Validators: []validator.String{...}` |
| Suppress a diff | `DiffSuppressFunc: fn` | Custom `planmodifier.String` |
| Hash sensitive value | `StateFunc: hashFn` | `planmodifiers.SensitiveHash()` |
| Stable computed field | *(automatic in SDKv2)* | `UseStateForUnknown()` |
| Nested set of objects | `TypeSet` + `Elem: &schema.Resource{}` | `SetNestedBlock{}` in `Blocks:` |
| Import by name | `StateContext` function | `ImportState` method with regex parsing |
| Test provider factory | `Providers: testAccProviders` | `ProtoV6ProviderFactories: testAccProtoV6ProviderFactories` |
| Check for null | *(no direct equivalent)* | `value.IsNull()` |
| Check for unknown | *(no direct equivalent)* | `value.IsUnknown()` |

---

## 20. Rollback Strategy

This section describes how to roll back from v3.0.0 to v2.x if a critical issue is
discovered after customers have upgraded.

### Why Rollback is Viable Here

The v2.x → v3.0.0 migration is a **provider implementation change only** — the
underlying Britive API, resource names, and attribute names are all unchanged. The
Terraform state file stores the same attribute values in both versions, which means
a backed-up state file can be cleanly restored and used with v2.x.

### Step 0: Back Up the State File (Before Upgrading)

Customers **must** take a state backup before upgrading to v3.0.0. This is the
critical prerequisite for any rollback.

**Local state:**
```bash
cp terraform.tfstate terraform.tfstate.pre-v3-backup
```

**Remote state (S3, GCS, Azure Blob, etc.):**
```bash
terraform state pull > terraform.tfstate.pre-v3-backup
```

**Terraform Cloud / Terraform Enterprise:**
State history is kept automatically. You can restore a previous state version from
the workspace UI or via the API. No manual backup is required.

---

### Rollback Runbook (for customers)

Use this procedure if you need to revert to v2.x after applying v3.0.0.

#### 1. Pin the provider back to v2.x

In your `required_providers` block:

```hcl
terraform {
  required_providers {
    britive = {
      source  = "britive/britive"
      version = "~> 2.2"
    }
  }
}
```

#### 2. Restore the state backup

**Local state:**
```bash
cp terraform.tfstate.pre-v3-backup terraform.tfstate
```

**Remote state:**
```bash
terraform state push terraform.tfstate.pre-v3-backup
```

#### 3. Re-initialise with the v2.x provider

```bash
terraform init -upgrade
```

#### 4. Verify no unexpected changes

```bash
terraform plan
```

The plan should be empty (no changes). If it shows unexpected changes, compare
the backup state with your actual infrastructure and reconcile manually before
applying.

---

### Provider-Side Rollback Actions

| Scenario | Action |
|----------|--------|
| Bug found, quick fix available | Publish v3.0.1 patch release (preferred) |
| Critical regression, no quick fix | Publish v2.2.x hotfix; advise customers to pin `~> 2.2` |
| v3.0.0 must be yanked | Mark release as `revoked` in the Terraform Registry; it stops new downloads but does not break existing users |
| Widespread data-loss risk | Open a GitHub issue immediately; notify via release notes and Registry advisory |

> **Recommended approach:** Always fix forward with a v3.0.1 patch rather than asking
> customers to roll back. Rolling back requires manual state manipulation and is
> error-prone. Reserve the rollback runbook for cases where the v3.0.x line cannot
> be stabilised quickly.

---

### Why This Migration Is Low-Risk to Roll Back

| Factor | Detail |
|--------|--------|
| Same attribute names | No HCL rewrites required |
| Same resource IDs | API-assigned IDs are preserved in state |
| Same API client | `britive-client-go` is used unchanged in both versions |
| No `schema_version` bumps | No state migration functions were added, so Terraform does not record an incremented schema version that would prevent downgrade |
| Wire-compatible protocols | Protocol v5 (SDKv2) and Protocol v6 (Framework) both read the same state attribute values |

---

## 21. Post-Migration Changes (v3.0.0+)

This section tracks feature additions and resource migrations added **after** the
initial SDKv2 → Framework migration, incorporated directly into the Framework
codebase (commit `ca2dc23`, 2026-03-23).

---

### 21.1 `britive_resource_manager_profile` — Extensible Session Fields

**Source commit:** `ca2dc23`

Four new optional attributes were added to support configurable session extension:

| Attribute | Type | Notes |
|-----------|------|-------|
| `extendable` | `bool` | Default `false`. When `true`, the three fields below are required. |
| `notification_prior_to_expiration` | `string` | Go duration string (e.g. `"1h0m0s"`). Validated by `validators.Duration()`. |
| `extension_duration` | `string` | Go duration string. Validated by `validators.Duration()`. |
| `extension_limit` | `int64` | Maximum number of extensions allowed. |

**Duration conversion:** The API stores these values in milliseconds (`int64`). The
Framework resource converts to/from Go `time.Duration` strings on read and write.

**Conditional validation:** `mapResourceToModel` returns an error if `extendable = true`
and either `notification_prior_to_expiration` or `extension_duration` is empty.

**Null handling:** When `extendable = false` and a field is unknown (e.g. first apply
before the API responds), the field is set to `types.StringNull()` /
`types.Int64Null()` to avoid "provider returned unknown value" errors.

**Files changed:**
- [britive/resources/resourcemanager/profile_resource.go](britive/resources/resourcemanager/profile_resource.go) — model, schema, `mapResourceToModel`, `mapModelToResource`

**Tests added:**
- `TestBritiveResourceManagerProfileExtensibleSessionIdempotency` — verifies no drift after apply with all extension fields set
- `TestBritiveResourceManagerProfileExtensibleSessionEditFields` — verifies `extension_limit` can be changed in-place (no Replace)

---

### 21.2 New Resource: `britive_resource_manager_profile_policy_prioritization`

**Source commit:** `ca2dc23`

This resource controls the priority ordering of policies within a resource manager
profile. It is the resource-manager counterpart to the existing
`britive_profile_policy_prioritization` resource.

**Schema:**

| Attribute/Block | Type | Notes |
|-----------------|------|-------|
| `profile_id` | `string` (required) | The resource manager profile ID. |
| `policy_priority_enabled` | `bool` (optional, computed) | Default `true`. Config-level validation rejects `false`. |
| `policy_priority` (block) | `SetNestedBlock` | Each block has `id` (string) and `priority` (int64, 0-based). |

**Ordering logic:** User specifies explicit priorities for a subset of policies.
Remaining policies are filled into gaps in their existing API order. This matches
the SDKv2 implementation exactly.

**ID format:** `resource-manager/{profileID}/policies/priority`

**Import formats:**
- `resource-manager/{profileID}/policies/priority`
- `{profileID}` (bare ID)

**Delete behaviour:** Disables policy ordering (`policyOrderingEnabled = false`) on the
profile. Does not delete any policies.

**Framework patterns used:**
- `SetNestedBlock` for `policy_priority` (HCL block syntax)
- Go model uses `[]RMPolicyPriorityModel` (not `types.Set`) — required for `SetNestedBlock`
- `resource.ResourceWithValidateConfig` for the `policy_priority_enabled = false` guard
- `stringplanmodifier.UseStateForUnknown()` on `id`

**Files changed:**
- [britive/resources/resourcemanager/profile_policy_prioritization_resource.go](britive/resources/resourcemanager/profile_policy_prioritization_resource.go) — new file
- [britive/provider_framework.go](britive/provider_framework.go) — registered `resourcemanager.NewRMProfilePolicyPrioritizationResource`

**Tests added / migrated:**
- [britive/tests/resource_resource_manager_profile_policy_prioritization_test.go](britive/tests/resource_resource_manager_profile_policy_prioritization_test.go) — migrated from SDKv2 (`testAccProviders`, `testAccPreCheck`) to Framework (`ProtoV6ProviderFactories`, `testAccPreCheckFramework`)
- `TestBritiveResourceManagerProfilePolicyPrioritizationIdempotency` — verifies no drift after apply
