package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritivePermission(t *testing.T) {
	permissionName := "AT - Britive Permission Test"
	permissionDescription := "AT - Britive Permission Test Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePermissionConfig(permissionName, permissionDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePermissionExists("britive_permission.new"),
				),
			},
		},
	})
}

func testAccCheckBritivePermissionConfig(permissionName, permissionDescription string) string {
	return fmt.Sprintf(`
	resource "britive_permission" "new" {
		name = "%s"
		description = "%s"
		consumer    = "authz"
		resources   = [
			"*",
		]
		actions     = [
			"authz.action.list",
			"authz.action.read",
		]
	}`, permissionName, permissionDescription)

}

func testAccCheckBritivePermissionExists(n string) resource.TestCheckFunc {
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
