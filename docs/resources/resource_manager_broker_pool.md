---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool Resource - britive"
description: |-
  Manages a broker pool and its tokens for the Britive provider.
---

# britive_resource_manager_broker_pool Resource

The `britive_resource_manager_broker_pool` resource manages a broker pool and its tokens. A
broker pool is a group of brokers, the small programs that run inside a private network so
Britive can reach servers there. Brokers use one of the pool's tokens to join it.

Tokens are set in the required `tokens` argument, a map with the token name as the key. See
[Tokens](#tokens). To add labels, use
[`britive_resource_manager_broker_pool_labels`](resource_manager_broker_pool_labels.md). To link
the pool to a resource, use
[`britive_resource_manager_resource_broker_pools`](resource_manager_broker_pools.md).

To see the pool's brokers and linked resources, use the
[`britive_resource_manager_broker_pool`](../data-sources/resource_manager_broker_pool.md) data
source.

## Example Usage

```hcl
# A pool that keeps only its default token.
resource "britive_resource_manager_broker_pool" "west" {
  name   = "pool-west"
  tokens = { default = { status = "ACTIVE" } }
}

resource "britive_resource_manager_broker_pool" "east" {
  name               = "pool-east"
  description        = "Brokers in the east data center"
  keep_alive_seconds = 120

  # The exact set of tokens on the pool.
  tokens = {
    default    = { status = "ACTIVE", description = "Default token for east brokers" }
    "ci-token" = { status = "ACTIVE" }
  }
}

# Read a token value without storing it in the state.
ephemeral "britive_resource_manager_broker_pool_token" "ci" {
  token_id = britive_resource_manager_broker_pool.east.tokens["ci-token"].id
}
```

## Argument Reference

* `name` - (Required) The name of the broker pool. Can be changed in place.
* `description` - (Optional) The description of the broker pool. Defaults to an empty string.
* `keep_alive_seconds` - (Optional) How often, in seconds, brokers check in with Britive. Defaults to `60`.
* `disconnect_seconds` - (Optional) After how many seconds without a check-in a broker is marked disconnected. Defaults to `86400`.
* `tokens` - (Required) The tokens of the pool, as a map with the token name as the key. It must have at least one token, and when the pool is created it must include `default`. Britive allows at most 3 tokens, at most 2 of them `ACTIVE`, and at least 1 `ACTIVE`. The map is the exact set of tokens: a token in Britive that is not in the map is deleted. To keep only the default token, set `tokens = { default = { status = "ACTIVE" } }`. See [Tokens](#tokens). Each token supports:
  * `status` - (Required) `ACTIVE` or `INACTIVE`.
  * `description` - (Optional) The token description. Defaults to an empty string.

Removing `description`, `keep_alive_seconds` or `disconnect_seconds` from the configuration
resets it to its default.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The identifier of the broker pool, in the form `resource-manager/broker-pools/{pool_id}`.
* `pool_id` - The unique identifier of the broker pool.
* `broker_count` - The number of brokers in the pool.
* `tokens.<name>.id` - The token ID, in the form `{pool_id}/{token_name}`. Pass it to the
  `token_id` of the
  [`britive_resource_manager_broker_pool_token`](../ephemeral-resources/resource_manager_broker_pool_token.md)
  ephemeral resource. It is unknown until the token exists.
* `tokens.<name>.created` - When the token was created.

~> **Note:** Token values are not attributes of this resource and are never stored in the
Terraform state or in a plan file. Read a value with the
[`britive_resource_manager_broker_pool_token`](../ephemeral-resources/resource_manager_broker_pool_token.md)
ephemeral resource (Terraform 1.10 or later) and pass it to a write-only argument, for example
of a secret store.

## Tokens

Britive does not allow a pool without an `ACTIVE` token, so tokens are managed inside the pool
resource instead of as separate resources. This lets the provider order the token changes, and
lets one delete remove the pool together with its tokens.

### Rules

The provider checks these rules in the plan, before any API call:

* `tokens` must have at least 1 token.
* When the pool is created, `tokens` must include `default` (see below).
* A token name allows letters, numbers, `_` and `-`.

Britive also limits the tokens of every pool: at most 3 tokens, at most 2 of them `ACTIVE`, and
at least 1 `ACTIVE`. The provider does not check these limits in the plan. If `tokens` breaks
one, the provider first makes the changes Britive accepts, then Britive rejects the next one
with its own error (for example, "There must be at least 1 remaining active token"). The state
keeps the tokens Britive holds, and the next plan shows what is left to change. On a new pool,
the pool is already created at that point, so it is saved as tainted and replaced on the next
apply.

A token name cannot be changed: a new key is a new token with a new value.

### The default token

Britive creates a token named `default` with every pool. It is `ACTIVE` and its description is
"default token for pool".

When the pool is created, `tokens` must include `default`. The provider does not create a
second token: it takes over the existing one and sets its status and description from the
configuration. A missing description means an empty string. The plan fails if `default` is
missing on create.

After the pool exists, `default` is a normal token. You can change its status or description,
or remove it from `tokens`, which deletes it from Britive like any other token (after the
other tokens are created and activated). If the pool is later created again (for example,
because it was deleted outside Terraform), `tokens` must include `default` again.

A deleted default token is created again when `default` is added back to `tokens`. It gets a
new value.

### How the provider changes tokens

On create and update, the provider reads the pool's tokens and makes one API call at a time
until they match `tokens`. A new token is always `ACTIVE` before the token it replaces is
deleted, so the pool never loses its last active token. When the pool has 3 tokens, its only
`ACTIVE` token must be deleted and the other two stay `INACTIVE`, the provider activates one of
the two for a short time and sets it back to `INACTIVE` at the end.

If a call fails, the provider saves the tokens that Britive really holds and returns the error.
The next plan shows what is left to change.

Tokens changed outside Terraform are changed back on the next apply: an extra token is deleted,
and a missing token is created again (with a new value).

### Rotating a token

1. Add the new token to `tokens` with status `ACTIVE` and apply.
2. Move the brokers to the new token.
3. Remove the old token from `tokens` and apply.

## Import

Broker pools can be imported using the pool ID or the full ID:

```sh
terraform import britive_resource_manager_broker_pool.east {pool_id}
terraform import britive_resource_manager_broker_pool.east resource-manager/broker-pools/{pool_id}
```

Importing a pool also reads the name, status and description of each token into the state,
never the token value. Tokens need no import of their own. Set `tokens` in the configuration to
the imported tokens; any token that differs is changed on the next apply, and a token that is
not in `tokens` is deleted.

## Delete Behavior

Destroying this resource deletes the broker pool. Britive also deletes all of the pool's
tokens, labels and resource links.
