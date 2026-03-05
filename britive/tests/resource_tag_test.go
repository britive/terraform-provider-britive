package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveTag(t *testing.T) {
	name := "AT - New Britive Tag Test"
	description := "AT - New Britive Tag Test Description"
	identityProviderName := "Britive"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveTagConfig(name, description, identityProviderName),
				Check: resource.ComposeAggregateTestCheckFunc(
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

func testAccCheckBritiveTagExists(resourceName string) resource.TestCheckFunc {
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
