package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveTagOwner(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Tag Owner Test"
	tagDescription := "AT - New Britive Tag Owner Test Description"
	ownerTagName := "AT - New Britive Tag Owner Test Owner Tag"
	ownerTagDescription := "AT - New Britive Tag Owner Test Owner Tag Description"
	ownerUsername := "britiveprovideracceptancetest"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Step 1: create with a user owner (by name) and a tag owner (by name)
				Config: testAccCheckBritiveTagOwnerConfig(identityProviderName, tagName, tagDescription, ownerTagName, ownerTagDescription, ownerUsername),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagOwnerExists("britive_tag_owner.new"),
					resource.TestCheckResourceAttrSet("britive_tag_owner.new", "tag_id"),
				),
			},
		},
	})
}

func testAccCheckBritiveTagOwnerConfig(identityProviderName, tagName, tagDescription, ownerTagName, ownerTagDescription, ownerUsername string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "ownertag" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	resource "britive_tag" "new" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	resource "britive_tag_owner" "new" {
		tag_id = britive_tag.new.id

		user {
			name = "%s"
		}

		tag {
			id = britive_tag.ownertag.id
		}
	}
	`, identityProviderName, ownerTagName, ownerTagDescription, tagName, tagDescription, ownerUsername)
}

func testAccCheckBritiveTagOwnerExists(n string) resource.TestCheckFunc {
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
