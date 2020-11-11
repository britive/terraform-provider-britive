package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveTagMember(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "BPAT - New Britive Tag Member Test"
	tagDescription := "BPAT - New Britive Tag Member Test Description"
	username := "britiveprovideracceptancetest"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagMemberConfig(identityProviderName, tagName, tagDescription, username),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagMemberExists("britive_tag_member.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveTagMemberConfig(identityProviderName string, tagName string, tagDescription string, username string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new" {
		name = "%s"
		description = "%s"
		user_tag_identity_providers {
			identity_provider {
				id = data.britive_identity_provider.existing.id
			}
		}
	}

	resource "britive_tag_member" "new" {
		tag_id = britive_tag.new.id
    	username = "%s"
	}
	`, identityProviderName, tagName, tagDescription, username)

}

func testAccCheckBritiveTagMemberExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No TagID set")
		}

		return nil
	}
}
