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
		value = "jgjgjg"
	}
	sensitive_properties {
		name  = "publicKey"
		value = "khkghkg"
	}
	sensitive_properties {
		name  = "privateKeyPassword"
		value = "Password"
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
