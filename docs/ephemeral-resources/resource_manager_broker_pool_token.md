---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_broker_pool_token Ephemeral Resource - britive"
description: |-
  Reads the value of a broker pool token without storing it in the state.
---

# britive_resource_manager_broker_pool_token Ephemeral Resource

The `britive_resource_manager_broker_pool_token` ephemeral resource reads the secret value of a
broker pool token. Brokers use this value to join the
pool.

The [`britive_resource_manager_broker_pool`](../resources/resource_manager_broker_pool.md)
resource manages the pool's tokens but does not store their values. Use this ephemeral resource
to read a value. Terraform
reads it again on every plan and apply and keeps the value in memory for that run only, so the
value is never written to the state file.

~> **Note:** Ephemeral resources need Terraform 1.10 or later. Write-only arguments, used in the examples below, need Terraform 1.11 or later.

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
  secret_string_wo_version = 1 # increase when the token value changes, see below
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
  data_json_wo_version = 1 # increase when the token value changes, see below
}
```

### Start a broker with the token

Use this when the broker runs on the same host as Terraform. This pattern needs only Terraform
1.10. The token is passed to the broker in the environment, not on the command line or in a
file.

```hcl
resource "terraform_data" "broker" {
  # The broker is started on create and when one of these values changes. The id stays the
  # same when the default token is created again in the same pool ({pool_id}/default), but
  # created changes, so the broker is restarted with the new value.
  triggers_replace = {
    default_token_id      = britive_resource_manager_broker_pool.east.tokens["default"].id
    default_token_created = britive_resource_manager_broker_pool.east.tokens["default"].created
    version               = 1 # increase to restart the broker
  }

  provisioner "local-exec" {
    # Stop any running broker, then start the new one in the background so terraform apply
    # does not wait for it. The last line fails the apply if the broker is not running.
    command = <<-EOT
      cd /opt/britive-broker
      pkill -x britive-broker || true
      sleep 2
      nohup setsid ./britive-broker run > broker.log 2>&1 < /dev/null &
      sleep 5
      pgrep -x britive-broker > /dev/null
    EOT

    environment = {
      BRITIVE_BROKER_AUTH_TOKEN = ephemeral.britive_resource_manager_broker_pool_token.default.token_value
    }
  }
}
```

Terraform hides all output of a `local-exec` script that receives an ephemeral value, so check
`broker.log` in the broker directory to see how the broker started. Terraform does not stop the
broker on destroy.

## When the token value changes

A token keeps its value for its lifetime; Britive does not change it in place. The value behind
the configured token changes only when:

* the configuration switches to another token key;
* the token is deleted outside Terraform, and the next apply creates it again (same key, new
  value);
* the pool is deleted outside Terraform, and the next apply creates it again (new `pool_id`,
  new values).

Terraform sends a write-only value only when the resource is created or its `*_wo_version`
changes. In each case above, increase `secret_string_wo_version` (or `data_json_wo_version`) in
the same apply. Otherwise the secret store keeps a value that no longer works, and the brokers
fail to join the pool. Restart the brokers afterwards so they read the new value.

To send the value again automatically, replace the secret version whenever the token's
`created` changes. `created` changes in all three cases; `id` does not change when a token is
created again in the same pool.

```hcl
# Changes whenever the token is created again, which means it has a new value.
resource "terraform_data" "ci_token" {
  input = britive_resource_manager_broker_pool.east.tokens["ci-token"].created
}

resource "aws_secretsmanager_secret_version" "broker_token" {
  secret_id                = aws_secretsmanager_secret.broker_token.id
  secret_string_wo         = ephemeral.britive_resource_manager_broker_pool_token.ci.token_value
  secret_string_wo_version = 1

  lifecycle {
    replace_triggered_by = [terraform_data.ci_token]
  }
}
```

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