package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveProfile(t *testing.T) {
	name := "AT - New Britive Profile Test"
	description := "AT - New Britive Profile Test Description"
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	resourceName := "britive_profile.new"

	associationType := "EnvironmentGroup"
	associationValue := "QA"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccBritiveProfileConfig(name, description, applicationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccBritiveProfileConfigAddAssociations(
					name, description, applicationName, associationType, associationValue,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "associations.0.type", associationType),
					resource.TestCheckResourceAttr(resourceName, "associations.0.value", associationValue),
				),
			},
		},
	})
}

func testAccBritiveProfileConfig(name string, description string, applicationName string) string {
	return fmt.Sprintf(`

	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id     = data.britive_application.app.id
		name                 = "%s"
		description          = "%s"
		expiration_duration  = "25m0s"
		allow_impersonation  = true

		associations {
			type  = "Environment"
			value = "Subscription 1"
		}
	}
	`, applicationName, name, description)
}

func testAccBritiveProfileConfigAddAssociations(
	name, description, applicationName, associationType, associationValue string,
) string {
	return fmt.Sprintf(`

	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id     = data.britive_application.app.id
		name                 = "%s"
		description          = "%s"
		expiration_duration  = "25m0s"

		associations {
			type  = "%s"
			value = "%s"
		}
	}
	`, applicationName, name, description,
		associationType, associationValue)
}

func testAccCheckBritiveProfileExists(resourceName string) resource.TestCheckFunc {
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
