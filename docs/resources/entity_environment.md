---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_entity_environment Resource - britive"
description: |-
  Manages entity environment for the Britive provider.
---

# britive_entity_environment Resource

This resource allows you to create and configure an application entity of the type "Environment".

-> This resource is only supported for Snowflake Standalone, AWS Standalone and Okta applications.

-> For applications created from the Britive console, the first entity must be created through the console so that this resource has a parent under which it can be created. This step is not needed for applications created via the Britive Terraform provider plugin.

## Example Usage
### Snowflake Standalone Entity Environment

```hcl
resource "britive_application" "new_snowflake_standalone" {
    application_type = "Snowflake Standalone"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "New Snowflake Standalone"
    }
    properties {
      name = "description"
      value = "New Snowflake Standalone Description"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = "1000"
    }
}

resource "britive_entity_environment" "new" {
  application_id  = britive_application.new_snowflake_standalone.id
  parent_group_id = britive_application.new_snowflake_standalone.entity_root_environment_group_id

  properties {
    name  = "displayName"
    value = "My Snowflake Environment"
  }
  properties {
    name  = "description"
    value = "My Snowflake Environment Description"
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
    value = "QXZ72233xx"
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
-> The `properties` and `sensitive_properties` in the above example are mandatory for creating a valid entity of type environment.  

~> This resource does not track changes made to `sensitive_properties` through the Britive console.

>**Properties:**
> - `displayName`: Environment Name.
> - `description`: Environment Description.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `username`: Username of the User in Snowflake.
> - `accountId`: Account ID.
> - `role`: Custom Role assigned to the user.
> - `loginNameForAccountMapping`: Use login name for account mapping.
> - `snowflakeSchemaScanFilter`: Skip collecting schema level privileges.

>**Sensitive Properties:**
> - `privateKeyPassword`: Password of the Private Key.
> - `publicKey`: Public Key configured for the user.
> - `privateKey`: Private Key configured for the user.

### AWS Standalone Entity Environment

```hcl
resource "britive_entity_environment" "aws_env_1" {
  application_id = "jhaidy8q7ywqSyxxxxxx"
  parent_group_id = "aksd67euhexxxx7we98exx"
  properties {
    name = "displayName"
    value = "My AWS Environment"
  }
  properties {
    name = "description"
    value = "My AWS Environment Description"
  }
  properties {
    name = "showAwsAccountNumber"
    value = true
  }
  properties {
    name = "accountId"
    value = "897xxxx5476xxxx"
  }
  properties {
    name = "supportsInvalidation"
    value = false
  }
}
```

-> `properties` in the above example are mandatory for creating a valid entity of type environment. 

>**Properties:**
> - `displayName`: Environment Name.
> - `description`: Environment Description.
> - `showAwsAccountNumber`: Display AWS account number.
> - `accountId`: Account ID.
> - `supportsInvalidation`: Indicates if invalidation is supported.

### Okta Entity Environment

```hcl
resource "britive_entity_environment" "okta_env_1" {
  application_id = "aydhsadjxxxdw8dhxxxx"
  parent_group_id = "auysgxxxxxuaysux"
  properties {
    name = "displayName"
    value = "My Okta Environment"
  }
  properties {
    name = "description"
    value = "My Okta Environment Description"
  }
  properties {
    name = "appAccessMethod_static_loginUrl"
    value = "https://test.okta.com/"
  }
  properties {
    name = "userFilter"
    value = "user@test.com"
  }
  properties {
    name = "groupFilter"
    value = "testGroup"
  }
  properties {
    name = "scanRoles"
    value = true
  }
  sensitive_properties {
    name = "apiToken"
    value = "kjhduyad76xxxxoadiye7eyiahsxxxioad9qwe98xxxx"
  }
}
```

-> `properties` in the above example are mandatory for creating a valid entity of type environment. 

>**Properties:**
> - `displayName`: Environment Name.
> - `description`: Environment Description.
> - `userFilter`: Filter for specific users.
> - `groupFilter`: Filter for specific groups.
> - `scanRoles`: Enable scanning of roles.

## Argument Reference

The following arguments are supported:

* `application_id` - (Required, ForceNew) The identity of the Britive application.
* `parent_group_id` - (Required, ForceNew)  The identity of the parent group under which the environment entity will be created.
* `properties` - (Optional) A block defining environment properties. Each block supports:
  - `name` - (Required) The name of the property.
  - `value` - (Required) The value of the property.
* `sensitive_properties` - (Optional) A block defining sensitive environment properties. Each block supports:
  - `name` - (Required) The name of the sensitive property.
  - `value` - (Required) The value of the sensitive property.

-> Refer to the `environment_group_ids_names` attribute of the `britive_application` data source to get the set of group IDs and names for an application.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `entity_id` - An identifier of the environment entity.
* `id` - An identifier of the resource with format `apps/{{application_id}}/root-environment-group/environments/{{entity_id}}`

## Import

You can import an environment entity using any of these accepted formats:

```sh
terraform import britive_entity_environment.new apps/{{application_id}}/root-environment-group/environments/{{entity_id}}
terraform import britive_entity_environment.new {{application_id}}/environments/{{entity_id}}
```
