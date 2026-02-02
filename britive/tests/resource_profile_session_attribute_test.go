package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveProfileSessionAttribute(t *testing.T) {
	applicationName := "DO NOT DELETE - AWS TF Plugin"
	profileName := "AT - New Britive Profile Session Attribute Test"
	attributeName := "Date Of Birth"
	mappingName := "dob"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileSessionAttributeConfig(applicationName, profileName, attributeName, mappingName),
				Check: resource.ComposeAggregateTestCheckFunc(
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
			value = "Root"
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

func testAccCheckBritiveProfileSessionAttributeExists(resourceName string) resource.TestCheckFunc {
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
