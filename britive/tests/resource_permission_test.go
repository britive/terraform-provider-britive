package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritivePermission(t *testing.T) {
	permissionName := "AT - Britive Permission Test"
	permissionDescription := "AT - Britive Permission Test Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

func TestBritivePermissionScopes(t *testing.T) {
	permissionName := "AT - Britive Permission Scopes Test"
	permissionDescription := "AT - Britive Permission Scopes Test Description"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// permission_scopes is only valid when consumer is "apps" and resources is ["*"].
				// Values are application types (e.g. AWS, Azure, GCP).
				Config: testAccCheckBritivePermissionScopesConfig(permissionName, permissionDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePermissionExists("britive_permission.with_scopes"),
				),
			},
		},
	})
}

func testAccCheckBritivePermissionScopesConfig(permissionName, permissionDescription string) string {
	return fmt.Sprintf(`
	resource "britive_permission" "with_scopes" {
		name        = "%s"
		description = "%s"
		consumer    = "apps"
		resources   = [
			"*",
		]
		actions     = [
			"apps.app.list",
			"apps.app.view",
		]
		permission_scopes = [
			"AWS",
			"Azure",
		]
	}`, permissionName, permissionDescription)
}

func TestBritivePermissionScopesInvalidConsumer(t *testing.T) {
	permissionName := "AT - Britive Permission Scopes Invalid Consumer Test"
	permissionDescription := "AT - Britive Permission Scopes Invalid Consumer Test Description"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// permission_scopes is rejected by the Britive API here because consumer
				// is "authz" instead of "apps" (the resource itself applies no such
				// restriction; the backend is the source of truth for this validation).
				Config:      testAccCheckBritivePermissionScopesInvalidConsumerConfig(permissionName, permissionDescription),
				ExpectError: regexp.MustCompile(`(?s).*`),
			},
		},
	})
}

func testAccCheckBritivePermissionScopesInvalidConsumerConfig(permissionName, permissionDescription string) string {
	return fmt.Sprintf(`
	resource "britive_permission" "invalid_scopes" {
		name        = "%s"
		description = "%s"
		consumer    = "authz"
		resources   = [
			"*",
		]
		actions     = [
			"authz.action.list",
			"authz.action.read",
		]
		permission_scopes = [
			"AWS",
			"Azure",
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
