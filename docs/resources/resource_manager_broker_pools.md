---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource_broker_pools Resource - britive"
description: |-
  Manages resource broker pools for the Britive provider.
---

# britive_resource_manager_resource_broker_pools Resource

The `britive_resource_manager_resource_broker_pools` resource allows you to associate broker pools with a server access resource in Britive.

## Example Usage

```hcl
resource "britive_resource_manager_resource_broker_pools" "example" {
    resource_id  = "aius3dsadv8c02xi8j4"
    broker_pools = [
        "pool-east-1",
        "pool-west-2"
    ]
}
```

## Argument Reference

The following arguments are supported:

* `resource_id` - (Required, ForceNew) The ID of the server access resource to associate broker pools with.
* `broker_pools` - (Required, ForceNew) List of broker pool names to be associated with the resource.

## Import

Broker pool associations can be imported using the resource ID:

```sh
terraform import britive_resource_manager_resource_broker_pools.example resources/{resource_id}/broker-pools
terraform import britive_resource_manager_resource_broker_pools.example resources/aius3dsadv8c02xi8j4/broker-pools
```