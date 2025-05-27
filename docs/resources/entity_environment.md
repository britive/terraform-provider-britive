# britive_entity_environment Resource

The `britive_entity_environment` resource is used to manage application entity environments in Britive.

## Example Usage

```hcl
resource "britive_entity_environment" "new" {
  application_id  = britive_application.application_snowflake_standalone.id
  parent_group_id = britive_application.application_snowflake_standalone.entity_root_environment_group_id

  properties {
    name  = "displayName"
    value = "Snowflake Env Name"
  }
  properties {
    name  = "description"
    value = "Snowflake Env Description"
  }
  properties {
    name  = "loginNameForAccountMapping"
    value = false
  }
  properties {
    name  = "snowflakeSchemaScanFilter"
    value = true
  }
  properties {
    name  = "accountId"
    value = "AccountID"
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://snowflake.test.com"
  }
  properties {
    name  = "username"
    value = "snowflakeUser"
  }
  properties {
    name  = "role"
    value = "ROLE"
  }

  sensitive_properties {
    name  = "privateKeyPassword"
    value = "Password"
  }
  sensitive_properties {
    name  = "publicKey"
    value = file("${path.module}/snowflake_pb.key")
  }
  sensitive_properties {
    name  = "privateKey"
    value = file("${path.module}/snowflake_pr.key")
  }
}
```

## Argument Reference

- `application_id` - (Required) The ID of the Britive application.
- `parent_group_id` - (Required) The parent group ID under which the environment will be created.
- `entity_id` - (Computed) The ID of the application entity of type environment.
- `properties` - (Optional) A set of key-value pairs representing properties for the environment.
  - `name` - (Required) The name of the property. 
  Supported properties names include:
    - `displayName`: Environment Name.
    - `description`: Environment Description.
    - `appAccessMethod_static_loginUrl`: Login URL.
    - `username`: Username of the User in Snowflake.
    - `accountId`: Account ID.
    - `role`: Custom Role assigned to the user.
    - `loginNameForAccountMapping`: Use login name for account mapping.
    - `snowflakeSchemaScanFilter`: Skip collecting schema level privileges.
  - `value` - (Required) The value of the property.
- `sensitive_properties` - (Optional) A set of key-value pairs representing sensitive properties for the environment.
  - `name` - (Required) The name of the sensitive property. Supported properties names include:
    - `privateKey`: Private Key configured for the user.
    - `privateKeyPassword`: Password of the Private Key.
    - `publicKey`: Public Key configured for the user.
  - `value` - (Required) The value of the sensitive property.

## Import

Entity environments can be imported using the following format:

```sh
terraform import britive_entity_environment.example <application_id>/environments/<entity_id>
```
