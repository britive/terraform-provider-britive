package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The association approver uses the application's root environment group, by ID, since a
// "Britive" type application has no environments of its own. profile_expiration_timeout is 15
// minutes because the API limits its range per tenant.

func TestBritiveApplicationAccessBuilderSettings(t *testing.T) {
	appName := "AT-App_Access_Builder_Settings"
	tagName := "AT - Access Builder Settings Tag"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationAccessBuilderSettingsConfig(appName, tagName, true, "00:01:00", "Include"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("britive_application_access_builder_settings.test", "id"),
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "allow_access_request", "true"),
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "association_approver.#", "1"),
				),
			},
			// In-place update: disable access requests, change the timeout and member rule.
			{
				Config: testAccCheckBritiveApplicationAccessBuilderSettingsConfig(appName, tagName, false, "00:02:00", "Exclude"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "allow_access_request", "false"),
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "approval_timeout", "00:02:00"),
				),
			},
			{
				ResourceName:            "britive_application_access_builder_settings.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"notification_medium", "member_rule"},
			},
		},
	})
}

func testAccCheckBritiveApplicationAccessBuilderSettingsConfig(appName, tagName string, allow bool, approvalTimeout, memberRuleCondition string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "Britive"
	}

	data "britive_notification_medium" "email" {
		name = "Email"
	}

	resource "britive_tag" "requesters" {
		name                 = "%s"
		description          = "%s Description"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	resource "britive_application" "access_request_app" {
		application_type = "Britive"
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_approvers_group" "approvers" {
		application_id = britive_application.access_request_app.id
		name           = "AT-access-request-approvers"
		condition      = "Any"

		member {
			type = "Tag"
			id   = britive_tag.requesters.id
		}
	}

	resource "britive_application_access_builder_settings" "test" {
		application_id             = britive_application.access_request_app.id
		allow_access_request       = %t
		approval_timeout           = "%s"
		profile_expiration_timeout = 15

		notification_medium {
			id = data.britive_notification_medium.email.id
		}

		association_approver {
			name = "AT-root-approvers"

			association {
				type  = "EnvironmentGroup"
				value = britive_application.access_request_app.entity_root_environment_group_id
			}

			approvers_group_ids = [britive_application_approvers_group.approvers.group_id]
		}

		member_rule {
			type      = "Tag"
			name      = britive_tag.requesters.name
			condition = "%s"
		}
	}`, tagName, tagName, appName, appName, allow, approvalTimeout, memberRuleCondition)
}

// TestBritiveApplicationAccessBuilderSettingsNotificationOnly manages only notification
// mediums; every other field is left unmanaged.
func TestBritiveApplicationAccessBuilderSettingsNotificationOnly(t *testing.T) {
	appName := "AT-App_Access_Request_Notification_Only"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
	resource "britive_application" "notification_only_app" {
		application_type = "Britive"
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_access_builder_settings" "test" {
		application_id = britive_application.notification_only_app.id

		notification_medium {
			name = "Email"
		}
	}`, appName, appName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "notification_medium.#", "1"),
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "association_approver.#", "0"),
					resource.TestCheckResourceAttr("britive_application_access_builder_settings.test", "allow_access_request", "false"),
				),
			},
		},
	})
}
