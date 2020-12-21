package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfileTag(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "BPAT - New Britive Tag Test"
	applicationName := "Azure-ValueLabs"
	profileName := "BPAT - New Britive Profile Tag Test"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileTagConfig(identityProviderName, tagName, applicationName, profileName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileTagExists("britive_profile_tag.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileTagConfig(identityProviderName, tagName, applicationName, profileName string) string {
	return fmt.Sprintf(`
	
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new" {
		name = "%s"
		description = "BPAT - Profile Tag Test"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "Environment"
			value = "QA Subscription"
		}
	}

	resource "britive_profile_tag" "new" {
		profile_id = britive_profile.new.id
		tag_name   = britive_tag.new.name
	}
	
	`, identityProviderName, tagName, applicationName, profileName)
}

func testAccCheckBritiveProfileTagExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Profile Tag ID set")
		}

		return nil
	}
}
