package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveEntityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveEntityGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveEntityGroupExists("britive_application.snowflake_standalone_new"),
					testAccCheckBritiveEntityGroupExists("britive_entity_group.entity_group_new"),
				),
			},
		},
	})
}

func testAccCheckBritiveEntityGroupConfig() string {
	return `
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

	resource "britive_entity_group" "entity_group_new" {
    application_id     = britive_application.snowflake_standalone_new.id
    entity_name        = "AT - Entity Group"
    entity_description = "AT - Entity Group Description"
    parent_id = britive_application.snowflake_standalone_new.entity_root_environment_group_id
	}
	`
}

func testAccCheckBritiveEntityGroupExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %s ID is not set", resourceName)
		}

		return nil
	}
}
