package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfileIdentity(t *testing.T) {
	applicationName := "Azure-ValueLabs"
	profileName := "AT - New Britive Profile Identity Test"
	profileDescription := "AT - New Britive Profile Identity Test Description"
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
		expiration_duration = "25m0s"
		associations {
			type  = "Environment"
			value = "QA Subscription"
		}
	}

	resource "britive_profile_identity" "new" {
		profile_id = britive_profile.new.id
    	username = "%s"
	}`, applicationName, profileName, profileDescription, username)

}

func testAccCheckBritiveProfileIdentityExists(n string) resource.TestCheckFunc {
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
