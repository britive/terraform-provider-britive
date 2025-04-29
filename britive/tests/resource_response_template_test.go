package britivetest

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResponseTemplate(t *testing.T) {
	name := "AT - Britive Resource Manager Test Response Template"
	description := "AT - Britive Resource Manager Test Response Template Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResponseTemplateConfig(name, description),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypeExists("britive_response_template.new_resource_type_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResponseTemplateConfig(name, description string) string {
	return fmt.Sprintf(`
	resource "britive_response_template" "new_response_template_1" {
    	name        = "%s"
    	description = "%s"
    	template_data = "The user {{name}} for the role {{role}}."
    	is_console_access_enabled = true
    	show_on_ui = false
	}`, name, description)
}

func testAccCheckBritiveResponseTemplateExists(n string) resource.TestCheckFunc {
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
