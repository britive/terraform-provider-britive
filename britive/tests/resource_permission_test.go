package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBritivePermission(t *testing.T) {
	permissionName := "AT - Britive Permission Test"
	permissionDescription := "AT - Britive Permission Test Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePermissionConfig(permissionName, permissionDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
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

func testAccCheckBritivePermissionExists(resourceName string) resource.TestCheckFunc {
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
