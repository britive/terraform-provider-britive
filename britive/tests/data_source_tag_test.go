package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBritiveDataSourceTagByName(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Data Source Tag Test"
	tagDescription := "AT - New Britive Data Source Tag Test Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveDataSourceTagByNameConfig(identityProviderName, tagName, tagDescription),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.britive_tag.by_name", "id"),
					resource.TestCheckResourceAttr("data.britive_tag.by_name", "name", tagName),
					resource.TestCheckResourceAttrSet("data.britive_tag.by_name", "tag_id"),
				),
			},
		},
	})
}

func TestBritiveDataSourceTagByID(t *testing.T) {
	identityProviderName := "Britive"
	tagName := "AT - New Britive Data Source Tag By ID Test"
	tagDescription := "AT - New Britive Data Source Tag By ID Test Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveDataSourceTagByIDConfig(identityProviderName, tagName, tagDescription),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.britive_tag.by_id", "id"),
					resource.TestCheckResourceAttr("data.britive_tag.by_id", "name", tagName),
					resource.TestCheckResourceAttrSet("data.britive_tag.by_id", "tag_id"),
				),
			},
		},
	})
}

func testAccCheckBritiveDataSourceTagByNameConfig(identityProviderName, tagName, tagDescription string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "test" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	data "britive_tag" "by_name" {
		name = britive_tag.test.name
	}
	`, identityProviderName, tagName, tagDescription)
}

func testAccCheckBritiveDataSourceTagByIDConfig(identityProviderName, tagName, tagDescription string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "test" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
	}

	data "britive_tag" "by_id" {
		tag_id = britive_tag.test.id
	}
	`, identityProviderName, tagName, tagDescription)
}
