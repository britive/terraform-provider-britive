# britive_application Resource

This resource allows you to create and manage applications in Britive.

-> This resource is only supported for Snowflake and Snowflake Standalone applications.

## Example Usage

### Snowflake Application

```hcl
resource "britive_application" "new" {
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
    value = "QXZ72233xx"
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://snowflake.test.com"
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
  sensitive_properties {
    name  = "privateKey"
    value = file("${path.module}/private_key.key")
  }
  sensitive_properties {
    name  = "publicKey"
    value = file("${path.module}/public_key.key")
  }
  sensitive_properties {
    name  = "privateKeyPassword"
    value = "Password"
  }
}
```

-> The `properties` and `sensitive_properties` in the above example are mandatory for creating a valid Snowflake application.  

~> This resource does not track changes made to `sensitive_properties` through the Britive console.
>**Properties:**
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

>**Sensitive Properties:**
> - `privateKeyPassword`: Password of the Private Key.
> - `publicKey`: Public Key configured for the user.
> - `privateKey`: Private Key configured for the user.

### Snowflake Standalone Application

```hcl
resource "britive_application" "new" {
    application_type = "Snowflake Standalone"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "Snowflake App 2"
    }
    properties {
      name = "description"
      value = "Britive Snowflake Standalone App"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
}
```

-> The `properties` in the above Snowflake Standalone example are mandatory for creating a valid Snowflake Standalone application.
>**Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

## Argument Reference

The following arguments are supported:

* `application_type` - (Required) The type of the application. Supported types are `Snowflake` and `Snowflake Standalone`.

* `user_account_mappings` - (Optional) A block defining user account mappings for the application. Each block supports:
  - `name` - (Required) The name of the user account mapping.
  - `description` - (Required) The description of the user account mapping.

* `properties` - (Optional) A block defining application properties. Each block supports:
  - `name` - (Required) The name of the property.
  - `value` - (Required) The value of the property.

* `sensitive_properties` - (Optional) A block defining sensitive application properties. Each block supports:
  - `name` - (Required) The name of the sensitive property.
  - `value` - (Required) The value of the sensitive property.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - An identifier for the application.

* `catalog_app_id` - The id of the application type.

* `entity_root_environment_group_id` - The root environment group ID for the Snowflake Standalone application.

## Import

You can import an application using any of these accepted formats:

```sh
terraform import britive_application.new apps/{{application_id}}
terraform import britive_application.new {{application_id}}
```
