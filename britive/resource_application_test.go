package britive

import (
	"fmt"
	"testing"

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

	resource "britive_application" "Snowflake standalone new" {
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

	resource "britive_application" "GCP new" {
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
	`)
}

func testAccCheckBritiveApplicationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return NewNotFoundErrorf("%s in state", n)
		}

		if rs.Primary.ID == "" {
			return NewNotFoundErrorf("ID for %s in state", n)
		}

		return nil
	}
}
