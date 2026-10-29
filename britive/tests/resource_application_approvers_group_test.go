package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBritiveApplicationApproversGroup(t *testing.T) {
	appName := "AT-App_Approvers_Group"
	tagName := "AT - Approvers Group Tag"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationApproversGroupConfig(appName, tagName, "AT-approvers", "All"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("britive_application_approvers_group.test", "group_id"),
					resource.TestCheckResourceAttr("britive_application_approvers_group.test", "condition", "All"),
					resource.TestCheckResourceAttr("britive_application_approvers_group.test", "member.#", "2"),
				),
			},
			// In-place update: name and condition change without replacing the resource.
			{
				Config: testAccCheckBritiveApplicationApproversGroupConfig(appName, tagName, "AT-approvers-renamed", "Any"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_application_approvers_group.test", "name", "AT-approvers-renamed"),
					resource.TestCheckResourceAttr("britive_application_approvers_group.test", "condition", "Any"),
				),
			},
			{
				ResourceName:            "britive_application_approvers_group.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"member"},
			},
		},
	})
}

func testAccCheckBritiveApplicationApproversGroupConfig(appName, tagName, groupName, condition string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "Britive"
	}

	resource "britive_tag" "approvers_a" {
		name                 = "%s A"
		description          = "%s A Description"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	resource "britive_tag" "approvers_b" {
		name                 = "%s B"
		description          = "%s B Description"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	resource "britive_application" "approvers_group_app" {
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

	resource "britive_application_approvers_group" "test" {
		application_id = britive_application.approvers_group_app.id
		name           = "%s"
		condition      = "%s"

		member {
			type = "Tag"
			name = britive_tag.approvers_a.name
		}

		member {
			type = "Tag"
			id   = britive_tag.approvers_b.id
		}
	}`, tagName, tagName, tagName, tagName, appName, appName, groupName, condition)
}
