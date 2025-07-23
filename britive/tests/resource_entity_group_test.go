package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveEntityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
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

	resource "britive_entity_group" "entity_group_new" {
    application_id     = britive_application.snowflake_standalone_new.id
    entity_name        = "AT - Entity Group"
    entity_description = "AT - Entity Group Description"
    parent_id = britive_application.snowflake_standalone_new.entity_root_environment_group_id
	}
	`)
}

func testAccCheckBritiveEntityGroupExists(n string) resource.TestCheckFunc {
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
