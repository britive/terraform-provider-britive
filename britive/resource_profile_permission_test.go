package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfilePermission(t *testing.T) {
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	profileName := "AT - New Britive Profile Permission Test"
	profileDescription := "AT - New Britive Profile Permission Test Description"
	permissionName := "Application Developer"
	permissionType := "role"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePermissionConfig(applicationName, profileName, profileDescription, permissionName, permissionType),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePermissionExists("britive_profile_permission.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfilePermissionConfig(applicationName, profileName, profileDescription, permissionName, permissionType string) string {
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
			value = "Root"
		}
	}

	resource "britive_profile_permission" "new" {
		profile_id = britive_profile.new.id
		permission_name = "%s"
		permission_type = "%s"
	}`, applicationName, profileName, profileDescription, permissionName, permissionType)

}

func testAccCheckBritiveProfilePermissionExists(n string) resource.TestCheckFunc {
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
