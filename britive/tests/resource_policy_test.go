package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				aiIdentities      = [
        			{
          				name : "DO_NOT_DELETE_AI_IDENTITY"
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
					managerApproval = {
						condition = "All",
						required = true
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
			return errs.NewNotFoundErrorf("%s in state", n)
		}

		if rs.Primary.ID == "" {
			return errs.NewNotFoundErrorf("ID for %s in state", n)
		}

		return nil
	}
}

// testAccPolicyConfig renders britive_policy.test, with a permission and a role of its own,
// and the given members, condition, permissions and roles (HCL expressions) and any extra
// config.
func testAccPolicyConfig(policyName, description string, isActive bool, members, condition, permissions, roles, extra string) string {
	return fmt.Sprintf(`
	resource "britive_permission" "policy_permission" {
		name        = "%[1]s Permission"
		description = "%[1]s Permission Description"
		consumer    = "secretmanager"
		resources   = ["*"]
		actions     = ["authz.policy.list", "sm.secret.read"]
	}

	resource "britive_role" "policy_role" {
		name        = "%[1]s Role"
		description = "%[1]s Role Description"
		permissions = jsonencode([{ name = britive_permission.policy_permission.name }])
	}

	resource "britive_policy" "test" {
		name         = %[1]q
		description  = %[2]q
		access_type  = "Allow"
		is_active    = %[3]t
		is_draft     = false
		is_read_only = false
		members      = jsonencode(%[4]s)
		condition    = jsonencode(%[5]s)
		permissions  = jsonencode(%[6]s)
		roles        = jsonencode(%[7]s)
	}
	%[8]s`, policyName, description, isActive, members, condition, permissions, roles, extra)
}

// The permissions and roles of the britive_policy configs, by name and by ID. The
// resource IDs are "permissions/{id}" and "roles/{id}".
const (
	testAccPolicyPermissionsByName = `[{ name = britive_permission.policy_permission.name }]`
	testAccPolicyPermissionsByID   = `[{ id = element(split("/", britive_permission.policy_permission.id), 1) }]`
	testAccPolicyRolesByName       = `[{ name = britive_role.policy_role.name }]`
	testAccPolicyRolesByID         = `[{ id = element(split("/", britive_role.policy_role.id), 1) }]`
)

// testAccPolicyEditConfigFor returns a name-based britive_policy config. The permission and
// role names derive from policyName, so policyName must not change between steps.
func testAccPolicyEditConfigFor(policyName string) testAccPolicyEditConfig {
	return func(description string, isActive bool, tags []string, timeToApprove int) string {
		return testAccPolicyConfig(policyName, description, isActive,
			testAccPolicyMembersByName(tags...), testAccPolicyApprovalCondition(false, "", timeToApprove),
			testAccPolicyPermissionsByName, testAccPolicyRolesByName, "")
	}
}

// testAccPolicyByIDConfigFor returns a britive_policy config whose members, condition,
// permissions and roles reference everything by ID. The permission and role names derive
// from baseName, which stays fixed when the policy is renamed.
func testAccPolicyByIDConfigFor(baseName, emailMediumID string) func(description, policyName string) string {
	return func(description, policyName string) string {
		config := testAccPolicyConfig(baseName, description, true,
			testAccPolicyMembersByID, testAccPolicyApprovalCondition(true, emailMediumID, 30),
			testAccPolicyPermissionsByID, testAccPolicyRolesByID, testAccPolicyByIDDataSources())
		// Rename only the policy, not the permission and role
		return strings.Replace(config, fmt.Sprintf("name         = %q", baseName), fmt.Sprintf("name         = %q", policyName), 1)
	}
}

// testAccPolicyDrift removes a member tag from the policy outside Terraform.
func testAccPolicyDrift(t *testing.T, captured map[string]string) func() {
	return func() {
		client := testAccPolicyAPIClient(t)
		name := captured["name"]
		policy, err := client.GetPolicyByName(name)
		if err != nil {
			t.Fatalf("could not read policy %s: %v", name, err)
		}
		policy.Members = testAccDriftMembers(t, policy.Members)
		if _, err := client.UpdatePolicy(*policy, name); err != nil {
			t.Fatalf("could not update policy %s: %v", name, err)
		}
	}
}

// TestBritivePolicyEditMembersAndCondition updates description, is_active, members and
// condition of a name-based policy in place, with an empty plan after each apply.
func TestBritivePolicyEditMembersAndCondition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicyEditFieldsSteps("britive_policy.test", testAccPolicyEditConfigFor("AT - Britive Policy Edit Test")),
	})
}

// TestBritivePolicyImportByName imports a name-based policy with both import ID formats
// and checks the imported values match the state.
func TestBritivePolicyImportByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicyImportByNameSteps("britive_policy.test", testAccPolicyEditConfigFor("AT - Britive Policy Import Test"), testAccPolicyImportID),
	})
}

// TestBritivePolicyDrift removes a member tag of a name-based policy outside Terraform:
// the next plan must show the change and applying must restore it.
func TestBritivePolicyDrift(t *testing.T) {
	testAccRequireAcc(t)
	captured := map[string]string{}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: testAccPolicyDriftSteps(testAccPolicyEditConfigFor("AT - Britive Policy Drift Test"),
			testAccCaptureAttrs("britive_policy.test", captured, "name"),
			testAccPolicyDrift(t, captured)),
	})
}

// TestBritivePolicyMembersByID is the regression test for policies whose members,
// condition, permissions and roles use IDs: create used to fail with "Provider produced
// inconsistent result after apply", and update with "PP-0005: Inconsistent request".
func TestBritivePolicyMembersByID(t *testing.T) {
	testAccRequireAcc(t)
	const name = "AT - Britive Policy By ID Test"
	config := testAccPolicyByIDConfigFor(name, testAccNotificationMediumID(t, "Email"))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicyByIDSteps("britive_policy.test", name, config, testAccPolicyImportID),
	})
}

// TestBritivePolicyMembersByIDDrift removes a member tag of a by-ID policy outside
// Terraform: refresh reads it back by name, the plan must show the change, and applying
// the by-ID config must restore it with an empty plan after.
func TestBritivePolicyMembersByIDDrift(t *testing.T) {
	testAccRequireAcc(t)
	const name = "AT - Britive Policy By ID Drift Test"
	captured := map[string]string{}
	byID := testAccPolicyByIDConfigFor(name, testAccNotificationMediumID(t, "Email"))
	config := func(string, bool, []string, int) string {
		return testAccWithSecondTagByID(byID("By ID drift policy description", name))
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: testAccPolicyDriftSteps(config,
			testAccCaptureAttrs("britive_policy.test", captured, "name"),
			testAccPolicyDrift(t, captured)),
	})
}

// TestBritivePolicySwitchNamesAndIDs switches a policy's members, condition, permissions
// and roles from names to the same values by ID and back, each an in-place update.
func TestBritivePolicySwitchNamesAndIDs(t *testing.T) {
	testAccRequireAcc(t)
	const name = "AT - Britive Policy Switch Test"
	byName := testAccPolicyEditConfigFor(name)("Switch policy description", true, []string{testAccPolicyTag}, 30)
	byID := testAccPolicyByIDConfigFor(name, testAccNotificationMediumID(t, "Email"))("Switch policy description", name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicySwitchNamesAndIDsSteps("britive_policy.test", byName, byID),
	})
}
