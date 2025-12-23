package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveApplication(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationExists("britive_application.snowflake_new"),
					testAccCheckBritiveApplicationExists("britive_application.snowflake_standalone_new"),
					testAccCheckBritiveApplicationExists("britive_application.gcp_new"),
					testAccCheckBritiveApplicationExists("britive_application.gcp_standlone_new"),
					testAccCheckBritiveApplicationExists("britive_application.gcp_wif_new"),
					testAccCheckBritiveApplicationExists("britive_application.google_workspace_new"),
					testAccCheckBritiveApplicationExists("britive_application.aws_new"),
					testAccCheckBritiveApplicationExists("britive_application.aws_standalone_new"),
					testAccCheckBritiveApplicationExists("britive_application.azure_new"),
					testAccCheckBritiveApplicationExists("britive_application.okta_new"),
				),
			},
		},
	})
}

func testAccCheckBritiveApplicationConfig() string {
	return fmt.Sprint(`
	resource "britive_application" "snowflake_new" {
	application_type = "Snowflake"
	version = "1.0"
	user_account_mappings {
		name        = "Mobile"
		description = "Mobile"
	}
	properties {
		name  = "displayName"
		value = "AT - Snowflake App"
	}
	properties {
		name  = "description"
		value = "AT - Britive Snowflake App"
	}
	properties {
		name  = "loginNameForAccountMapping"
		value = true
	}
	properties {
		name  = "accountId"
		value = "QXZ7XX33xx"
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
		value = "<Private Key>"
	}
	sensitive_properties {
		name  = "publicKey"
		value = "<Public Key>"
	}
	sensitive_properties {
		name  = "privateKeyPassword"
		value = "<Private Key Password>"
	}
	}

	resource "britive_application" "snowflake_standalone_new" {
    application_type = "Snowflake Standalone"
    version = "1.0"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - Snowflake Standalone App"
    }
    properties {
      name = "description"
      value = "AT - Britive Snowflake Standalone App"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1500
    }
	}

	resource "britive_application" "gcp_new" {
	application_type = "GCP"
	version = "2.0"
	user_account_mappings {
		name        = "Mobile"
		description = "Mobile"
	}
	properties {
		name  = "displayName"
		value = "AT - GCP App"
	}
	properties {
		name  = "description"
		value = "AT - Britive GCP App"
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
		value = "https:/console-gcp.com"
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
		value = "gcp-project"
	}
	properties {
		name  = "acsUrl"
		value = "test-gcp.com"
	}
	properties {
		name  = "audience"
		value = "admin-gcp@test.com"
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
		value = "Cu5t0merId"
	}
	properties {
		name  = "maxSessionDurationForProfiles"
		value = "1500"
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
		value = "<Service Key>"
	}
	}

	resource "britive_application" "gcp_standlone_new" {
	application_type = "GCP Standalone"
	version = "1.0"
	user_account_mappings {
		name        = "Mobile"
		description = "Mobile"
	}
	properties {
		name  = "displayName"
		value = "AT - GCP Standalone App"
	}
	properties {
		name  = "description"
		value = "AT - Britive GCP Standalone App"
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
		value = "admin-gcp@test.com"
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
		value = "admin-gcp@test.com"
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
		value = "Cu51omerId"
	}
	properties {
		name  = "maxSessionDurationForProfiles"
		value = "1500"
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
		value = "<Service-Key>"
	}
	}

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
		name  = "scanOrganization"
		value = false
	}
	properties {
		name  = "scanProjectsOnly"
		value = true
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

	resource "britive_application" "google_workspace_new" {
    application_type = "Google Workspace"
    version = "1.0"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - Google Workspace App"
    }
    properties {
      name = "description"
      value = "AT - Google Workspace App Description"
    }
    properties {
      name = "appAccessMethod_static_loginUrl"
      value = "https://console-google.com"
    }
    properties {
      name = "provisionUserGw"
      value = "true"
    }
    properties {
      name = "gSuiteAdmin"
      value = "admin-google@test.com"
    }
    properties {
      name = "acsUrl"
      value = "test-google.com"
    }
    properties {
      name = "audience"
      value = "admin-google@test.com"
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
      value = "1500"
    }
    sensitive_properties {
      name = "serviceAccountCredentials"
      value = "<Service-key>"
    }
	}

	resource "britive_application" "aws_new" {
    application_type = "AWS"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - AWS APP"
    }
    properties {
      name = "description"
      value = "AT - AWS APP DESC"
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
      value = "BritiveProvider"
    }
    properties {
      name = "roleName"
      value = "britive-integration-role"
    }
    properties {
      name = "accountId"
      value = "<AccountID>"
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

	resource "britive_application" "aws_standalone_new" {
    application_type = "aws standalone"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - AWS STANDALONE APP"
    }
    properties {
      name = "description"
      value = "AT - AWS STANDALONE APP DESC"
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
      value = "BritiveProvider"
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

	resource "britive_application" "azure_new" {
    application_type = "azure"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - AZURE APP"
    }
    properties {
      name = "description"
      value = "AT - AZURE APP DESC"
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
    sensitive_properties {
      name = "clientSecret"
      value = "<Client-Secret>"
    }
    properties {
      name = "userFilter"
      value = ""
    }
    properties {
      name = "groupFilter"
      value = ""
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
	}

	resource "britive_application" "okta_new" {
    application_type = "okta"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - OKTA APP"
    }
    properties {
      name = "description"
      value = "AT - OKTA APP DESC"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
	}
	`)
}

func testAccCheckBritiveApplicationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return errs.NewNotFoundErrorf("%s in state", n)
		}

		if rs.Primary.ID == "" {
			return errs.NewNotFoundErrorf("ID for %s in state", n)
		}

		return nil
	}
}
