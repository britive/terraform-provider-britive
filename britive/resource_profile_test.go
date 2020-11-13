package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfile(t *testing.T) {
	name := "BPAT - New Britive Profile Test"
	description := "BPAT - New Britive Profile Test Description"
	applicationName := "Azure-ValueLabs"
	resourceName := "britive_profile.new"
	associationType := "Environment"
	associationValue := "QA Subscription"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileConfig(name, description, applicationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckBritiveProfileConfigAddAssociations(name, description, applicationName, associationType, associationValue),
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

func testAccCheckBritiveProfileConfig(name string, description string, applicationName string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "Environment"
			value = "QA Subscription"
		}
	}`, applicationName, name, description)
}

func testAccCheckBritiveProfileConfigAddAssociations(name, description, applicationName, associationType, associationValue string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "%s"
			value = "%s"
		}
	}`, applicationName, name, description, associationType, associationValue)
}

func testAccCheckBritiveProfileExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ProfileID set")
		}

		return nil
	}
}
