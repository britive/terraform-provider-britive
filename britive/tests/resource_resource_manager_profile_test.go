package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceManagerProfile(t *testing.T) {
	resourceLabelName1 := "AT-Britive_Resource_Manager_Test_Resource_Label_11"
	resourceLabelDescription1 := "AT-Britive_Resource_Manager_Test_Resource_Label_11_Description"
	resourceLabelName2 := "AT-Britive_Resource_Manager_Test_Resource_Label_22"
	resourceLabelDescription2 := "AT-Britive_Resource_Manager_Test_Resource_Label_22_Description"
	resourceProfileName := "AT-Britive_Resource_Manager_Test_Resource_Profile-_11"
	resourceProfileDescription := "AT-Britive_Resource_Manager_Test_Resource_Profile_11_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfileConfig(resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_resource_label.resource_label_1"),
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_resource_label.resource_label_2"),
					testAccCheckBritiveResourceManagerProfileExists("britive_resource_manager_profile.resource_profile_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfileConfig(resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_label" "resource_label_1" {
		name         = "%s"
		description  = "%s"
		label_color  = "#abc123"

		values {
			name = "Production"
			description = "Production Desc"
		}
		values {
			name = "Development"
			description = "Development Desc"
		}
	}

	resource "britive_resource_manager_resource_label" "resource_label_2" {
		name         = "%s"
		description  = "%s"
		label_color  = "#1a2b3c"

		values {
			name = "us-east-1"
			description = "us-east-1 Desc"
		}
		values {
			name = "eu-west-1"
			description = "eu-west-1 Desc"
		}
	}

	resource "britive_resource_manager_profile" "resource_profile_1" {
		name                 = "%s"
		description          = "%s"
		expiration_duration  = 3600000
		delegation_enabled  = true

		associations {
			label_key   = britive_resource_manager_resource_label.resource_label_1.name
			values = ["Production", "Development"]
		}
		associations {
			label_key   = britive_resource_manager_resource_label.resource_label_2.name
			values = ["us-east-1", "eu-west-1"]
		}
	}
	`, resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription)
}

func testAccCheckBritiveResourceManagerProfileExists(n string) resource.TestCheckFunc {
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
