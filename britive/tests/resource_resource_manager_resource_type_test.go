package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceType(t *testing.T) {
	name := "AT_Britive_Resource_Manager_Test_Resource_Type"
	description := "AT_Britive_Resource_Manager_Test_Resource_Type_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypeConfig(name, description),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypeExists("britive_resource_manager_resource_type.new_resource_type_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceTypeConfig(name, description string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_1" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name = "testfield1"
			param_type = "password"
			is_mandatory = true
		}
		parameters {
			param_name = "testfield2"
			param_type = "Password"
			is_mandatory = false
		}
		parameters {
			param_name = "testfield3"
			param_type = "string"
			is_mandatory = true
		}
		parameters {
			param_name = "testfield4"
			param_type = "stRing"
			is_mandatory = false
		}
		parameters {
			param_name   = "testfield5"
			param_type   = "ip-cidr"
			is_mandatory = true
		}
		parameters {
			param_name   = "testfield6"
			param_type   = "iP-cIdr"
			is_mandatory = true
		}
		parameters {
			param_name   = "testfield7"
			param_type   = "regex-pattern"
			is_mandatory = true
		}
		parameters {
			param_name   = "testfield8"
			param_type   = "reGex-pAttErn"
			is_mandatory = true
		}
		parameters {
			param_name   = "testfield9"
			param_type   = "list"
			is_mandatory = true
		}
			parameters {
			param_name   = "testfield10"
			param_type   = "liSt"
			is_mandatory = true
		}
	}`, name, description)
}

func testAccCheckBritiveResourceTypeExists(n string) resource.TestCheckFunc {
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
