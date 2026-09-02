---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource_type_scan_settings Resource - britive"
description: |-
  Manages a resource type's scan settings for the Britive provider.
---

# britive_resource_manager_resource_type_scan_settings Resource

The `britive_resource_manager_resource_type_scan_settings` resource allows you to manage
scan settings for a Britive resource manager resource type. Unlike
[`britive_resource_manager_resource_type_rotation_template`](resource_manager_resource_type_rotation_template.md),
which manages a named collection of templates, scan settings is a **singleton**: exactly
one scan settings configuration exists per resource type, with no `name`/`description` of
its own.

Scan settings has one of three modes, set via `template_type`:

* `Local` - scanning is handled by logic already deployed on the target; no script is uploaded.
* `InlineFile` - the scan script is authored inline via `script_content`/`script_language`.
* `FilePath` - the scan script is a local file uploaded via `script_file_path`.

## Example Usage

### Local

```hcl
resource "britive_resource_manager_resource_type_scan_settings" "local_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  time_limit        = 20
  template_type     = "Local"

  variables {
    name         = "username"
    type         = "String"
    multi_valued = false
  }
}
```

### Insert Code (InlineFile)

```hcl
resource "britive_resource_manager_resource_type_scan_settings" "insert_code_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  time_limit        = 20
  template_type     = "InlineFile"
  script_language   = "text"
  script_content    = <<-EOT
    Test script here
    hii
    byyy
  EOT
}
```

### Add File (FilePath)

```hcl
resource "britive_resource_manager_resource_type_scan_settings" "add_file_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  time_limit        = 20
  template_type     = "FilePath"
  script_file_path  = "${path.module}/scripts/registration-file.txt"
}
```

## Argument Reference

* `resource_type_id` - (Required) The ID of the associated resource type. Forces replacement if changed. Exactly one scan settings resource exists per resource type.
* `time_limit` - (Optional) The time limit in minutes for the scan script to run (converted to seconds on the wire). Defaults to `20`, matching the API's own default. The valid range is enforced by the API, not the provider.
* `template_type` - (Required) The scan settings mode. Must be one of `Local`, `InlineFile`, `FilePath` (case-insensitive).
* `script_file_path` - (Optional) Path to a local file to upload as the scan script. Required when `template_type = "FilePath"`; must be unset otherwise. Content-Type on upload is derived from the file's own extension (falling back to `application/octet-stream` if unrecognized).
* `script_content` - (Optional) Inline scan script content. Required when `template_type = "InlineFile"`; must be unset otherwise. The provider re-reads the live content on every `terraform plan`/`refresh`, so an out-of-band edit made directly on the backend shows up as drift and gets reverted to match this value on the next `apply`.
* `script_language` - (Optional) The language of `script_content`. One of `Text`, `Python`, `Batch`, `JavaScript`, `PowerShell`, `Shell` (case-insensitive). Defaults to `text`. Only meaningful when `template_type = "InlineFile"`.
* `variables` - (Optional) A set of variables exposed to the scan script. Each variable supports:
  * `name` - (Required) The variable name.
  * `type` - (Required) The variable type. One of `String`, `Number`, `Date` (case-insensitive).
  * `multi_valued` - (Optional) Whether the variable accepts multiple values. Defaults to `false`.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the scan settings, derived from `resource_type_id` alone.
* `script_file_hash` - SHA-256 hash of the file at `script_file_path`. The provider also hashes the live remote content on every `terraform plan`/`refresh`; a mismatch between the two is shown as drift on this attribute and gets corrected (the local file re-uploaded) on the next `apply`.
* `script_name` - Server-derived script file name: the basename of `script_file_path` for `FilePath` mode, an auto-generated name for `InlineFile` mode, empty for `Local` mode.

## Import

Scan settings can be imported using their composite ID:

```sh
terraform import britive_resource_manager_resource_type_scan_settings.example resource-manager/resource-types/<resource_type_id>/scan-settings
```

`script_content` (`InlineFile` mode) is recovered from the API on import via its presigned
download URL. `script_file_path` (`FilePath` mode) cannot be recovered - only the original
local path was ever known, not the API - and stays null after import; set it explicitly if
you want drift detection against a local source in that mode.

## Delete Behavior

There is no evidence of a delete endpoint for scan settings - it's a resource-type-scoped
singleton, not a removable record. `terraform destroy` instead resets the settings to
`Local` mode with the API's own observed defaults (no script, 20-minute time limit, no
variables), the closest equivalent to "un-configuring" it.
