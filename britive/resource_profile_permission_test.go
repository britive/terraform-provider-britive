package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfilePermission(t *testing.T) {
	applicationName := "Azure-ValueLabs"
	profileName := "BPAT - New Britive Profile Permission Test"
	profileDescription := "BPAT - New Britive Profile Permission Test Description"
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
			type  = "Environment"
			value = "QA Subscription"
		}
	}

	resource "britive_profile_permission" "new" {
		profile_id = britive_profile.new.profile_id
		permission_name = "%s"
		permission_type = "%s"
	}`, applicationName, profileName, profileDescription, permissionName, permissionType)

}

func testAccCheckBritiveProfilePermissionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Profile Idenity ID set")
		}

		return nil
	}
}
