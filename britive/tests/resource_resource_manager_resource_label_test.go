package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveResourceManagerResourceLabel(t *testing.T) {
	resourceLabelName := "AT-Britive_Resource_Manager_Test_Resource_Label"
	resourceLabelDescription := "AT-Britive_Resource_Manager_Test_Resource_Label_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveResourceLabelExists("britive_resource_manager_resource_label.resource_label_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_label" "resource_label_1" {
		name         = "%s"
		description  = "%s"
		label_color  = "#abc123"

		values {
			name = "YS Val"
			description = "YS Val Desc"
		}
		values {
			name = "YS Val 1"
			description = "YS Val Desc1"
		}
	}
	`, resourceLabelName, resourceLabelDescription)
}

func testAccCheckBritiveResourceLabelExists(resourceName string) resource.TestCheckFunc {
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
