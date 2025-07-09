package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveAdvancedSettings(t *testing.T) {
	applicationName := "DO NOT DELETE - AWS TF Plugin"
	profileName := "AT - TF ADVANCED SETTINGS PROFILE"
	profilePolicyName := "AT - TF ADVANCED SETTINGS PROFILE POLICY"
	profilePolicyDescription := "AT - TF ADVANCED SETTINGS PROFILE POLICY DESCRIPTION"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveAdvancedSettingsConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveAdvancedSettingsExists("britive_advanced_settings.new_application_advanced_settings"),
					testAccCheckBritiveAdvancedSettingsExists("britive_advanced_settings.new_profile_advanced_settings"),
					testAccCheckBritiveAdvancedSettingsExists("britive_advanced_settings.new_profile_policy_advanced_settings"),
				),
			},
		},
	})
}

func testAccCheckBritiveAdvancedSettingsConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription string) string {
	return fmt.Sprintf(`
	data "britive_connection" "new_connection"{
		name = "TF_ACCEPTANCE_TEST_ITSM_DO_NOT_DELETE"
	}

	data "britive_application" "new_app" {
		name = "%s"
	}

	resource "britive_profile" "new_profile" {
		app_container_id = data.britive_application.new_app.id
		name = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "EnvironmentGroup"
			value = "Development"
		}
		associations {
			type  = "EnvironmentGroup"
			value = "Stage"
		}
		associations {
			type  = "Environment"
			value = "Sigma Corporate"
		}
	}

	resource "britive_profile_policy" "new_profile_policy" {
		policy_name  = "%s"
		description  = "%s"
		profile_id   = britive_profile.new_profile.id
		access_type  = "Allow"
		members      = jsonencode(
			{
				users             = [
					{
						name = "britiveprovideracceptancetest"
					},
				]
			}
		)
		consumer     = "papservice"   
		is_active    = true
		is_draft     = false
		is_read_only = false
	}

	resource "britive_advanced_settings" "new_application_advanced_settings" {
		resource_id   = data.britive_application.new_app.id
		resource_type = "APPLICATION"

		justification_settings {
			is_justification_required = true
			justification_regex        = "AT - TEST APP ADVANCED SETTINGS"
		}

		itsm {
			connection_id       = data.britive_connection.new_connection.id
			connection_type     = data.britive_connection.new_connection.type
			is_itsm_enabled     = false

			itsm_filter_criteria {
			supported_ticket_type = "issue"
			filter                = jsonencode({
				jql = ""
			})
			}
		}
	}

	resource "britive_advanced_settings" "new_profile_advanced_settings" {
		resource_id   = britive_application.new_profile.id
		resource_type = "PROFILE"

		justification_settings {
			is_justification_required = true
			justification_regex        = "AT - TEST PROFILE ADVANCED SETTINGS"
		}

		itsm {
			connection_id       = data.britive_connection.new_connection.id
			connection_type     = data.britive_connection.new_connection.type
			is_itsm_enabled     = false

			itsm_filter_criteria {
			supported_ticket_type = "issue"
			filter                = jsonencode({
				jql = ""
			})
			}
		}
	}

	resource "britive_advanced_settings" "new_profile_policy_advanced_settings" {
		resource_id   = britive_application.new_profile_policy.id
		resource_type = "PROFILE"

		justification_settings {
			is_justification_required = true
			justification_regex        = "AT - TEST PROFILE POLICY ADVANCED SETTINGS"
		}

		itsm {
			connection_id       = data.britive_connection.new_connection.id
			connection_type     = data.britive_connection.new_connection.type
			is_itsm_enabled     = false

			itsm_filter_criteria {
			supported_ticket_type = "issue"
			filter                = jsonencode({
				jql = ""
			})
			}
		}
	}

`, applicationName, profileName, profilePolicyName, profilePolicyDescription)
}

func testAccCheckBritiveAdvancedSettingsExists(n string) resource.TestCheckFunc {
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
