---
subcategory: "resource_manager"
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
}
```

## Argument Reference

* `name` - (Required) The name of the Britive resource type. Only letters, numbers, hyphens (`-`), and underscores (`_`) are allowed, no other special characters. Used to uniquely identify the resource type within Britive.
* `description` - (Optional) The description of the Britive resource type.
* `icon` - (Required) The icon of the Britive resource type
* `parameters` - (Optional) A set of parameters/fields for the resource type. Each parameter supports the following attributes:
  * `param_name` - (Required) The name of the parameter. Only letters, numbers, hyphens (`-`), and underscores (`_`) are allowed, no other special characters.
  * `param_type` - (Required) The type of the parameter. Must be one of `string` or `password` (case-insensitive).
  * `is_mandatory` - (Required) A boolean indicating whether the parameter is mandatory.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The unique identifier of the resource type.

## Import

Resource types can be imported using their ID:

```sh
terraform import britive_resource_manager_resource_type.example resource-manager/resource-types/<resource_type_id>
```