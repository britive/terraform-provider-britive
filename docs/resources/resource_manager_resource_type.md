---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource_type Resource - britive"
description: |-
  Manages resource type for the Britive provider.
---

# britive_resource_manager_resource_type Resource

The `britive_resource_manager_resource_type` resource allows you to manage resource types in Britive.

## Example Usage

```hcl
resource "britive_resource_manager_resource_type" "example" {
  name        = "example-resource-type"
  description = "An example resource type"
  icon = file("resource_type.svg")
  parameters {
    param_name   = "username"
    param_type   = "string"
    is_mandatory = true
  }
  parameters {
    param_name   = "password"
    param_type   = "password"
    is_mandatory = true
  }
  parameters {
    param_name   = "ip"
    param_type   = "ip-cidr"
    is_mandatory = true
  }
  parameters {
    param_name   = "regex"
    param_type   = "regex-pattern"
    is_mandatory = true
  }
  parameters {
    param_name   = "list"
    param_type   = "list"
    is_mandatory = true
  }
}
```

## Argument Reference

* `name` - (Required) The name of the Britive resource type. Only letters, numbers, hyphens (`-`), and underscores (`_`) are allowed, no other special characters. Used to uniquely identify the resource type within Britive.
* `description` - (Optional) The description of the Britive resource type.
* `icon` - (Required) The icon of the Britive resource type
* `scan_enabled` - (Optional) Whether scheduled scanning is enabled for this resource type. Left **unmanaged** when omitted from config - the provider only ever calls the enable/disable API when this argument is explicitly present, so resource types that predate this argument (with or without scanning already turned on some other way) are left exactly as they are. Removing this argument from config after previously setting it disables scanning. The underlying scan task service is registered automatically when the resource type is created (and removed automatically when it's deleted), so this can be set to `true` in the very same apply that creates the resource type and any of its [`britive_resource_manager_resource_type_schedule_scan`](resource_manager_resource_type_schedule_scan.md) - see `scan_enabled` Semantics below.
* `rotation_enabled` - (Optional) Whether rotation is enabled for this resource type's rotation templates. Left **unmanaged** when omitted from config, with the same semantics as `scan_enabled` (see below) - the provider only ever sends this field when it's explicitly present in config, so resource types that predate this argument are left exactly as they are. Removing this argument from config after previously setting it disables rotation. Independent of any [`britive_resource_manager_resource_type_rotation_template`](resource_manager_resource_type_rotation_template.md) existing, so unlike `scan_enabled` it can always be set in the same apply that creates the resource type.
* `parameters` - (Optional) A set of parameters/fields for the resource type. Each parameter supports the following attributes:
  * `param_name` - (Required) The name of the parameter. Only letters, numbers, hyphens (`-`), and underscores (`_`) are allowed, no other special characters.
  * `param_type` - (Required) The type of the parameter. Must be one of [`string`, `password`, `ip-cidr`, `regex-pattern`, `list`] (case-insensitive).
  * `is_mandatory` - (Required) A boolean indicating whether the parameter is mandatory.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The unique identifier of the resource type.
* `task_service_id` - The ID of this resource type's scan task service. Registered automatically when the resource type is created and removed automatically when it's deleted - not independently managed by this provider. Used internally by `scan_enabled` and by [`britive_resource_manager_resource_type_schedule_scan`](resource_manager_resource_type_schedule_scan.md).

## Import

Resource types can be imported using their ID:

```sh
terraform import britive_resource_manager_resource_type.example resource-manager/resource-types/<resource_type_id>
```

## `scan_enabled` / `rotation_enabled` Semantics

`scan_enabled` and `rotation_enabled` are unlike this provider's other attributes in one
respect: **omitting either from config is not the same as setting it to `false`**. Each is
only ever acted on when present in config at all:

* Omitted entirely - the provider never touches it. The exported value simply reflects
  whatever the resource type's actual status already is (`false` for a resource type that
  has never used the feature). This is deliberate so that adopting this provider version, or
  managing an existing resource type that already has scanning/rotation turned on some other
  way, never flips anything.
* Set to `true` or `false` - the provider actively updates it to match.
* Previously set, then removed from config - treated as an explicit request to turn it off
  (not "stop managing it and leave it as-is").

The resource type's scan task service (what `scan_enabled` actually toggles) is registered
automatically as part of resource type creation, so - unlike an earlier version of this
provider - `scan_enabled = true` can be set in the very same apply that creates both the
resource type and its first
[`britive_resource_manager_resource_type_schedule_scan`](resource_manager_resource_type_schedule_scan.md):

```hcl
resource "britive_resource_manager_resource_type" "example" {
  name         = "example-resource-type"
  description  = "An example resource type"
  icon         = file("resource_type.svg")
  scan_enabled = true
}

resource "britive_resource_manager_resource_type_schedule_scan" "example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name              = "daily-scan"
  frequency_type    = "Daily"
  start_time        = "06:30"
}
```

`rotation_enabled` has no such dependency at all - it's a plain field on the resource type
record itself, independent of whether any
[`britive_resource_manager_resource_type_rotation_template`](resource_manager_resource_type_rotation_template.md)
exists, so it can always be set at resource type creation:

```hcl
resource "britive_resource_manager_resource_type" "example" {
  name             = "example-resource-type"
  description      = "An example resource type"
  icon             = file("resource_type.svg")
  rotation_enabled = true
}
```