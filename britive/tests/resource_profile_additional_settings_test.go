package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveProfileAdditionalSettings(t *testing.T) {
	applicationName := "DO NOT DELETE - GCP TF Plugin"
	profileName := "AT - New Britive Constraint Test"
	profileDescription := "AT - New Britive Constraint Test Description"
	associationValue := "britive-gdev-cis.net"
	useAppCredentialType := "false"
	consoleAccess := "true"
	programmaticAccess := "false"
	projectIdForServiceAccount := ""

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileAdditionalSettingsConfig(applicationName, profileName, profileDescription, associationValue, useAppCredentialType, consoleAccess, programmaticAccess, projectIdForServiceAccount),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileAdditionalSettingsExists("britive_profile_additional_settings.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileAdditionalSettingsConfig(applicationName, profileName, profileDescription, associationValue, useAppCredentialType, consoleAccess, programmaticAccess, projectIdForServiceAccount string) string {
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

	resource "britive_profile_additional_settings" "new" {
		profile_id = britive_profile.new.id
  		use_app_credential_type        = %s
  		console_access                 = %s
  		programmatic_access            = %s
  		project_id_for_service_account = "%s"
	}`, applicationName, profileName, profileDescription, associationValue, useAppCredentialType, consoleAccess, programmaticAccess, projectIdForServiceAccount)

}

func testAccCheckBritiveProfileAdditionalSettingsExists(resourceName string) resource.TestCheckFunc {
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
