---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_application Resource - Britive"
description: |-
  Manages applications for the Britive provider.
---

# britive_application Resource

This resource allows you to create and manage applications in Britive.

-> This resource is supported only on Snowflake, Snowflake Standalone, GCP, GCP Standalone, GCP WIF, Google Workspace, AWS, AWS Standalone, Azure, Azure WIF, Okta, Britive, Oracle WIF, and Kubernetes applications.

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

### GCP WIF Application

```hcl
resource "britive_application" "gcp_wif_new" {
  application_type = "gcp wif"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "GCP WIF Application"
  }
  properties {
    name  = "description"
    value = "GCP WIF Application Description"
  }
  properties {
    name  = "programmaticAccess"
    value = false
  }
  properties {
    name  = "consoleAccess"
    value = true
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://gcp_wif.test.com"
  }
  properties {
    name  = "orgId"
    value = "test_gcp_wif1"
  }
  properties {
    name  = "britiveIssuerUrl"
    value = "https://gcp_wif_test.com/oauth2"
  }
  properties {
    name  = "wifPool"
    value = "test-gcp-wif-pool"
  }
  properties {
    name  = "wifProvider"
    value = "testgcpwifprovider"
  }
  properties {
    name  = "wifSA"
    value = "test@gcp.wif.com"
  }
  properties {
    name  = "projectNumberForWifSA"
    value = "test_gcp_wif_project_number"
  }
  properties {
    name  = "projectIdForServiceAccount"
    value = "test_gcp_wif_account"
  }
  properties {
    name  = "acsUrl"
    value = "test_gcp_wif"
  }
  properties {
    name  = "audience"
    value = "gcp_test@gcpwif.net"
  }
  properties {
    name  = "enableSso"
    value = false
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
    value = false
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = 10000
  }
  properties {
    name  = "displayProgrammaticKeys"
    value = false
  }
  properties {
    name  = "gcpProjectFilter"
    value = false
  }
  properties {
    name  = "gcpProjectFilterInclusion"
    value = false
  }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `programmaticAccess`: Programmatic Access.
> - `consoleAccess`: Console Access.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `orgId`: The Organizations Unique Identifier.
> - `britiveIssueUrl`: Britive Issuer URL.
> - `wifPool`: Workload Identity Pool ID.
> - `wifProvider`: Workload Identity Provider ID.
> - `wifSA`: Connected Service Account Email.
> - `projectNumberForWifSA`: Project Number For Connected Service Account.
> - `projectIdForServiceAccount`: Project ID for creating Service Accounts.
> - `acsUrl`: ACS URL.
> - `audience`: Audience.
> - `enableSso`: Enable SSO.
> - `primaryDomain`: Email Domain of Britive Users.
> - `secondaryDomain`: Primary Domain in Google Workspace.
> - `replaceDomain`: Use another domain for Account Mapping.
> - `scanExternalUsersGroups`: Scan External Users and Groups.
> - `maxSessionDurationForProfiles`: Maximum Session Duration for Profiles.
> - `gcpProjectFilter`: Exclude Projects from Scan.
> - `gcpProjectFilterInclusion`: Include projects in Scan.
>
> **Note:** `scanOrganization` and `scanProjectsOnly` are no longer supported for `GCP WIF` applications and must not be set in the `properties` block.

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
      value = "TF Google Workspace"
    }
    properties {
      name = "description"
      value = "TF Google Workspace Description"
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
      value = "12345"
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

### AWS Application

```hcl
resource "britive_application" "aws_1" {
    application_type = "AWS"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AWS Application App"
    }
    properties {
      name = "description"
      value = "AWS Application App Desc"
    }
    properties {
      name = "showAwsAccountNumber"
      value = true
    }
    properties {
      name = "sessionDuration"
      value = 2
    }
    properties {
      name = "identityProvider"
      value = "Provider"
    }
    properties {
      name = "roleName"
      value = "roleName"
    }
    properties {
      name = "accountId"
      value = "<Account-Id>"
    }
    properties {
      name = "region"
      value = "us-east-1"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
    properties {
      name = "supportsInvalidationGlobal"
      value = true
    }
    properties {
      name = "allowCopyingConsoleUrl"
      value = false
    }
    properties {
      name = "displayProgrammaticKeys"
      value = false
    }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `showAwsAccountNumber`: Show AWS account number.
> - `sessionDuration`: AWS session duration.
> - `identityProvider`: Identity Provider.
> - `roleName`: Role name.
> - `accountId`: Management Account ID.
> - `region`: Region.
> - `supportsInvalidationGlobal`: Support invalidation global.
> - `allowCopyingConsoleUrl`: Allow copying console url.
> - `displayProgrammaticKeys`: Display progragmmatic keys.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

### AWS Standalone Application

```hcl
resource "britive_application" "aws_standalone_1" {
    application_type = "aws standalone"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AWS Standalone App"
    }
    properties {
      name = "description"
      value = "AWS Standalone App Desc"
    }
    properties {
      name = "showAwsAccountNumber"
      value = false
    }
    properties {
      name = "allowCopyingConsoleUrl"
      value = false
    }
    properties {
      name = "displayProgrammaticKeys"
      value = false
    }
    properties {
      name = "identityProvider"
      value = "Provider"
    }
    properties {
      name = "sessionDuration"
      value = 1000
    }
    properties {
      name = "region"
      value = "us-east-1"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `showAwsAccountNumber`: Show AWS account number.
> - `sessionDuration`: AWS session duration.
> - `identityProvider`: Identity Provider.
> - `region`: Region.
> - `allowCopyingConsoleUrl`: Allow copying console url.
> - `displayProgrammaticKeys`: Display programmatic keys.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

### Azure Application

```hcl
resource "britive_application" "azure_1" {
    application_type = "azure"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "Azure App"
    }
    properties {
      name = "description"
      value = "Azure App Desc"
    }
    properties {
      name = "programmaticAccess"
      value = true
    }
    properties {
      name = "consoleAccess"
      value = true
    }
    properties {
      name = "appAccessMethod_static_loginUrl"
      value = "https://azure.test.com"
    }
    properties {
      name = "tenantId"
      value = "<Tenant-Id>"
    }
    properties {
      name = "clientId"
      value = "<Client-Id>"
    }
    properties {
      name = "userFilter"
      value = "user"
    }
    properties {
      name = "groupFilter"
      value = "group"
    }
    properties {
      name = "scanMethod"
      value = "collectUsersGroups"
    }
    properties {
      name = "scanMgmtGroupsAndSubscriptions"
      value = false
    }
    properties {
      name = "scanSubscriptionsOnly"
      value = false
    }
    properties {
      name = "scanResources"
      value = false
    }
    properties {
      name = "scanGroupsMemberships"
      value = false
    }
    properties {
      name = "scanServicePrincipals"
      value = false
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
    properties {
      name = "displayProgrammaticKeys"
      value = false
    }
    sensitive_properties {
      name = "clientSecret"
      value = "<Client-Secret>"
    }
}
```

~> This resource does not track changes made to `sensitive_properties` through the Britive console.
> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `programmaticAccess`: Programmatic Access.
> - `consoleAccess`: Console Access.
> - `appAccessMethod_static_loginUrl`: Login url.
> - `tenantId`: Tenant ID.
> - `clientId`: Client ID.
> - `userFilter`: User filter.
> - `groupFilter`: Group filter.
> - `scanMethod`: Scan method — one of `collectUsersGroups`, `collectGroupsMemberships`, or `collectUsersMembership`.
> - `scanMgmtGroupsAndSubscriptions`: Scan management group and subscription.
> - `scanSubscriptionsOnly`: Scan subscription Only.
> - `scanResources`: Scan resources.
> - `scanGroupsMemberships`: Scan group membership.
> - `scanServicePrincipals`: Scan service principals.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.
> - `displayProgrammaticKeys`: Display programmatic keys.

>**Sensitive Properties:**
> - `clientSecret`: Client secret.

### Azure WIF Application

```hcl
resource "britive_application" "azure_wif" {
  application_type = "Azure wif"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "Azure WIF"
  }
  properties {
    name  = "description"
    value = "Azure WIF Desc"
  }
  properties {
    name  = "programmaticAccess"
    value = false
  }
  properties {
    name  = "consoleAccess"
    value = true
  }
  properties {
    name  = "appAccessMethod_static_loginUrl"
    value = "https://portal.azure.com"
  }
  properties {
    name  = "britiveIssuerUrl"
    value = "https://test.britive-test-app.com/api/auth/sso/oauth2"
  }
  properties {
    name  = "tenantId"
    value = "<Azure-Tenant-ID>"
  }
  properties {
    name  = "clientId"
    value = "<Azure-Client-ID>"
  }
  properties {
    name  = "azureWifAudience"
    value = "api://AzureADTokenExchange"
  }
  properties {
    name  = "userFilter"
    value = ""
  }
  properties {
    name  = "groupFilter"
    value = ""
  }
  properties {
    name  = "scanMethod"
    value = "collectUsersGroups"
  }
  properties {
    name  = "scanMgmtGroupsAndSubscriptions"
    value = false
  }
  properties {
    name  = "scanSubscriptionsOnly"
    value = false
  }
  properties {
    name  = "scanResources"
    value = false
  }
  properties {
    name  = "scanGroupsMemberships"
    value = true
  }
  properties {
    name  = "scanServicePrincipals"
    value = false
  }
  properties {
    name  = "scanAiIdentities"
    value = false
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = "604800"
  }
  properties {
    name  = "displayProgrammaticKeys"
    value = false
  }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `programmaticAccess`: Programmatic Access.
> - `consoleAccess`: Console Access.
> - `appAccessMethod_static_loginUrl`: Login URL.
> - `britiveIssuerUrl`: Britive Issuer URL (used as the OIDC issuer for Workload Identity Federation).
> - `tenantId`: Azure Tenant ID.
> - `clientId`: Azure Application (Client) ID.
> - `azureWifAudience`: Federated credential audience value configured in Azure (e.g. `api://AzureADTokenExchange`).
> - `userFilter`: Filter for users.
> - `groupFilter`: Filter for groups.
> - `scanMethod`: Scan method — one of `collectUsersGroups`, `collectGroupsMemberships`, or `collectUsersMembership`.
> - `scanMgmtGroupsAndSubscriptions`: Scan management groups and subscriptions.
> - `scanSubscriptionsOnly`: Scan subscriptions only.
> - `scanResources`: Scan Azure resource groups and resources.
> - `scanGroupsMemberships`: Scan user group memberships.
> - `scanServicePrincipals`: Scan Azure service principals.
> - `scanAiIdentities`: Scan AI identities.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.
> - `displayProgrammaticKeys`: Display programmatic access keys.

### Oracle WIF Application

```hcl
resource "britive_application" "oracle_wif" {
  application_type = "Oracle wif"
  user_account_mappings {
    name        = "Mobile"
    description = "Mobile"
  }
  properties {
    name  = "displayName"
    value = "Oracle WIF App"
  }
  properties {
    name  = "description"
    value = "YGS Oracle WIF Desc"
  }
  properties {
    name  = "tenancy"
    value = "ocid.tenancy.oc1.nnnnnbbbbbtttttjgjgwdwcwcwwcee"
  }
  properties {
    name  = "tenantName"
    value = "Britive"
  }
  properties {
    name  = "britiveIssuerUrl"
    value = "https://test.britive-test-app.com/api/auth/sso/oauth2"
  }
  properties {
    name  = "clientId"
    value = "7hbhjb7bvg2hbb3jjbbg4kjknn1hhh"
  }
  properties {
    name  = "region"
    value = "us-phoenix-1"
  }
  properties {
    name  = "domainUrl"
    value = "https://idcs-uunb75fcc322ghbbb322hggg1vg.identity.oraclecloud.com"
  }
  properties {
    name  = "maxSessionDurationForProfiles"
    value = "604800"
  }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `tenancy`: OCI OCID of the tenancy.
> - `tenantName`: Name of the Britive tenant.
> - `britiveIssuerUrl`: Britive Issuer URL (used as the OIDC issuer for Workload Identity Federation).
> - `clientId`: Client ID of the OCI confidential application.
> - `region`: Oracle Cloud region (e.g. `us-phoenix-1`).
> - `domainUrl`: Oracle Identity Domain URL.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

### Okta Application

```hcl
resource "britive_application" "okta_1" {
    application_type = "okta"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "Okta App"
    }
    properties {
      name = "description"
      value = "Okta App Desc"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

### Britive Application

```hcl
resource "britive_application" "britive_1" {
    application_type = "Britive"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "Britive App"
    }
    properties {
      name = "description"
      value = "Britive App Desc"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 604800
    }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.

### Kubernetes Application

```hcl
resource "britive_application" "kubernetes_1" {
    application_type = "Kubernetes"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "Kubernetes App"
    }
    properties {
      name = "description"
      value = "Kubernetes App Desc"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 43200
    }
    properties {
      name = "displayProgrammaticKeys"
      value = true
    }
}
```

> **Properties:**
> - `displayName`: Application Name.
> - `description`: Application Description.
> - `maxSessionDurationForProfiles`: Maximum session duration for profiles.
> - `displayProgrammaticKeys`: Display programmatic access keys.

Once created, environment groups and environments (Kubernetes clusters) for this application can be managed with the [`britive_entity_group`](entity_group.md) and [`britive_entity_environment`](entity_environment.md) resources, using `entity_root_environment_group_id` as the root `parent_id`/`parent_group_id`.

## Argument Reference

The following arguments are supported:

* `application_type` - (Required) The type of the application. Supported types are `Snowflake`, `Snowflake Standalone`, `GCP`, `GCP Standalone`, `GCP WIF`, `Google Workspace`, `AWS`, `AWS Standalone`, `Azure`, `Azure WIF`, `Okta`, `Britive`, `Oracle WIF` and `Kubernetes`.

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

* `scan_enabled` - (Optional) Whether scheduled scanning is enabled for this application. Left **unmanaged** when omitted from config - see `scan_enabled` Semantics below.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - The unique identifier for the application resource.

* `catalog_app_id` - The identifier of the application type in the Britive catalog.

* `entity_root_environment_group_id` - The root environment group ID (only for AWS Standalone, Okta, Snowflake Standalone, Britive, and Kubernetes applications).

* `task_service_id` - The ID of this application's scan task service. Registered automatically when the application is created - not independently managed by this provider. Used internally by `scan_enabled` and by [`britive_application_scan_schedule`](application_scan_schedule.md).

## Import

Applications can be imported using one of the following formats:

```sh
terraform import britive_application.new apps/{{application_id}}
terraform import britive_application.new {{application_id}}
```
  
-> During the import process, only properties with values explicitly set or different from their default values will be imported. This avoids overwriting default configurations and ensures only customized settings are preserved in the Terraform state.

## `scan_enabled` Semantics

`scan_enabled` is unlike this resource's other attributes in one respect: **omitting it from
config is not the same as setting it to `false`**. It is only ever acted on when present in
config at all:

* Omitted entirely - the provider never touches it. The exported value simply reflects
  whatever the application's actual scanning status already is. This is deliberate so that
  adopting this provider version, or managing an existing application that already has
  scanning turned on some other way, never flips anything.
* Set to `true` or `false` - the provider actively updates it to match.
* Previously set, then removed from config - treated as an explicit request to turn it off
  (not "stop managing it and leave it as-is").

Setting `scan_enabled` to any value for an `application_type` that does not support
scheduled scanning at all (currently only `Kubernetes`) fails `terraform apply` with an
"Unsupported Application Scan Schedule Configuration" error - see
[`britive_application_scan_schedule`](application_scan_schedule.md#application-type-support)
for the full per-type support matrix (also covering that resource's own `scope`/`org_scan`
arguments).

The backend registers an application's scan task service asynchronously after the
application itself is created, so on rare occasions `terraform apply` can fail on a brand
new application with an "Error Reading Application Scan Task Service" error if the lookup
runs before the task service exists yet. When that happens, the application itself is not
rolled back or deleted - it and everything else configured in that same apply (properties,
user account mappings, etc.) are still saved to state, with `task_service_id` and
`scan_enabled` left at their not-yet-resolved defaults (empty and `false`). Simply running
`terraform apply` again once the backend has caught up resolves both automatically via this
resource's normal refresh - there's no need to `terraform destroy`/recreate the application.

The application's scan task service (what `scan_enabled` actually toggles) is registered
automatically as part of application creation, so `scan_enabled = true` can be set in the
very same apply that creates both the application and its first
[`britive_application_scan_schedule`](application_scan_schedule.md):

```hcl
resource "britive_application" "example" {
  application_type = "AWS Standalone"
  scan_enabled     = true
}

resource "britive_application_scan_schedule" "example" {
  application_id     = britive_application.example.id
  name               = "daily-scan"
  frequency_type     = "Daily"
  start_time         = "06:30"
  org_scan           = true
}
```

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

### GCP WIF
```sh
{
    'displayName': 'GCP',
    'consoleAccess': True,
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

### AWS
```sh
{
    'displayName': 'AWS',
    'identityProvider': 'Britive',
    'roleName': 'roleName',
    'sessionDuration': '1',
    'appAccessMethod': 'appAccessMethod',
    'maxSessionDurationForProfiles': '43200',
    'allowCopyingConsoleUrl': 'true',
    'displayProgrammaticKeys': 'true'
}
```

### AWS Standalone
```sh
{
    'displayName': 'AWS Standalone',
    'appAccessMethod': 'appAccessMethod',
    'url': 'https://aws.test.com',
    'roleName': 'britive-integration-role',
    'sessionDuration': '1',
    'identityProvider': 'Provider',
    'maxSessionDurationForProfiles': '43200',
    'allowCopyingConsoleUrl': 'true',
    'displayProgrammaticKeys': 'true'
}
```

### Azure
```sh
{
    'displayName': 'Azure',
    'consoleAccess': 'true',
    'appAccessMethod_static_loginUrl': 'https://azure.test.com',
    'clientId': '<Client-ID>',
    'clientSecret': '<Client-Secret>',
    'scanMethod': 'collectUsersGroups',
    'scanGroupsMemberships': 'true',
    'maxSessionDurationForProfiles': '604800'
}
```

### Azure WIF
```sh
{
    'displayName': 'Azure WIF',
    'consoleAccess': 'true',
    'programmaticAccess': 'false',
    'appAccessMethod_static_loginUrl': 'https://portal.azure.com',
    'britiveIssuerUrl': 'https://<britive_tenant_url>/api/auth/sso/oauth2',
    'azureWifAudience': 'api://AzureADTokenExchange',
    'scanMethod': 'collectUsersGroups',
    'scanGroupsMemberships': 'true',
    'scanServicePrincipals': 'false',
    'scanAiIdentities': 'false',
    'displayProgrammaticKeys': 'false',
    'maxSessionDurationForProfiles': '604800'
}
```

### Oracle WIF
```sh
{
    'displayName': 'Oracle WIF',
    'britiveIssuerUrl': 'https://<britive_tenant_url>/api/auth/sso/oauth2'
    'maxSessionDurationForProfiles': '604800'
}
```

### Okta
```sh
{
   'displayName': 'Okta',
   'maxSessionDurationForProfiles': '604800'
}
```