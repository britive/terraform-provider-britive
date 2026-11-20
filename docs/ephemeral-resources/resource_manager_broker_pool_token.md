---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool_token Ephemeral Resource - britive"
description: |-
  Reads the value of a broker pool token without storing it in the state.
---

# britive_resource_manager_broker_pool_token Ephemeral Resource

The `britive_resource_manager_broker_pool_token` ephemeral resource reads the secret value of a
broker pool token, including the pool's `default` token. Brokers use this value to join the
pool.

The [`britive_resource_manager_broker_pool`](../resources/resource_manager_broker_pool.md)
resource manages the pool's tokens but does not store their values. Use this ephemeral resource
to read a value. Terraform
reads it again on every plan and apply and keeps the value in memory for that run only, so the
value is never written to the state file or to a plan file.

~> **Note:** Ephemeral resources need Terraform 1.10 or later. Write-only arguments, used in
the examples below, need Terraform 1.11 or later. The other broker pool resources work on older
versions too.

Terraform accepts an ephemeral value only in write-only arguments of other resources, in
`provider` blocks and in provisioner blocks. It rejects any use that would store the value, such
as a normal resource argument or a root module output.

## Example Usage

### Store the token in AWS Secrets Manager

This is the preferred pattern. The token goes into a secret store through a write-only
argument, and the broker host reads it from the secret store when it starts.

```hcl
resource "britive_resource_manager_broker_pool" "east" {
  name = "pool-east"

  tokens = {
    default    = { status = "ACTIVE" } # required when the pool is created
    "ci-token" = { status = "ACTIVE" }
  }
}

# Read the token value. Nothing is stored in state.
ephemeral "britive_resource_manager_broker_pool_token" "ci" {
  token_id = britive_resource_manager_broker_pool.east.tokens["ci-token"].id
}

resource "aws_secretsmanager_secret" "broker_token" {
  name = "britive/broker-pools/east/ci-token"
}

resource "aws_secretsmanager_secret_version" "broker_token" {
  secret_id                = aws_secretsmanager_secret.broker_token.id
  secret_string_wo         = ephemeral.britive_resource_manager_broker_pool_token.ci.token_value
  secret_string_wo_version = 1 # increase to send the value again
}
```

### Store the default token in HashiCorp Vault

```hcl
# Read the default token of the pool. This works when the pool keeps its default token.
ephemeral "britive_resource_manager_broker_pool_token" "default" {
  token_id = "${britive_resource_manager_broker_pool.east.pool_id}/default"
}

resource "vault_kv_secret_v2" "broker_token" {
  mount = "secret"
  name  = "britive/broker-pools/east"

  data_json_wo = jsonencode({
    token = ephemeral.britive_resource_manager_broker_pool_token.default.token_value
  })
  data_json_wo_version = 1 # increase to send the value again
}
```

### Pass the token to a script

Use this when the target has no Terraform resource with a write-only argument. This pattern
needs only Terraform 1.10. Pass the token in the environment, not on the command line.

```hcl
resource "terraform_data" "broker_token_secret" {
  # The script runs on create and when one of these values changes.
  triggers_replace = {
    pool_id       = britive_resource_manager_broker_pool.east.pool_id
    token_version = 1
  }

  provisioner "local-exec" {
    command = "python3 ${path.module}/store_broker_token.py"

    environment = {
      BROKER_POOL_TOKEN = ephemeral.britive_resource_manager_broker_pool_token.default.token_value
    }
  }
}
```

The script can store the token anywhere. Terraform does not detect later changes to the stored
value and does not remove it on destroy. Terraform hides all output of a `local-exec` script
that receives an ephemeral value.

## Argument Reference

* `token_id` - (Required) The token to read, as `{pool_id}/{token_name}`. This is the `id` of a
  token in the `tokens` map of the `britive_resource_manager_broker_pool` resource or data
  source. Use `{pool_id}/default` for the pool's default token.

## Attribute Reference

* `pool_id` - The ID of the broker pool.
* `name` - The token name.
* `status` - The token status: `ACTIVE` or `INACTIVE`.
* `token_value` - (Sensitive) The secret value brokers use to join the pool. Never stored in
  the state.

If the token does not exist, the plan or apply fails with the error "Broker Pool Token Not
Found".

## Usage Notes

* Reference a token's `id` from the pool's `tokens` map in `token_id`. The `id` is unknown until
  the token exists, so Terraform waits until the token is created before it reads the value. A
  `token_id` written as a string is read during the plan, before a new token exists.
* You can read only a token that is in the pool's `tokens`. If you remove a token from `tokens`,
  the provider deletes it, so also remove the ephemeral resource that reads it. A reference to a
  key that is not in `tokens`, such as `tokens["default"].id` after `default` was removed, fails
  with Terraform's "Invalid index" error. A `token_id` written as a string fails with "Broker
  Pool Token Not Found" instead.
* The value cannot be shown with an `output`: Terraform rejects ephemeral values in root module
  outputs, and hides all output of a `local-exec` that receives one. To check a value while
  testing, compare a hash of it (for example `sha256sum` in a `local-exec`) with the token in
  the Britive UI.
* A write-only argument is sent again only when its version argument changes. Terraform cannot
  compare the value, because the state holds `null`.
* A token value cannot be changed. To rotate, add a new token to the pool's `tokens` with a new
  name, increase the version argument, move the brokers to the new token, and then remove the
  old token from `tokens`. A pool allows 3 tokens, 2 of them active.
* Terraform rejects the token in normal arguments such as `user_data`, so it cannot leak into
  the state through another resource.
