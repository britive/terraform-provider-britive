---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool_token Resource - britive"
description: |-
  Manages a token of a broker pool for the Britive provider.
---

# britive_resource_manager_broker_pool_token Resource

The `britive_resource_manager_broker_pool_token` resource manages a token of a broker pool.
Brokers use a token to join the pool.

A pool can have at most 3 tokens, and at most 2 can be active at the same time. A pool must
always keep at least 1 active token. The `default` token that Britive creates with every pool
counts toward these limits.

With `name = "default"`, this resource manages the pool's `default` token. See
[Managing the default token](#managing-the-default-token).

This resource does not store the token value in the Terraform state. To read the value, use the
[`britive_resource_manager_broker_pool_token`](../ephemeral-resources/resource_manager_broker_pool_token.md)
ephemeral resource (Terraform 1.10 or later), which never writes it to the state or to a plan
file.

## Example Usage

```hcl
resource "britive_resource_manager_broker_pool" "east" {
  name = "pool-east"
}

resource "britive_resource_manager_broker_pool_token" "ci" {
  pool_id     = britive_resource_manager_broker_pool.east.pool_id
  name        = "ci-token"
  description = "Token for CI brokers"
  status      = "ACTIVE"
}

# Read the token value without storing it in the state.
ephemeral "britive_resource_manager_broker_pool_token" "ci" {
  token_id = britive_resource_manager_broker_pool_token.ci.id
}
```

### Manage the default token

```hcl
resource "britive_resource_manager_broker_pool_token" "default" {
  pool_id     = britive_resource_manager_broker_pool.east.pool_id
  name        = "default"
  description = "Default token for east brokers"
  status      = "ACTIVE"
}
```

## Argument Reference

* `pool_id` - (Required) The ID of the broker pool. Changing it creates a new token.
* `name` - (Required) The token name. Only letters, numbers, `_` and `-`. Must be unique in the pool. `default` manages the pool's default token. Changing it creates a new token.
* `status` - (Required) `ACTIVE` or `INACTIVE`. Britive creates every new token `INACTIVE`, so for `ACTIVE` the provider activates it right after creating it.
* `description` - (Optional) The token description. If it is not set, the description is empty; removing it clears the description.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The identifier of the token, in the form `resource-manager/broker-pools/{pool_id}/tokens/{name}`.
* `created` - When the token was created.

~> **Note:** The token value is not an attribute of this resource and is never stored in the
Terraform state. Read it with the
[`britive_resource_manager_broker_pool_token`](../ephemeral-resources/resource_manager_broker_pool_token.md)
ephemeral resource and pass it to a write-only argument, for example of a secret store.

## Managing the default token

Britive creates a `default` token with every pool. It is `ACTIVE` and its description is
"default token for pool". A resource with `name = "default"` does not create a second token: it
takes over the existing one and sets the configured `status` and `description` on it. If the
`default` token was deleted earlier, it is created again.

~> **Important:** A pool must have only one `britive_resource_manager_broker_pool_token` resource
with `name = "default"`. The provider does not detect a second one. If two resources manage the
same default token with different values, each apply changes the token back and forth.

* **Rejected changes on create.** If Britive rejects the change while taking over the token (for
  example, `status = "INACTIVE"` while it is the pool's only active token), the apply fails with
  Britive's message and the resource is not saved in the state. The token itself is not changed
  or deleted, so brokers that use it keep working. The next apply tries the takeover again; fix
  the configuration first, for example by making another token in the pool `ACTIVE`.
* **On a new pool,** the `default` token is the only token and it is `ACTIVE`, so
  `status = "INACTIVE"` is rejected. Set `status = "ACTIVE"`, or make another token `ACTIVE`
  first (use `depends_on` so that token is created first).
* **Deleting.** Destroying the resource deletes the `default` token. If Britive rejects the
  delete (for example, because it is the pool's last active token), the resource is removed from
  the Terraform state with a warning. The token stays in Britive and is deleted together with the
  pool.
* **Importing.** Import it with `{pool_id}/default`. Set `status` and `description` in the
  configuration to the token's current values; otherwise the next plan changes them.

## Import

Tokens can be imported using the pool ID and token name, or the full ID:

```sh
terraform import britive_resource_manager_broker_pool_token.ci {pool_id}/{name}
terraform import britive_resource_manager_broker_pool_token.ci resource-manager/broker-pools/{pool_id}/tokens/{name}
terraform import britive_resource_manager_broker_pool_token.default {pool_id}/default
```

## Delete Behavior

Destroying this resource deletes the token. If the pool was already deleted (which also deletes
its tokens), the delete succeeds without error. Britive does not allow deleting the pool's last
active token: for a normal token the delete then fails with an error and the resource stays in
the state; for the `default` token see [Managing the default token](#managing-the-default-token).
