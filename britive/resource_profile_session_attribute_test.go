package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfileSessionAttribute(t *testing.T) {
	applicationName := "AWS-ValueLabs"
	profileName := "AT - New Britive Profile Session Attribute Test"
	attributeName := "Date Of Birth"
	mappingName := "dob"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileSessionAttributeConfig(applicationName, profileName, attributeName, mappingName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileSessionAttributeExists("britive_profile_session_attribute.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileSessionAttributeConfig(applicationName, profileName, attributeName, mappingName string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "EnvironmentGroup"
			value = "QA"
		}
	}

	resource "britive_profile_session_attribute" "new" {
		profile_id 		= britive_profile.new.id
		attribute_name	= "%s"
		mapping_name	= "%s"
		transitive		= true
	}	
	`, applicationName, profileName, attributeName, mappingName)
}

func testAccCheckBritiveProfileSessionAttributeExists(n string) resource.TestCheckFunc {
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
