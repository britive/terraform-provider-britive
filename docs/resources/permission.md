---
subcategory: ""
layout: "britive"
page_title: "britive_permission Resource - britive"
description: |-
  Manages permissions for the Britive provider.
---

# britive_permission Resource

This resource allows you to create and configure a permission.

## Example Usage

```hcl
resource "britive_permission" "new" {
    name        = "My Permission"
    description = "View permission description"
    consumer    = "authz"
    resources   = [
        "*",
    ]
    actions     = [
        "authz.action.list",
        "authz.action.read",
    ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the permission.

* `description` - (Optional) A description of the permission.

* `consumer` - (Required) A component/entity that will use the policy engine for access decisions.

* `resources` - (Required) A resource in the definition of the corresponding consumer, or '*' (meaning any).

* `actions` - (Required) A set of pre-defined actions for each consumer.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with format `permissions/{{permissionID}}`

## Import

You can import a permission using any of these accepted formats:

```SH
terraform import britive_permission.new permissions/{{name}}
terraform import britive_permission.new {{name}}
```
