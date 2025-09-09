---
subcategory: "resource_manager"
layout: "britive"
page_title: "britive_resource_manager_resource_type_permission Resource - britive"
description: |-
  Manages resource type permission for the Britive provider.
---

# britive_resource_manager_resource_type_permission Resource

The `britive_resource_manager_resource_type_permission` resource allows you to create, update, and manage resource type permissions in Britive.

  
-> You can attach scripts or commands to permissions in two ways:
- **File-based code:** Reference external files for check-in and check-out scripts. This is useful for managing scripts separately, reusing them, or keeping your Terraform configuration clean.
- **Inline code:** Provide the script directly in your Terraform configuration. This is convenient for small scripts or when you want everything defined in a single place.

## Example Usage

### Example 1: Using Code Files

```hcl
resource "britive_resource_manager_resource_type_permission" "example_file" {
    name                = "ys-example-permission-file"
    resource_type_id    = "iwau98weuwiodja98du3w9sid98"
    description         = "An example resource type permission using code files"
    checkin_time_limit  = 160
    checkout_time_limit = 360
    is_draft            = false
    show_orig_creds     = true
    variables           = ["test1", "test2"]
    checkin_code_file   = "checkin_command.txt"
    checkout_code_file  = "checkout_command.txt"
    response_templates  = ["TF_RESOURCE_TEMPLATE_1", "TF_RESOURCE_TEMPLATE_2"]
}
```

### Example 2: Using Inline Code

```hcl
resource "britive_resource_manager_resource_type_permission" "example_inline" {
    name                = "ys-example-permission-inline"
    resource_type_id    = "iwau98weuwiodja98du3w9sid98"
    description         = "An example resource type permission using inline code"
    checkin_time_limit  = 120
    checkout_time_limit = 240
    is_draft            = true
    show_orig_creds     = false
    variables           = ["inline1", "inline2"]
    code_language       = "Python"
    checkin_code        = <<EOT
        #!/bin/bash
        echo "Running check-in task"
    EOT
    checkout_code       = <<EOT
        #!/bin/bash
        echo "Running check-out task"
    EOT
    response_templates  = ["TF_RESOURCE_TEMPLATE_1", "TF_RESOURCE_TEMPLATE_2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the permission. Must be unique.
* `resource_type_id` - (Required) The ID of the associated resource type.
* `description` - (Optional) The description of the permission. Default is an empty string.
* `checkin_time_limit` - (Optional) The check-in time limit in minutes. Default is `60`.
* `checkout_time_limit` - (Optional) The check-out time limit in minutes. Default is `60`.
* `is_draft` - (Optional) Indicates if the permission is a draft. Default is `false`.
* `show_orig_creds` - (Optional) Indicates if original credentials should be shown. Default is `false`.
* `variables` - (Optional) List of variables.
* `checkin_code_file` - (Optional) The file path for check-in code. Conflicts with `checkin_code`, `checkout_code`, and `code_language`.
* `checkout_code_file` - (Optional) The file path for check-out code. Conflicts with `checkin_code`, `checkout_code`, and `code_language`.
* `checkin_code` - (Optional) The inline check-in code. Conflicts with `checkin_code_file` and `checkout_code_file`.
* `checkout_code` - (Optional) The inline check-out code. Conflicts with `checkin_code_file` and `checkout_code_file`.
* `code_language` - (Optional) The inline code language. Select one of `Text`, `Batch`, `Node`, `PowerShell`, `Python`, `Shell`. Default is `Text`.
* `response_templates` - (Optional) List of response template names.

> **Note:**  
> - `checkin_code_file` and `checkout_code_file` must both be set together or both left unset.  
> - `checkin_code` and `checkout_code` must both be set together or both left unset.  
> - You cannot set both file-based (`checkin_code_file`, `checkout_code_file`) and inline code (`checkin_code`, `checkout_code`, `code_language`) options at the same time.  
> - `show_orig_creds` must be `true` or at least one `response_templates` value must be set. Both `show_orig_creds` and `response_templates` cannot be empty or false at the same time.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `permission_id` - The ID of the permission.
* `version` - The version for the permission.
* `inline_file_exists` - Indicates if an inline file exists.
* `checkin_code_file_hash` - The file hash for check-in code.
* `checkout_code_file_hash` - The file hash for check-out code.
* `checkin_file_name` - The name of the check-in file.
* `checkout_file_name` - The name of the check-out file.

## Import

Resource type permissions can be imported using their `permission_id`:

```sh
terraform import britive_resource_manager_resource_type_permission.example resource-manager/permissions/{{permission_id}}
terraform import britive_resource_manager_resource_type_permission.example resource-manager/permissions/djjfh8wr73w9ruhwe8r23uyf
```
```sh
terraform import britive_resource_manager_resource_type_permission.example {{permission_id}}
terraform import britive_resource_manager_resource_type_permission.example djjfh8wr73w9ruhwe8r23uyf
```
