package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritiveRole(t *testing.T) {
	permissionName := "AT - Britive Permission Test Role"
	permissionDescription := "AT - Britive Permission Test Role Description"
	roleName := "AT - Britive Role Test"
	roleDescription := "AT - Britive Role Test Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRoleConfig(permissionName, permissionDescription, roleName, roleDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveRoleExists("britive_role.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveRoleConfig(permissionName, permissionDescription, roleName, roleDescription string) string {
	return fmt.Sprintf(`
	resource "britive_permission" "new_permission_role_1" {
		name = "%s1"
		description = "%s1"
		consumer    = "authz"
		resources   = [
			"*",
		]
		actions     = [
			"authz.action.list",
			"authz.action.read",
		]
	}

	resource "britive_permission" "new_permission_role_2" {
		name = "%s2"
		description = "%s2"
		consumer    = "diagnostics"
		resources   = [
			"*",
		]
		actions     = [
			"diagnostics.audit.list",
			"diagnostics.audit.view",
		]
	}

	resource "britive_role" "new" {
		name = "%s"
		description = "%s"
		permissions = jsonencode(
			[
				{
					name = britive_permission.new_permission_role_1.name
				},
				{
					name = britive_permission.new_permission_role_2.name
				},
			]
		)
	}`, permissionName, permissionDescription, permissionName, permissionDescription, roleName, roleDescription)

}

func testAccCheckBritiveRoleExists(resourceName string) resource.TestCheckFunc {
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
