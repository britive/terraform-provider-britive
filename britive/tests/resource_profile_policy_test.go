package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveProfilePolicy(t *testing.T) {
	applicationName := "DO NOT DELETE - AWS TF Plugin"
	profileName := "AT - New Britive Profile Policy Test"
	profilePolicyName := "AT - New Britive Profile Policy Test"
	profilePolicyDescription := "AT - New Britive Profile Policy Test Description"
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePolicyConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription, timeOfAccessFrom, timeOfAccessTo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyExists("britive_profile_policy.new"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfilePolicyConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription, timeOfAccessFrom, timeOfAccessTo string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		expiration_duration = "25m0s"
		allow_impersonation  = true
		associations {
			type  = "EnvironmentGroup"
			value = "Development"
		}
		associations {
			type  = "EnvironmentGroup"
			value = "Stage"
		}
		associations {
			type  = "Environment"
			value = "Sigma Corporate"
		}
		tag_associations {
			key    = "team"
			values = ["engineering"]
		}
	}

	resource "britive_profile_policy" "new" {
		policy_name  = "%s"
		description  = "%s"
		profile_id   = britive_profile.new.id
		access_type  = "Allow"
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
					notificationMedium = "Email"
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
		consumer     = "papservice"
		is_active    = true
		is_draft     = false
		is_read_only = false
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
		associations {
			type  = "EnvironmentGroup"
			value = "Development"
		}
		associations {
			type  = "Environment"
			value = "Sigma Corporate"
		}
		tag_associations {
			key    = "team"
			values = ["engineering"]
		}
	}`, applicationName, profileName, profilePolicyName, profilePolicyDescription, timeOfAccessFrom, timeOfAccessTo)

}

func TestBritiveProfilePolicyTagAssociations(t *testing.T) {
	applicationName := "DO NOT DELETE - AWS TF Plugin"
	profileName := "AT - New Britive Profile Policy Tag Association Test"
	profilePolicyName := "AT - New Britive Profile Policy Tag Association Test"
	profilePolicyDescription := "AT - New Britive Profile Policy Tag Association Test Description"
	tagKey := "team"
	tagValue := "engineering"
	resourceName := "britive_profile_policy.new"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePolicyConfigWithTagAssociations(applicationName, profileName, profilePolicyName, profilePolicyDescription, tagKey, tagValue),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "policy_name", profilePolicyName),
					resource.TestCheckResourceAttr(resourceName, "tag_associations.#", "1"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfilePolicyConfigWithTagAssociations(applicationName, profileName, profilePolicyName, profilePolicyDescription, tagKey, tagValue string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id    = data.britive_application.app.id
		name                = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "EnvironmentGroup"
			value = "Development"
		}
		tag_associations {
			key    = "team"
			values = ["engineering"]
		}
	}

	resource "britive_profile_policy" "new" {
		policy_name  = "%s"
		description  = "%s"
		profile_id   = britive_profile.new.id
		access_type  = "Allow"
		consumer     = "papservice"
		is_active    = false
		is_draft     = false
		is_read_only = false
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
		tag_associations {
			key    = "%s"
			values = ["%s"]
		}
	}`, applicationName, profileName, profilePolicyName, profilePolicyDescription, tagKey, tagValue)
}

func testAccCheckBritiveProfilePolicyExists(n string) resource.TestCheckFunc {
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

// testAccProfilePolicyConfig renders britive_profile_policy.test, on a profile of its own,
// with the given members and condition (HCL expressions) and any extra config.
func testAccProfilePolicyConfig(profileName, policyName, description string, isActive bool, members, condition, extra string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "DO NOT DELETE - AWS TF Plugin"
	}

	resource "britive_profile" "policy_profile" {
		app_container_id    = data.britive_application.app.id
		name                = %q
		expiration_duration = "25m0s"
		associations {
			type  = "EnvironmentGroup"
			value = "Development"
		}
	}

	resource "britive_profile_policy" "test" {
		profile_id   = britive_profile.policy_profile.id
		policy_name  = %q
		description  = %q
		consumer     = "papservice"
		access_type  = "Allow"
		is_active    = %t
		is_draft     = false
		is_read_only = false
		members      = jsonencode(%s)
		condition    = jsonencode(%s)
	}
	%s`, profileName, policyName, description, isActive, members, condition, extra)
}

// testAccProfilePolicyEditConfig returns a name-based britive_profile_policy config.
func testAccProfilePolicyEditConfig(profileName, policyName string) testAccPolicyEditConfig {
	return func(description string, isActive bool, tags []string, timeToApprove int) string {
		return testAccProfilePolicyConfig(profileName, policyName, description, isActive,
			testAccPolicyMembersByName(tags...), testAccPolicyApprovalCondition(false, "", timeToApprove), "")
	}
}

// testAccProfilePolicyByIDConfig returns a britive_profile_policy config whose members and
// condition reference the tag, user and notification medium by ID.
func testAccProfilePolicyByIDConfig(profileName, emailMediumID string) func(description, policyName string) string {
	return func(description, policyName string) string {
		return testAccProfilePolicyConfig(profileName, policyName, description, true,
			testAccPolicyMembersByID, testAccPolicyApprovalCondition(true, emailMediumID, 30), testAccPolicyByIDDataSources())
	}
}

// testAccProfilePolicyDrift removes a member tag from the profile policy outside Terraform.
func testAccProfilePolicyDrift(t *testing.T, captured map[string]string) func() {
	return func() {
		client := testAccPolicyAPIClient(t)
		profileID, policyName := captured["profile_id"], captured["policy_name"]
		policy, err := client.GetProfilePolicyByName(profileID, policyName)
		if err != nil {
			t.Fatalf("could not read profile policy %s/%s: %v", profileID, policyName, err)
		}
		policy.ProfileID = profileID
		policy.Members = testAccDriftMembers(t, policy.Members)
		if _, err := client.UpdateProfilePolicy(*policy, policyName); err != nil {
			t.Fatalf("could not update profile policy %s/%s: %v", profileID, policyName, err)
		}
	}
}

// TestBritiveProfilePolicyEditMembersAndCondition updates description, is_active, members and condition
// of a name-based profile policy in place, with an empty plan after each apply.
func TestBritiveProfilePolicyEditMembersAndCondition(t *testing.T) {
	config := testAccProfilePolicyEditConfig("AT - Britive Profile Policy Edit Test", "AT - Britive Profile Policy Edit Test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicyEditFieldsSteps("britive_profile_policy.test", config),
	})
}

// TestBritiveProfilePolicyImportByName imports a name-based profile policy with both import
// ID formats and checks the imported values match the state.
func TestBritiveProfilePolicyImportByName(t *testing.T) {
	config := testAccProfilePolicyEditConfig("AT - Britive Profile Policy Import Test", "AT - Britive Profile Policy Import Test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicyImportByNameSteps("britive_profile_policy.test", config, testAccProfilePolicyImportID, "tag_associations"),
	})
}

// TestBritiveProfilePolicyDrift removes a member tag of a name-based profile policy
// outside Terraform: the next plan must show the change and applying must restore it.
func TestBritiveProfilePolicyDrift(t *testing.T) {
	testAccRequireAcc(t)
	n := "britive_profile_policy.test"
	captured := map[string]string{}
	config := testAccProfilePolicyEditConfig("AT - Britive Profile Policy Drift Test", "AT - Britive Profile Policy Drift Test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: testAccPolicyDriftSteps(config,
			testAccCaptureAttrs(n, captured, "profile_id", "policy_name"),
			testAccProfilePolicyDrift(t, captured)),
	})
}

// TestBritiveProfilePolicyMembersByID is the regression test for profile policies whose
// members and condition use IDs: create used to fail with "Provider produced inconsistent
// result after apply", and update with "PP-0005: Inconsistent request".
func TestBritiveProfilePolicyMembersByID(t *testing.T) {
	testAccRequireAcc(t)
	config := testAccProfilePolicyByIDConfig("AT - Britive Profile Policy By ID Test", testAccNotificationMediumID(t, "Email"))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicyByIDSteps("britive_profile_policy.test", "AT - Britive Profile Policy By ID Test", config, testAccProfilePolicyImportID, "tag_associations"),
	})
}

// TestBritiveProfilePolicyMembersByIDDrift removes a member tag of a by-ID profile policy
// outside Terraform: refresh reads it back by name, the plan must show the change, and
// applying the by-ID config must restore it with an empty plan after.
func TestBritiveProfilePolicyMembersByIDDrift(t *testing.T) {
	testAccRequireAcc(t)
	n := "britive_profile_policy.test"
	captured := map[string]string{}
	byID := testAccProfilePolicyByIDConfig("AT - Britive Profile Policy By ID Drift Test", testAccNotificationMediumID(t, "Email"))
	config := func(string, bool, []string, int) string {
		return testAccWithSecondTagByID(byID("By ID drift policy description", "AT - Britive Profile Policy By ID Drift Test"))
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: testAccPolicyDriftSteps(config,
			testAccCaptureAttrs(n, captured, "profile_id", "policy_name"),
			testAccProfilePolicyDrift(t, captured)),
	})
}

// TestBritiveProfilePolicySwitchNamesAndIDs switches a profile policy's members and
// condition from names to the same values by ID and back, each an in-place update.
func TestBritiveProfilePolicySwitchNamesAndIDs(t *testing.T) {
	testAccRequireAcc(t)
	profileName, policyName := "AT - Britive Profile Policy Switch Test", "AT - Britive Profile Policy Switch Test"
	byName := testAccProfilePolicyEditConfig(profileName, policyName)("Switch policy description", true, []string{testAccPolicyTag}, 30)
	byID := testAccProfilePolicyByIDConfig(profileName, testAccNotificationMediumID(t, "Email"))("Switch policy description", policyName)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    testAccPolicySwitchNamesAndIDsSteps("britive_profile_policy.test", byName, byID),
	})
}
