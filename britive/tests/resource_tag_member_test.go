package britivetest

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveTagMember(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Tag Member Test"
	tagDescription := "AT - New Britive Tag Member Test Description"
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
