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
* `scan_enabled` - (Optional) Whether scheduled scanning is enabled for this resource type. Left **unmanaged** when omitted from config - the provider only ever calls the enable/disable API when this argument is explicitly present, so resource types that predate this argument (with or without scanning already turned on some other way) are left exactly as they are. Removing this argument from config after previously setting it disables scanning. The underlying scan task service is only created once at least one [`britive_resource_manager_resource_type_schedule_scan`](resource_manager_resource_type_schedule_scan.md) exists for this resource type - setting this to `true` before that fails with the API's own error, since there's nothing yet to enable. **This cannot be set to `true` in the same apply that also creates the resource type's first schedule scan** - see the note below.
* `parameters` - (Optional) A set of parameters/fields for the resource type. Each parameter supports the following attributes:
  * `param_name` - (Required) The name of the parameter. Only letters, numbers, hyphens (`-`), and underscores (`_`) are allowed, no other special characters.
  * `param_type` - (Required) The type of the parameter. Must be one of [`string`, `password`, `ip-cidr`, `regex-pattern`, `list`] (case-insensitive).
  * `is_mandatory` - (Required) A boolean indicating whether the parameter is mandatory.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The unique identifier of the resource type.

## Import

Resource types can be imported using their ID:

```sh
terraform import britive_resource_manager_resource_type.example resource-manager/resource-types/<resource_type_id>
```

## `scan_enabled` Semantics

`scan_enabled` is unlike this provider's other attributes in one respect: **omitting it from
config is not the same as setting it to `false`**. It's only ever acted on when present in
config at all:

* Omitted entirely - the provider never calls the enable/disable API. The exported value
  simply reflects whatever the resource type's actual scan status already is (`false` for a
  resource type that has never used this feature). This is deliberate so that adopting this
  provider version, or managing an existing resource type that already has scanning
  configured some other way, never flips anything.
* Set to `true` or `false` - the provider actively calls the enable/disable API to match.
* Previously set, then removed from config - treated as an explicit request to disable
  scanning (not "stop managing it and leave it as-is").

### Enabling Scanning Requires Two Applies

`scan_enabled` enables/disables the resource type's scan **task service** as a whole, which
the Britive API only creates once at least one
[`britive_resource_manager_resource_type_schedule_scan`](resource_manager_resource_type_schedule_scan.md)
exists for the resource type - there's no separate "just bootstrap the service" call. Because
of that, setting `scan_enabled = true` in the very same `terraform apply` that also creates
the resource type's first schedule scan will fail: at the point this resource is created,
the schedule scan doesn't exist yet, and there's no way to express "create the schedule scan
first" here without introducing a dependency cycle (the schedule scan already depends on this
resource type via `resource_type_id`).

Apply in two steps instead:

```hcl
# Step 1: apply with scan_enabled omitted.
resource "britive_resource_manager_resource_type" "example" {
  name        = "example-resource-type"
  description = "An example resource type"
  icon        = file("resource_type.svg")
}

resource "britive_resource_manager_resource_type_schedule_scan" "example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name              = "daily-scan"
  frequency_type    = "Daily"
  start_time        = "06:30"
}
```

```hcl
# Step 2: after the above has been applied once, add scan_enabled = true and apply again.
resource "britive_resource_manager_resource_type" "example" {
  name         = "example-resource-type"
  description  = "An example resource type"
  icon         = file("resource_type.svg")
  scan_enabled = true
}
```