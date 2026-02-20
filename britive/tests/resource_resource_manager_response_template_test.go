package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveResourceManagerResponseTemplate(t *testing.T) {
	name := "AT_Britive_Resource_Manager_Tests_Response_Template"
	description := "AT_Britive_Resource_Manager_Tests_Response_Template_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResponseTemplateConfig(name, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveResponseTemplateExists("britive_resource_manager_response_template.new_response_template_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResponseTemplateConfig(name, description string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_response_template" "new_response_template_1" {
    	name        = "%s"
    	description = "%s"
    	template_data = "The user {{name}} for the role {{role}}."
    	is_console_access_enabled = true
    	show_on_ui = false
	}`, name, description)
}

func testAccCheckBritiveResponseTemplateExists(resourceName string) resource.TestCheckFunc {
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
