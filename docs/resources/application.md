---
subcategory: ""
layout: "britive"
page_title: "britive_application Resource - britive"
description: |-
  Manages applications for the Britive provider.
---

# britive_application Resource

This resource allows you to create and manage applications in Britive.

-> This resource is only supported for the Snowflake, Snowflake Standalone, GCP, GCP Standalone and Google Workspace applications.

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

>**Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

### GCP Application

```hcl
resource "britive_application" "new" {
  application_type = "GCP"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "GCP App 1"
  }
  properties {
    name  = "description"
    value = "Britive GCP App"
  }
  properties {
    name  = "programmaticAccess"
    value = true
  }
  properties {
    name  = "consoleAccess"
    value = true
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://console.cloud.google.com"
  }
  properties {
    name  = "orgId"
    value = "gcp1"
  }
  properties {
    name  = "gSuiteAdmin"
    value = "admin@gcp-test.com"
  }
  properties {
    name  = "projectIdForServiceAccount"
    value = "gcp-project-1"
  }
  properties {
    name  = "acsUrl"
    value = "test-gcp.com"
  }
  properties {
    name  = "audience"
    value = "admin@gcp-test.com"
  }
  properties {
    name  = "enableSso"
    value = true
  }
  properties {
    name  = "primaryDomain"
    value = "domain1"
  }
  properties {
    name  = "secondaryDomain"
    value = "domain2"
  }
  properties {
    name  = "replaceDomain"
    value = true
  }
  properties {
    name  = "scanUsersGroups"
    value = true
  }
  properties {
    name  = "scanOrganization"
    value = true
  }
  properties {
    name  = "scanProjectsOnly"
    value = true
  }
  properties {
    name  = "scanExternalUsersGroups"
    value = true
  }
  properties {
    name  = "customerId"
    value = "Cu51XXr123"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = "12345"
  }
  properties {
    name  = "gcpProjectFilter"
    value = "gcpFilter1"
  }
  properties {
    name  = "gcpProjectFilterInclusion"
    value = "gcpFilterInclusion1"
  }
  sensitive_properties {
    name  = "serviceAccountCredentials"
    value = file("${path.module}/service_key.key")
  }
```

~> This resource does not track changes made to `sensitive_properties` through the Britive console.
> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `programmaticAccess`: Programmatic Access.
> - `consoleAccess`: Console Access.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `orgId`: The Organizations Unique Identifier.
> - `gSuiteAdmin`: G Suite Admin Email.
> - `projectIdForServiceAccount`: Project ID for creating Service Accounts.
> - `acsUrl`: ACS URL.
> - `audience`: Audience.
> - `enableSso`: Enable SSO.
> - `primaryDomain`: Email Domain of Britive Users.
> - `secondaryDomain`: Primary Domain in Google Workspace.
> - `replaceDomain`: Use another domain for account mapping.
> - `scanUsersGroups`: Scan users and groups.
> - `scanOrganization`: Scan all folders and projects.
> - `scanProjectsOnly`: Scan projects only.
> - `scanExternalUsersGroups`: Scan external users and groups.
> - `customerId`: Customer ID in Google Workspace Account Settings.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.
> - `gcpProjectFilter`: Exclude projects from scan.
> - `gcpProjectFilterInclusion`: Include projects in scan.

>**Sensitive Properties:**
> - `serviceAccountCredentials`: The Service Account Credentials - Content of Private Key File as JSON String.

### GCP Standalone Application

```hcl
resource "britive_application" "new" {
  application_type = "GCP Standalone"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "GCP Standalone App 1"
  }
  properties {
    name  = "description"
    value = "Britive GCP Standalone App"
  }
  properties {
    name  = "programmaticAccess"
    value = true
  }
  properties {
    name  = "consoleAccess"
    value = true
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://gcp.test.com"
  }
  properties {
    name  = "orgId"
    value = "gcp1"
  }
  properties {
    name  = "gSuiteAdmin"
    value = "admin@gcp-test.com"
  }
  properties {
    name  = "projectIdForServiceAccount"
    value = "gcp-project-1"
  }
  properties {
    name  = "acsUrl"
    value = "test-gcp.com"
  }
  properties {
    name  = "audience"
    value = "admin@gcp-test.com"
  }
  properties {
    name  = "enableSso"
    value = true
  }
  properties {
    name  = "primaryDomain"
    value = "domain1"
  }
  properties {
    name  = "secondaryDomain"
    value = "domain2"
  }
  properties {
    name  = "replaceDomain"
    value = true
  }
  properties {
    name  = "scanUsers"
    value = true
  }
  properties {
    name  = "scanExternalUsersGroups"
    value = true
  }
  properties {
    name  = "customerId"
    value = "Cu51omer123"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = "12345"
  }
  properties {
    name  = "displayProgrammaticKeys"
    value = true
  }
  properties {
    name  = "gcpProjectFilter"
    value = "gcpFilter1"
  }
  properties {
    name  = "gcpProjectFilterInclusion"
    value = "gcpFilterInclusion1"
  }
  sensitive_properties {
    name  = "serviceAccountCredentials"
    value = file("${path.module}/service_key.key")
  }
}
```

~> This resource does not track changes made to `sensitive_properties` through the Britive console.
> **Properties:**
> - `programmaticAccess`: Programmatic Access.
> - `consoleAccess`: Console Access.
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `orgId`: The Organization's Unique Identifier.
> - `gSuiteAdmin`: G Suite Admin Email.
> - `projectIdForServiceAccount`: Project ID for creating Service Accounts.
> - `acsUrl`: ACS URL.
> - `audience`: Audience.
> - `enableSso`: Enable SSO.
> - `primaryDomain`: Email Domain of Britive Users.
> - `secondaryDomain`: Primary Domain in Google Workspace.
> - `replaceDomain`: Use another domain for account mapping.
> - `scanUsers`: Scan users.
> - `scanExternalUsersGroups`: Scan external users and groups.
> - `customerId`: Customer ID in Google Workspace Account Settings.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.
> - `displayProgrammaticKeys`: Display programmatic access keys.
> - `gcpProjectFilter`: Exclude projects from scan.
> - `gcpProjectFilterInclusion`: Include projects in scan.

>**Sensitive Properties:**
> - `serviceAccountCredentials`: The Service Account Credentials - Content of Private Key File as JSON String.

### Google Workspace Application

```hcl
resource "britive_application" "application_google_workspace" {
    application_type = "Google Workspace"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "YS TF Google Workspace"
    }
    properties {
      name = "description"
      value = "YS TF Google Workspace Description"
    }
    properties {
      name = "appAccessMethod_static_loginUrl"
      value = "https://console.cloud.google.com"
    }
    properties {
      name = "provisionUserGw"
      value = "true"
    }
    properties {
      name = "gSuiteAdmin"
      value = "admin@google-test.com"
    }
    properties {
      name = "acsUrl"
      value = "test-google.com"
    }
    properties {
      name = "audience"
      value = "admin@google-test.com"
    }
    properties {
      name = "enableSso"
      value = true
    }
    properties {
      name = "primaryDomain"
      value = "domain1"
    }
    properties {
      name = "secondaryDomain"
      value = "domain2"
    }
    properties {
      name = "replaceDomain"
      value = true
    }
    properties {
      name = "scanRoles"
      value = true
    }
    properties {
      name = "scanGroups"
      value = true
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = "12345
    }
    sensitive_properties {
      name = "serviceAccountCredentials"
      value = file("${path.module}/service_key.json")
    }
}
```

~> This resource does not track changes made to `sensitive_properties` through the Britive console.
> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `provisionUserGw`: Create user account for super admin role.
> - `gSuiteAdmin`: Google Workspace admin email.
> - `acsUrl`: ACS URL.
> - `audience`: Audience.
> - `enableSso`: Enable SSO.
> - `primaryDomain`: Email Domain of Britive Users.
> - `secondaryDomain`: Primary Domain in Google Workspace.
> - `replaceDomain`: Use another domain for account mapping.
> - `scanRoles`: Scan roles.
> - `scanGroups`: Scan groups.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

>**Sensitive Properties:**
> - `serviceAccountCredentials`: The Service Account Credentials - Content of Private Key File as JSON String.

## Argument Reference

The following arguments are supported:

* `application_type` - (Required) The type of the application. Supported types are `Snowflake`, `Snowflake Standalone`, `GCP`, `GCP Standalone` and `Google Workspace`.

* `version` - (Optional) The version of the application resource.  
  If specified, it must match a supported version for the selected `application_type`.  
  If omitted, the provider will use the latest available version for selected `application_type`.  
  See the example usage for each application type above for valid version values.

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

* `id` - The unique identifier for the application resource.

* `catalog_app_id` - The identifier of the application type in the Britive catalog.

* `entity_root_environment_group_id` - The root environment group ID (only for Snowflake Standalone applications).

## Import

Applications can be imported using one of the following formats:

```sh
terraform import britive_application.new apps/{{application_id}}
terraform import britive_application.new {{application_id}}
```
  
->During the import process, only properties with values explicitly set or different from their default values will be imported. This avoids overwriting default configurations and ensures only customized settings are preserved in the Terraform state.

## Deleting Properties

When a property is deleted from the configuration, its value will revert to the default based on its data type:
- string: "" (empty string)
- boolean: False

**EXCEPTIONS:** Some applications require certain properties to retain specific default values, even when removed from the configuration. These exceptions are outlined below.
### GCP
```sh
{
    'consoleAccess': True,
    'displayName': 'GCP',
    'appAccessMethod_static_loginUrl': 'https://console.cloud.google.com',
    'scanUsersGroups': True,
    'maxSessionDurationForProfiles': '604800' 
}
```

### GCP Standalone
```sh
{
    'consoleAccess': True,
    'displayName': 'GCP Standalone',
    'appAccessMethod_static_loginUrl': 'https://console.cloud.google.com',
    'maxSessionDurationForProfiles': '604800' 
}
```

### Google Workspace
```sh
{
    'displayName': 'Google Workspace',
    'appAccessMethod_static_loginUrl': 'https://admin.google.com',
    'maxSessionDurationForProfiles': '604800',
    'scanRoles': True,
    'scanGroups': True 
}
```

### Snowflake
``` sh
{
    'appAccessMethod_static_loginUrl': 'https://{accountId}.snowflakecomputing.com/',
    'displayName': 'Snowflake',
    'maxSessionDurationForProfiles': '604800' 
}
```

### Snowflake Standalone
```sh
{
    'displayName': 'Snowflake Standalone',
    'description': 'Snowflake app for standalone instances',
    'maxSessionDurationForProfiles': '604800' 
}
```
