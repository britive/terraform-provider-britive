---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool Data Source - britive"
description: |-
  Retrieves an existing broker pool by name.
---

# britive_resource_manager_broker_pool Data Source

This data source retrieves an existing broker pool by its exact name, together with its active
brokers and the resources linked to it. It does not return token values.

## Example Usage

```hcl
data "britive_resource_manager_broker_pool" "shared" {
  name = "shared-pool"
}

resource "britive_resource_manager_resource_broker_pools" "db" {
  resource_id  = britive_resource_manager_resource.db.id
  broker_pools = [data.britive_resource_manager_broker_pool.shared.name]
}

output "shared_pool_brokers" {
  value = data.britive_resource_manager_broker_pool.shared.brokers[*].broker_name
}
```

## Argument Reference

The following argument is supported:

- `name` (Required) – The exact name of the broker pool. If no pool, or more than one pool, has this name, the data source returns an error.

## Attribute Reference

The following attributes are exported:

- `id` – The identifier of the broker pool, in the form `resource-manager/broker-pools/{pool_id}`.
- `pool_id` – The unique identifier of the broker pool.
- `description` – The description of the broker pool.
- `keep_alive_seconds` – How often, in seconds, brokers check in with Britive.
- `disconnect_seconds` – After how many seconds without a check-in a broker is marked disconnected.
- `broker_count` – The number of brokers in the pool.
- `brokers` – The active brokers in the pool. Each item has `broker_id`, `broker_name`, `status`, `status_timestamp` and `created`.
- `resources` – The resources linked to the pool. Each item has `resource_id`, `name` and `resource_type_name`.
