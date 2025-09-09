---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource Resource - britive"
description: |-
  Manages resources for the Britive provider.
---

# britive_resource_manager_resource Resource

The `britive_resource_manager_resource` resource allows you to create, update, and manage server access resources in Britive.

## Example Usage

```hcl
resource "britive_resource_manager_resource" "example" {
    name           = "prod-server-access"
    description    = "Access to production server"
    resource_type  = "LinuxServer"

    parameter_values = {
        "hostname" = "prod-server-01"
        "port"     = "22"
    }

    resource_labels = {
        "environment" = "Production"
        "region"      = "us-east-1"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the server access resource.
* `description` - (Optional) Description of the server access resource.
* `resource_type` - (Required) The resource type name associated with the server access resource.
* `parameter_values` - (Optional) Map of parameter values for the fields of the resource type.
* `resource_labels` - (Optional) Map of resource labels associated with the server access resource.

## Attribute Reference

In addition to the arguments above, the following attribute is exported:

* `resource_type_id` - The ID of the resource type associated with this server access resource.

## Import

Server access resources can be imported using their name:

```sh
terraform import britive_resource_manager_resource.example resources/{resource_name}
terraform import britive_resource_manager_resource.example resources/prod-server-access
```