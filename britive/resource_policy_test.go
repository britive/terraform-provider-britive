package britive

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritivePolicy(t *testing.T) {
	permissionName := "AT - Britive Permission Test"
	permissionDescription := "AT - Britive Permission Test Description"
	roleName := "AT - Britive Role Test"
	roleDescription := "AT - Britive Role Test Description"
	policyName := "AT - Britive Policy Test"
	policyDescription := "AT - Britive Policy Test Description"
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePolicyConfig(permissionName, permissionDescription, roleName, roleDescription, policyName, policyDescription, timeOfAccessFrom, timeOfAccessTo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePolicyExists("britive_policy.new"),
				),
			},
		},
	})
}

func testAccCheckBritivePolicyConfig(permissionName, permissionDescription, roleName, roleDescription, policyName, policyDescription, timeOfAccessFrom, timeOfAccessTo string) string {
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
	}

	resource "britive_role" "new" {
		name = "%s"
		description = "%s"
		permissions = jsonencode(
			[
				{
					name = "UserViewPermission"
				},
				{
					name = britive_permission.new.name
				}
			]
		)
	}

	resource "britive_policy" "new" {
		name         = "%s"
		description  = "%s"
		access_type  = "Allow"
		members      = jsonencode(
			{
				serviceIdentities = [
					{
						name = "britiveProviderAcceptanceTestSI"
					},
				]
				tags              = [
					{
						name = "britiveProviderAcceptanceTestTag"
					},
				]
				tokens            = [
					{
						name = "britiveProviderAcceptanceTestToken"
					},
				]
				users             = [
					{
						name = "britiveprovideracceptancetest"
					},
				]
			}
		)
		permissions  = jsonencode(
			[
				{
					name = britive_permission.new.name
				},
			]
		)
		roles        = jsonencode(
			[
				{
					name = britive_role.new.name
				},
			]
		)
		condition    = jsonencode(
			{
				approval     = {
					approvers          = {
						tags    = [
							"britiveProviderAcceptanceTestTag",
						]
						userIds = [
							"britiveprovideracceptancetest",
						]
					}
					notificationMedium = "Email"
					timeToApprove      = 30
					isValidForInDays   = true
					validFor           = 2
				}
				ipAddress    = "192.162.0.0/16,10.10.0.10"
				timeOfAccess = {
					from = "%s"
					to   = "%s"
				}
			}
		)
		is_active    = true
		is_draft     = false
		is_read_only = false
	}`, permissionName, permissionDescription, roleName, roleDescription, policyName, policyDescription, timeOfAccessFrom, timeOfAccessTo)

}

func testAccCheckBritivePolicyExists(n string) resource.TestCheckFunc {
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
