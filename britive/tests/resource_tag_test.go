package britivetest

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveTag(t *testing.T) {
	name := "AT - New Britive Tag Test"
	description := "AT - New Britive Tag Test Description"
	identityProviderName := "Britive"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagConfig(name, description, identityProviderName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveTagConfig(name string, description string, identityProviderName string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new" {
		name = "%s"
		description = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
	}`, identityProviderName, name, description)
}

func testAccCheckBritiveTagExists(n string) resource.TestCheckFunc {
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
