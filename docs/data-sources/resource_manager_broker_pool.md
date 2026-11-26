---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool Data Source - britive"
description: |-
  Retrieves an existing broker pool by name.
---

# britive_resource_manager_broker_pool Data Source

This data source retrieves an existing broker pool by its exact name, together with its active
brokers, the resources linked to it and its tokens. It does not return token values.

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

# Read the value of the pool's default token without storing it in the state.
ephemeral "britive_resource_manager_broker_pool_token" "shared_default" {
  token_id = data.britive_resource_manager_broker_pool.shared.tokens["default"].id
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
- `tokens` – The tokens of the pool, as a map with the token name as the key. Each token has:
  - `status` – `ACTIVE` or `INACTIVE`.
  - `description` – The token description.
  - `id` – The token ID, in the form `{pool_id}/{token_name}`. Pass it to the `token_id` of the [`britive_resource_manager_broker_pool_token`](../ephemeral-resources/resource_manager_broker_pool_token.md) ephemeral resource.
  - `created` – When the token was created.

~> **Note:** Token values are not attributes of this data source and are never stored in the
Terraform state. Read a value with the
[`britive_resource_manager_broker_pool_token`](../ephemeral-resources/resource_manager_broker_pool_token.md)
ephemeral resource.
