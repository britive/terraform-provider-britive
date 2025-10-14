package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveCustomTemplate(t *testing.T) {
	templateFileName := "AT_Custom_Template.baml"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveCustomTemplateConfig(templateFileName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveCustomTemplateExists("britive_custom_template.new_template"),
				),
			},
		},
	})
}

func testAccCheckBritiveCustomTemplateConfig(templateFileName string) string {
	return fmt.Sprintf(`
	resource "britive_custom_template" "new_template" {
		template = file("%s")
		template_name = "%s"
	}
	`, templateFileName, templateFileName)

}

func testAccCheckBritiveCustomTemplateExists(n string) resource.TestCheckFunc {
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
