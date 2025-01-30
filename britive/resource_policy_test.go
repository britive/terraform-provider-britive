package britive

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritivePolicy(t *testing.T) {
	permissionName := "AT - Britive Permission Test Policy"
	permissionDescription := "AT - Britive Permission Test Policy Description"
	roleName := "AT - Britive Role Test Policy"
	roleDescription := "AT - Britive Role Test Policy Description"
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
	resource "britive_permission" "new_permission_policy_1" {
		name = "%s1"
		description = "%s1"
		consumer    = "secretmanager"
		resources   = [
			"*",
		]
		actions     = [
			"authz.policy.list",
			"authz.policy.read",
			"sm.secret.read",
		]
	}

	resource "britive_permission" "new_permission_policy_2" {
		name = "%s2"
		description = "%s2"
		consumer    = "authz"
		resources   = [
			"*",
		]
		actions     = [
			"authz.action.list",
			"authz.action.read",
		]
	}

	resource "britive_role" "new_role_policy_1" {
		name = "%s1"
		description = "%s1"
		permissions = jsonencode(
			[
				{
					name = britive_permission.new_permission_policy_1.name
				},
				{
					name = britive_permission.new_permission_policy_2.name
				}
			]
		)
	}

	resource "britive_role" "new_role_policy_2" {
		name = "%s2"
		description = "%s2"
		permissions = jsonencode(
			[
				{
					name = britive_permission.new_permission_policy_1.name
				},
				{
					name = britive_permission.new_permission_policy_2.name
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
					{
						name = "britiveProviderAcceptanceTestSI1"
					},
				]
				tags              = [
					{
						name = "britiveProviderAcceptanceTestTag"
					},
					{
						name = "britiveProviderAcceptanceTestTag1"
					},
				]
				tokens            = [
					{
						name = "britiveProviderAcceptanceTestToken"
					},
					{
						name = "britiveProviderAcceptanceTestToken1"
					},
				]
				users             = [
					{
						name = "britiveprovideracceptancetest"
					},
					{
						name = "britiveprovideracceptancetest1"
					},
				]
			}
		)
		permissions  = jsonencode(
			[
				{
					name = britive_permission.new_permission_policy_1.name
				},
				{
					name = britive_permission.new_permission_policy_2.name
				},
			]
		)
		roles        = jsonencode(
			[
				{
					name = britive_role.new_role_policy_1.name
				},
				{
					name = britive_role.new_role_policy_2.name
				},
			]
		)
		condition    = jsonencode(
			{
				approval     = {
					approvers          = {
						tags    = [
							"britiveProviderAcceptanceTestTag",
							"britiveProviderAcceptanceTestTag1",
						]
						userIds = [
							"britiveprovideracceptancetest",
							"britiveprovideracceptancetest1",
						]
					}
					notificationMedium = ["Email"]
					timeToApprove      = 30
					isValidForInDays   = false
					validFor           = 120
				}
				ipAddress    = "192.162.0.0/16,10.10.0.10"
				timeOfAccess = {
					"dateSchedule": {
						"fromDate": "%s",
						"toDate": "%s",
						"timezone": "Asia/Calcutta"
					},
					"daysSchedule": {
						"fromTime": "17:00:00",
						"toTime": "17:30:00",
						"timezone": "Asia/Calcutta",
						"days": [
							"FRIDAY",
							"SATURDAY",
							"SUNDAY"
						]
					}
				}
			}
		)
		is_active    = true
		is_draft     = false
		is_read_only = false
	}`, permissionName, permissionDescription, permissionName, permissionDescription, roleName, roleDescription, roleName, roleDescription, policyName, policyDescription, timeOfAccessFrom, timeOfAccessTo)

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
