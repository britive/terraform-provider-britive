# britive_application Resource

> **Note:** The resource `britive_application` resource currently supports only Snowflake and Snowflake Standalone application types.

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
  sensitive_properties {
    name  = "publicKey"
    value = file("${path.module}/rsa_key_public.pub")
  }
  sensitive_properties {
    name  = "privateKeyPassword"
    value = "Password"
  }
}
```

> **Note:** Following `properties` and `sensitive_properties` in the above example are mandatory for creating a valid Snowflake application.  
> **Properties** include:
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `loginNameForAccountMapping`: Use login name for account mapping.
> - `accountId`: Account ID.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `username`: Username of the User in Snowflake.
> - `role`: Custom Role assigned to the user.
> - `snowflakeSchemaScanFilter`: Skip collecting schema level privileges.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.
> - `copyAppToEnvProps`: Use same user, role and keys for all accounts.

**Sensitive Properties** include:
> - `privateKeyPassword`: Password of the Private Key.
> - `publicKey`: Public Key configured for the user.
> - `privateKey`: Private Key configured for the user.

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
}
```

> **Note:** Following `properties` in the above Snowflake Standalone example are mandatory for creating a valid Snowflake Standalone application.  
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

## Argument Reference

### Required

- `application_type` - (Required) The type of the application. Supported types are `Snowflake` and `Snowflake Standalone`.

### Optional
- `entity_root_environment_group_id` - (Computed) Britive application root environment ID for Snowflake Standalone applications.
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
terraform import britive_application.application_test_1 application/<application_id>
terraform import britive_application.application_test_1 applications/<application_id>
```
