package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfile(t *testing.T) {
	name := "BPAT - New Britive Profile Test"
	description := "BPAT - New Britive Profile Test Description"
	applicationName := "Azure-ValueLabs"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileConfig(name, description, applicationName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileExists("britive_profile.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileConfig(name string, description string, applicationName string) string {
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
	}`, applicationName, name, description)
}

func testAccCheckBritiveProfileExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ProfileID set")
		}

		return nil
	}
}
