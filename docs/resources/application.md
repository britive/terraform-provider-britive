# britive_application Resource

The `britive_application` resource allows you to create and manage applications in Britive.

## Example Usage Snowflake APP

```hcl
resource "britive_application" "application_test_1" {
  application_type = "Snowflake"
  
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "Snowflake App 1"
  }
  properties {
    name  = "description"
    value = "Britive Snowflake App"
  }
  properties {
    name  = "loginNameForAccountMapping"
    value = true
  }
  properties {
    name  = "accountId"
    value = "AccountId"
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://<url>"
  }
  properties {
    name  = "username"
    value = "user1"
  }
  properties {
    name  = "role"
    value = "Role1"
  }
  properties {
    name  = "snowflakeSchemaScanFilter"
    value = false
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 1500
  }
  properties {
    name  = "copyAppToEnvProps"
    value = false
  }
  properties {
    name  = "copyAppToEnvProps"
    value = false
  }
  properties {
    name  = "iconUrl"
    value = "<URL>"
  }
  sensitive_properties {
    name  = "publicKey"
    value = file("${path.module}/rsa_key_public.pub")
  }
  sensitive_properties {
    name  = "privateKey"
    value = file("${path.module}/rsa_key.p8")
  }
}
```

> **Note:** Following `properties` and `sensitive_properties` in the above example are mandatory for creating a valid Snowflake application.  
> **Properties** include:
> - `displayName`
> - `description`
> - `loginNameForAccountMapping`
> - `accountId`
> - `appAccessMethod_static_loginUrl`
> - `username`
> - `role`  
> - `snowflakeSchemaScanFilter`
> - `maxSessionDurationForProfiles`
> - `copyAppToEnvProps`
> - `iconUrl`

**Sensitive Properties** include:
> - `privateKeyPassword`
> - `publicKey`
> - `privateKey`

## Example Usage Snowflake Standalone APP
```
resource "britive_application" "apllication_snowflake_standalone" {
    application_type = "snowFlake standalone"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "description"
      value = "Britive Snowflake Standalone APP"
    }
    properties {
      name = "displayName"
      value = "Snowflake Standalone APP"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
    properties {
      name  = "iconUrl"
      value = "<URL>"
   }
}
```

> **Note:** Following `properties` in the above Snowflake Standalone example are mandatory for creating a valid Snowflake Standalone application.  
> - `displayName`
> - `description`
> - `maxSessionDurationForProfiles`
> - `iconUrl`

## Argument Reference

### Required

- `application_type` - (Required) The type of the application. Supported types are `Snowflake` and `Snowflake Standalone`.

### Optional

- `user_account_mappings` - (Optional) A block defining user account mappings for the application. Each block supports:
  - `name` - (Required) The name of the user account mapping.
  - `description` - (Required) The description of the user account mapping.

- `properties` - (Optional) A block defining application properties. Each block supports:
  - `name` - (Required) The name of the property.
  - `value` - (Required) The value of the property.

- `sensitive_properties` - (Optional) A block defining sensitive application properties. Each block supports:
  - `name` - (Required) The name of the sensitive property.
  - `value` - (Required) The value of the sensitive property.

## Attribute Reference

The following attributes are exported:

- `catalog_app_id` - The catalog ID of the application.
- `properties` - The list of application properties.
- `sensitive_properties` - The list of sensitive application properties.
- `user_account_mappings` - The list of user account mappings.

## Import

Applications can be imported using the application ID:

```sh
terraform import britive_application.application_test_1 <application_id>
```
