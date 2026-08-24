package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveTagMember(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Tag Member Test"
	tagDescription := "AT - New Britive Tag Member Test Description"
	username := "britiveprovideracceptancetest"
	resourceAddr := "britive_tag_member.new"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagMemberConfig(identityProviderName, tagName, tagDescription, username),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagMemberExists(resourceAddr),
				),
			},
			// Import via the documented "tag-name/{tag_name}/username/{username}" format.
			{
				ResourceName:      resourceAddr,
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("tag-name/%s/username/%s", tagName, username),
				ImportStateVerify: true,
			},
			// Import via the undocumented but retained "tags/{tag_name}/users/{username}" format.
			{
				ResourceName:      resourceAddr,
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("tags/%s/users/%s", tagName, username),
				ImportStateVerify: true,
			},
			// Import via the bare "{tag_name}/{username}" format.
			{
				ResourceName:      resourceAddr,
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", tagName, username),
				ImportStateVerify: true,
			},
			// Import via the ID-first "tags/{tag_id}/users/{user_id}" format.
			{
				ResourceName:      resourceAddr,
				ImportState:       true,
				ImportStateIdFunc: testAccTagMemberImportStateIdByIDs(resourceAddr),
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTagMemberImportStateIdByIDs(resourceAddr string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", resourceAddr)
		}
		tagID := rs.Primary.Attributes["tag_id"]
		userID := rs.Primary.Attributes["user_id"]
		if tagID == "" || userID == "" {
			return "", errs.NewNotFoundErrorf("tag_id/user_id for %s in state", resourceAddr)
		}
		return fmt.Sprintf("tags/%s/users/%s", tagID, userID), nil
	}
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

func testAccCheckBritiveTagMemberExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return errs.NewNotFoundErrorf("%s in state", n)
		}

		if rs.Primary.ID == "" {
			return errs.NewNotFoundErrorf("ID for %s in state", n)
		}

		return nil
	}
}
