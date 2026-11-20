---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool_labels Resource - britive"
description: |-
  Manages all labels of a broker pool for the Britive provider.
---

# britive_resource_manager_broker_pool_labels Resource

The `britive_resource_manager_broker_pool_labels` resource manages all labels of a broker pool
in one resource. Each label key is one `resource_labels` block.

~> **Important:** A pool must have only one `britive_resource_manager_broker_pool_labels`
resource. Put all of the pool's labels in that one resource as separate `resource_labels`
blocks. If two of these resources point to the same pool, each one removes the other's labels
on every apply, and the plan never settles. The provider does not detect this.

## Example Usage

```hcl
resource "britive_resource_manager_broker_pool" "east" {
  name = "pool-east"
  tokens = { default = { status = "ACTIVE" } }
}

resource "britive_resource_manager_broker_pool_labels" "east" {
  pool_id = britive_resource_manager_broker_pool.east.pool_id

  resource_labels {
    label_key = "Environment"
    values    = ["Development", "Staging"]
  }

  resource_labels {
    label_key = "Team"
    values    = []
  }
}
```

## Argument Reference

* `pool_id` - (Required) The ID of the broker pool. Changing it creates a new resource.
* `resource_labels` - (Required) The labels assigned to the pool. At least one block is required. Each block supports:
  * `label_key` - (Required) The label key. Only letters, numbers, `_` and `-`. Each `label_key` can appear in only one block.
  * `values` - (Required) The label values. Can be empty (`[]`).

To remove all labels from the pool, remove this resource.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The identifier of the pool's labels, in the form `resource-manager/broker-pools/{pool_id}/labels`.

## Import

The labels of a pool can be imported using the pool ID or the full ID. Every label on the pool
is imported:

```sh
terraform import britive_resource_manager_broker_pool_labels.east {pool_id}
terraform import britive_resource_manager_broker_pool_labels.east resource-manager/broker-pools/{pool_id}/labels
```

## Delete Behavior

Destroying this resource removes every label key it manages from the pool. If the pool was
already deleted (which also deletes its labels), the delete succeeds without error.
