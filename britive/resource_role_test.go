package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveRole(t *testing.T) {
	permissionName := "AT - Britive Permission Test Role"
	permissionDescription := "AT - Britive Permission Test Role Description"
	roleName := "AT - Britive Role Test"
	roleDescription := "AT - Britive Role Test Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRoleConfig(permissionName, permissionDescription, roleName, roleDescription),
				Check: resource.ComposeTestCheckFunc(
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

func testAccCheckBritiveRoleExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return NewNotFoundErrorf("%s in state", n)
		}

		if rs.Primary.ID == "" {
			return NewNotFoundErrorf("ID for %s in state", n)
		}

		return nil
	}
}
