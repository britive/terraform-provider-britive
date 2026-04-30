package tests

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
					// requestable is Optional+Computed: not set in config, value is
					// determined by the backend and must be present in state.
					resource.TestCheckResourceAttrSet("britive_tag.new", "requestable"),
				),
			},
		},
	})
}

func TestBritiveTagRequestable(t *testing.T) {
	name := "AT - New Britive Tag Requestable Test"
	description := "AT - New Britive Tag Requestable Test Description"
	identityProviderName := "Britive"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create tag with requestable explicitly set to true
				Config: testAccCheckBritiveTagRequestableConfig(name, description, identityProviderName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.new_requestable"),
					resource.TestCheckResourceAttr("britive_tag.new_requestable", "requestable", "true"),
				),
			},
		},
	})
}

func TestBritiveTagWithAttributes(t *testing.T) {
	name := "AT - New Britive Tag Attributes Test"
	description := "AT - New Britive Tag Attributes Test Description"
	identityProviderName := "Britive"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Step 1: create with requestable=true and three attributes (one multi-valued)
				Config: testAccCheckBritiveTagWithAttributesConfig(name, description, identityProviderName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.new_with_attrs"),
					resource.TestCheckResourceAttr("britive_tag.new_with_attrs", "requestable", "true"),
					resource.TestCheckResourceAttr("britive_tag.new_with_attrs", "attributes.#", "3"),
				),
			},
			{
				// Step 2: set requestable=false and remove one multi-valued attribute entry
				Config: testAccCheckBritiveTagWithAttributesUpdatedConfig(name, description, identityProviderName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagExists("britive_tag.new_with_attrs"),
					resource.TestCheckResourceAttr("britive_tag.new_with_attrs", "requestable", "false"),
					resource.TestCheckResourceAttr("britive_tag.new_with_attrs", "attributes.#", "2"),
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

func testAccCheckBritiveTagRequestableConfig(name, description, identityProviderName string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new_requestable" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
		requestable          = true
	}`, identityProviderName, name, description)
}

func testAccCheckBritiveTagWithAttributesConfig(name, description, identityProviderName string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new_with_attrs" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
		requestable          = true

		attributes {
			attribute_name  = "Owner"
			attribute_value = "test1"
		}

		attributes {
			attribute_name  = "MultiVal"
			attribute_value = "23"
		}

		attributes {
			attribute_name  = "MultiVal"
			attribute_value = "45"
		}
	}`, identityProviderName, name, description)
}

func testAccCheckBritiveTagWithAttributesUpdatedConfig(name, description, identityProviderName string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "existing" {
		name = "%s"
	}

	resource "britive_tag" "new_with_attrs" {
		name                 = "%s"
		description          = "%s"
		identity_provider_id = data.britive_identity_provider.existing.id
		requestable          = false

		attributes {
			attribute_name  = "Owner"
			attribute_value = "test1"
		}

		attributes {
			attribute_name  = "MultiVal"
			attribute_value = "23"
		}
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
