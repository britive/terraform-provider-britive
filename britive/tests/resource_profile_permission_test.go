package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveProfilePermission(t *testing.T) {
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	profileName := "AT - New Britive Profile Permission Test"
	profileDescription := "AT - New Britive Profile Permission Test Description"
	associationValue := "QA"
	permissionName := "Application Developer"
	permissionType := "role"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePermissionConfig(applicationName, profileName, profileDescription, associationValue, permissionName, permissionType),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePermissionExists("britive_profile_permission.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfilePermissionConfig(applicationName, profileName, profileDescription, associationValue, permissionName, permissionType string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "EnvironmentGroup"
			value = "%s"
		}
	}

	resource "britive_profile_permission" "new" {
		profile_id = britive_profile.new.id
		permission_name = "%s"
		permission_type = "%s"
	}`, applicationName, profileName, profileDescription, associationValue, permissionName, permissionType)
}

func testAccCheckBritiveProfilePermissionExists(resourceName string) resource.TestCheckFunc {
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
