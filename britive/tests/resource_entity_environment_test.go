package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveEntityEnvironment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveEntityEnvironmentConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveEntityEnvironmentExists("britive_application.snowflake_standalone_new"),
					testAccCheckBritiveEntityEnvironmentExists("britive_entity_environment.entity_environment_new"),
				),
			},
		},
	})
}

func testAccCheckBritiveEntityEnvironmentConfig() string {
	return fmt.Sprintf(`
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

	resource "britive_entity_environment" "entity_environment_new" {
	application_id = britive_application.snowflake_standalone_new.id
	parent_group_id = britive_application.snowflake_standalone_new.entity_root_environment_group_id
	properties {
		name = "displayName"
		value = "AT - Snowflake Env"
	}
	properties {
		name = "description"
		value = "AT - Snowflake Env Desc"
	}
	properties {
		name = "loginNameForAccountMapping"
		value = false
	}
	properties {
		name = "snowflakeSchemaScanFilter"
		value = true
	}
	properties {
		name = "accountId"
		value = "accId"
	}
	properties {
		name = "appAccessMethod_static_loginUrl"
		value = "https://test-environment.com"
	}
	properties {
		name = "username"
		value = "test-uname"
	}
	properties {
		name = "role"
		value = "Test-Role"
	}
	sensitive_properties {
		name = "privateKeyPassword"
		value = "<Private-Key-Password>"
	}
	sensitive_properties {
		name = "publicKey"
		value = "<Public-Key>"
	}
	sensitive_properties {
		name = "privateKey"
		value = "<Private-Key>"
	}
	}
	`)
}

func testAccCheckBritiveEntityEnvironmentExists(n string) resource.TestCheckFunc {
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
