---
subcategory: ""
layout: "britive"
page_title: "britive_role Resource - britive"
description: |-
  Manages roles for the Britive provider.
---

# britive_role Resource

-> When using this version for the first time, you may encounter noisy diffs caused by the reordering of resource argument values. 

This resource allows you to create and configure a role.

## Example Usage

```hcl
resource "britive_role" "new" {
    name        = "My Role"
    description = "My Role description"
    permissions = jsonencode(
        [
            {
                name = "My Permission"
            },
            {
                name = britive_permission.new.name
            }
        ]
    )
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the role.

* `description` - (Optional) A description of the role.

* `permissions` - (Required) Names of the permissions associated to the role. This is a JSON formatted string. Mimimum of 1 permission is required to create a role. `name` corresponds to the name of the permission


## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with format `roles/{{roleID}}`

## Import

You can import a role using any of these accepted formats:

```SH
terraform import britive_role.new roles/{{name}}
terraform import britive_role.new {{name}}
```
