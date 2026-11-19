---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool Resource - britive"
description: |-
  Manages a broker pool for the Britive provider.
---

# britive_resource_manager_broker_pool Resource

The `britive_resource_manager_broker_pool` resource manages a broker pool. A broker pool is a
group of brokers, the small programs that run inside a private network so Britive can reach
servers there.

This resource manages the pool itself. Britive creates a `default` token with every pool; this
resource does not read or change it. To manage the `default` token, or to add more tokens, use
[`britive_resource_manager_broker_pool_token`](resource_manager_broker_pool_token.md) (with
`name = "default"` for the default token). To add
labels, use
[`britive_resource_manager_broker_pool_labels`](resource_manager_broker_pool_labels.md). To link
the pool to a resource, use
[`britive_resource_manager_resource_broker_pools`](resource_manager_broker_pools.md).

To see the pool's brokers and linked resources, use the
[`britive_resource_manager_broker_pool`](../data-sources/resource_manager_broker_pool.md) data
source.

## Example Usage

```hcl
resource "britive_resource_manager_broker_pool" "east" {
  name               = "pool-east"
  description        = "Brokers in the east data center"
  keep_alive_seconds = 120
}
```

## Argument Reference

* `name` - (Required) The name of the broker pool. Can be changed in place.
* `description` - (Optional) The description of the broker pool. Defaults to an empty string.
* `keep_alive_seconds` - (Optional) How often, in seconds, brokers check in with Britive. Defaults to `60`.
* `disconnect_seconds` - (Optional) After how many seconds without a check-in a broker is marked disconnected. Defaults to `86400`.

Removing `description`, `keep_alive_seconds` or `disconnect_seconds` from the configuration
resets it to its default.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The identifier of the broker pool, in the form `resource-manager/broker-pools/{pool_id}`.
* `pool_id` - The unique identifier of the broker pool. Pass this to `britive_resource_manager_broker_pool_token`.
* `broker_count` - The number of brokers in the pool.

## Import

Broker pools can be imported using the pool ID or the full ID:

```sh
terraform import britive_resource_manager_broker_pool.east {pool_id}
terraform import britive_resource_manager_broker_pool.east resource-manager/broker-pools/{pool_id}
```

## Delete Behavior

Destroying this resource deletes the broker pool. Britive also deletes all of the pool's
tokens, labels and resource links.
