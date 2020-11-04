package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfileIdentity(t *testing.T) {
	applicationName := "Azure-ValueLabs"
	profileName := "BPAT - New Britive Profile Identity Test"
	profileDescription := "BPAT - New Britive Profile Identity Test Description"
	username := "britiveprovideracceptancetest"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileIdentityConfig(applicationName, profileName, profileDescription, username),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileIdentityExists("britive_profile_identity.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileIdentityConfig(applicationName string, profileName string, profileDescription string, username string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		status = "active"
		expiration_duration = "25m0s"
	}

	resource "britive_profile_identity" "new" {
		profile_id = britive_profile.new.profile_id
    	username = "%s"
	}`, applicationName, profileName, profileDescription, username)

}

func testAccCheckBritiveProfileIdentityExists(n string) resource.TestCheckFunc {
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
