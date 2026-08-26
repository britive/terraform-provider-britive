---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource_type_rotation_template Resource - britive"
description: |-
  Manages a rotation template for the Britive provider.
---

# britive_resource_manager_resource_type_rotation_template Resource

The `britive_resource_manager_resource_type_rotation_template` resource allows you to manage rotation
templates for a Britive resource manager resource type.

A rotation template has one of three modes, set via `template_type`:

* `Local` - rotation is handled by logic already deployed on the target; no script is uploaded.
* `InlineFile` - the rotation script is authored inline via `script_content`/`script_language`.
* `FilePath` - the rotation script is a local file uploaded via `script_file_path`.

## Example Usage

### Local

```hcl
resource "britive_resource_manager_resource_type_rotation_template" "local_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name             = "LocalTemplate"
  description      = "Rotation handled by logic already deployed on the target"
  time_limit       = 5
  template_type    = "Local"

  variables {
    name         = "username"
    type         = "String"
    multi_valued = false
  }
  variables {
    name         = "expiry_date"
    type         = "Date"
    multi_valued = false
  }
}
```

### Insert Code (InlineFile)

```hcl
resource "britive_resource_manager_resource_type_rotation_template" "insert_code_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name             = "InsertCodePython"
  time_limit       = 10
  template_type    = "InlineFile"
  script_language  = "python"
  script_content   = <<-EOT
    import log
    log.info("rotating credential")
  EOT

  variables {
    name         = "username"
    type         = "String"
    multi_valued = false
  }
}
```

Content can also be authored in a separate file and read in with `file()` - this is still
`InlineFile` mode (uploaded via the inline-code path, `script_name` auto-generated), not
`FilePath` mode (which uploads the raw file as-is and keeps its original name):

```hcl
resource "britive_resource_manager_resource_type_rotation_template" "insert_code_from_file" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name             = "InsertCodePythonFromFile"
  time_limit       = 10
  template_type    = "InlineFile"
  script_language  = "python"
  script_content   = file("${path.module}/scripts/rotate.py")
}
```

### Add File (FilePath)

```hcl
resource "britive_resource_manager_resource_type_rotation_template" "add_file_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name             = "AddFileTemplate"
  time_limit       = 10
  template_type    = "FilePath"
  script_file_path = "${path.module}/scripts/rotate.sh"

  variables {
    name         = "username"
    type         = "String"
    multi_valued = false
  }
}
```

## Argument Reference

* `resource_type_id` - (Required) The ID of the associated resource type. Forces replacement if changed.
* `name` - (Required) The name of the rotation template. Forces replacement if changed - the API never accepts a name change on update.
* `description` - (Optional) The description of the rotation template. Cannot be changed once set - unlike `name`, changing it does not replace the resource; it fails the plan outright, since the API never accepts a description update.
* `time_limit` - (Optional) The time limit in minutes for the rotation script to run (converted to seconds on the wire). Defaults to `1`. The valid range is enforced by the API, not the provider, and may change independently of a provider release.
* `template_type` - (Required) The template mode. Must be one of `Local`, `InlineFile`, `FilePath` (case-insensitive).
* `script_file_path` - (Optional) Path to a local file to upload as the rotation script. Required when `template_type = "FilePath"`; must be unset otherwise.
* `script_content` - (Optional) Inline rotation script content. Required when `template_type = "InlineFile"`; must be unset otherwise.
* `script_language` - (Optional) The language of `script_content`. One of `Text`, `Python`, `Batch`, `JavaScript`, `PowerShell`, `Shell` (case-insensitive). Defaults to `text`. Only meaningful when `template_type = "InlineFile"`.
* `variables` - (Optional) A set of variables exposed to the rotation script. Each variable supports:
  * `name` - (Required) The variable name.
  * `type` - (Required) The variable type. One of `String`, `Number`, `Date` (case-insensitive).
  * `multi_valued` - (Optional) Whether the variable accepts multiple values. Defaults to `false`.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the rotation template.
* `template_id` - The unique identifier of the rotation template.
* `script_file_hash` - SHA-256 hash of the file at `script_file_path`, used to detect content drift.
* `script_name` - Server-derived script file name: the basename of `script_file_path` for `FilePath` mode, an auto-generated name for `InlineFile` mode, absent for `Local` mode.

## Import

Rotation templates can be imported using their composite ID:

```sh
terraform import britive_resource_manager_resource_type_rotation_template.example resource-manager/resource-types/<resource_type_id>/rotation-templates/<template_id>
```

`script_file_path`/`script_content` cannot be recovered from the API on import and will be
null after import; set them explicitly if you want drift detection against a local source.
