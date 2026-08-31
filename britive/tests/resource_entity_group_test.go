package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveEntityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveEntityGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveEntityGroupExists("britive_application.snowflake_standalone_new"),
					testAccCheckBritiveEntityGroupExists("britive_entity_group.entity_group_new"),
					testAccCheckBritiveEntityGroupExists("britive_application.kubernetes_new"),
					testAccCheckBritiveEntityGroupExists("britive_entity_group.kubernetes_entity_group_new"),
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

	resource "britive_application" "kubernetes_new" {
    application_type = "Kubernetes"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "AT - Kubernetes APP"
    }
    properties {
      name = "description"
      value = "AT - Kubernetes APP DESC"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = 1000
    }
    properties {
      name = "displayProgrammaticKeys"
      value = true
    }
	}

	resource "britive_entity_group" "kubernetes_entity_group_new" {
    application_id     = britive_application.kubernetes_new.id
    entity_name        = "AT - Kubernetes Entity Group"
    entity_description = "AT - Kubernetes Entity Group Description"
    parent_id = britive_application.kubernetes_new.entity_root_environment_group_id
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
