package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceLabel(t *testing.T) {
	resourceLabelName := "AT-Britive_Resource_Manager_Test_Resource_Label"
	resourceLabelDescription := "AT-Britive_Resource_Manager_Test_Resource_Label_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription),
				Check: resource.ComposeTestCheckFunc(
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

func testAccCheckBritiveResourceLabelExists(n string) resource.TestCheckFunc {
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
