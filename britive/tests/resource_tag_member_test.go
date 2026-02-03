package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveTagMember(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Tag Member Test"
	tagDescription := "AT - New Britive Tag Member Test Description"
	username := "britiveprovideracceptancetest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagMemberConfig(identityProviderName, tagName, tagDescription, username),
				Check: resource.ComposeAggregateTestCheckFunc(
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
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	resource "britive_tag_member" "new" {
		tag_id = britive_tag.new.id
    	username = "%s"
	}
	`, identityProviderName, tagName, tagDescription, username)

}

func testAccCheckBritiveTagMemberExists(resourceName string) resource.TestCheckFunc {
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
