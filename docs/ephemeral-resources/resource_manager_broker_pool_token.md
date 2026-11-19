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

The [`britive_resource_manager_broker_pool_token`](../resources/resource_manager_broker_pool_token.md)
resource does not store the token value. Use this ephemeral resource to read it. Terraform
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
resource "britive_resource_manager_broker_pool_token" "ci" {
  pool_id = britive_resource_manager_broker_pool.east.pool_id
  name    = "ci-token"
  status  = "ACTIVE"
}

# Read the token value. Nothing is stored in state.
ephemeral "britive_resource_manager_broker_pool_token" "ci" {
  token_id = britive_resource_manager_broker_pool_token.ci.id
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
# Read the default token of the pool. No token resource is needed for this.
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

* `token_id` - (Required) The token to read. Either the `id` of a
  `britive_resource_manager_broker_pool_token` resource
  (`resource-manager/broker-pools/{pool_id}/tokens/{name}`), or `{pool_id}/{name}`. Use
  `{pool_id}/default` for the pool's default token.

## Attribute Reference

* `pool_id` - The ID of the broker pool.
* `name` - The token name.
* `status` - The token status: `ACTIVE` or `INACTIVE`.
* `token_value` - (Sensitive) The secret value brokers use to join the pool. Never stored in
  the state.

If the token does not exist, the plan or apply fails with an error.

## Usage Notes

* Reference the token resource's `id` in `token_id`. The `id` is unknown until the token
  exists, so Terraform waits until the token is created before it reads the value.
* A write-only argument is sent again only when its version argument changes. Terraform cannot
  compare the value, because the state holds `null`.
* A token value cannot be changed. To rotate, create a new token with a new name, increase the
  version argument, move the brokers to the new token, and then delete the old token. A pool
  allows 3 tokens, 2 of them active.
* Terraform rejects the token in normal arguments such as `user_data`, so it cannot leak into
  the state through another resource.
