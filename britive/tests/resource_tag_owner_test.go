package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveTagOwner(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Tag Owner Test"
	tagDescription := "AT - New Britive Tag Owner Test Description"
	ownerTagName := "AT - New Britive Tag Owner Test Owner Tag"
	ownerTagDescription := "AT - New Britive Tag Owner Test Owner Tag Description"
	ownerUsername := "britiveprovideracceptancetest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
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

// TestBritiveTagOwnerForEachDynamic is a regression test: count on the resource
// combined with a dynamic "user" block driven by count.index previously crashed
// ValidateConfig's req.Config.Get into []TagOwnerEntityModel, since the un-expanded
// resource is validated with count.index left unknown.
//
// count (not for_each) is used here deliberately: terraform-plugin-testing's legacy
// state shim used by TestCheckResourceAttr/RootModule().Resources lookups does not
// support for_each string-keyed instances ("unexpected index type (string) ...,
// for_each is not supported"), only count's integer-keyed instances.
func TestBritiveTagOwnerForEachDynamic(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Tag Owner ForEach Test"
	tagDescription := "AT - New Britive Tag Owner ForEach Test Description"
	ownerUsername := "britiveprovideracceptancetest"
	resourceName := "britive_tag_owner.iterated.0"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagOwnerForEachDynamicConfig(identityProviderName, tagName, tagDescription, ownerUsername),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagOwnerExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "tag_id"),
					resource.TestCheckResourceAttr(resourceName, "user.0.name", ownerUsername),
				),
			},
		},
	})
}

func testAccCheckBritiveTagOwnerForEachDynamicConfig(identityProviderName, tagName, tagDescription, ownerUsername string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	locals {
		owners = [
			{
				users = [{ name = "%s" }]
			}
		]
	}

	resource "britive_tag_owner" "iterated" {
		count = length(local.owners)

		tag_id = britive_tag.new.id

		dynamic "user" {
			for_each = local.owners[count.index].users
			content {
				name = user.value.name
			}
		}
	}`, identityProviderName, tagName, tagDescription, ownerUsername)
}
