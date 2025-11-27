package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestBritiveProfilePolicyPrioritization runs the policy prioritization acceptance test
func TestBritiveProfilePolicyPrioritization(t *testing.T) {
	applicationName := "DO NOT DELETE - AWS TF Plugin"
	profileName := "AT - New Britive Profile Policy for Prioritization Test"
	profilePolicyName := "AT - New Britive Profile Policy for Prioritization Test"
	profilePolicyDescription := "AT - New Britive Profile Policy for Prioritization Test Description"
	profilePolicyNamePriority := 0
	profilePolicyName1 := "AT - New Britive Profile Policy for Prioritization Test 1"
	profilePolicyDescription1 := "AT - New Britive Profile Policy for Prioritization Test 1 Description"
	profilePolicyName1Priority := 1
	profilePolicyName2 := "AT - New Britive Profile Policy for Prioritization Test 2"
	profilePolicyDescription2 := "AT - New Britive Profile Policy for Prioritization Test 2 Description"
	profilePolicyName2Priority := 2

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Initial creation
			{
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(
					applicationName,
					profileName,
					profilePolicyName,
					profilePolicyDescription,
					profilePolicyName1,
					profilePolicyDescription1,
					profilePolicyName2,
					profilePolicyDescription2,
					profilePolicyNamePriority,
					profilePolicyName1Priority,
					profilePolicyName2Priority,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy.new"),
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy.new_1"),
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy.new_2"),
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy_prioritization.new_priority"),
				),
			},
			// Update priorities safely for TypeSet
			{
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(
					applicationName,
					profileName,
					profilePolicyName,
					profilePolicyDescription1,
					profilePolicyName1,
					profilePolicyDescription2,
					profilePolicyName2,
					profilePolicyDescription2,
					2, 0, 1,
				),
				Check: testAccCheckPolicyPriorities(map[string]int{
					"profile_policy_new":   2,
					"profile_policy_new_1": 0,
					"profile_policy_new_2": 1,
				}),
			},
			// PLAN should be empty
			{
				PlanOnly: true,
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(
					applicationName,
					profileName,
					profilePolicyName,
					profilePolicyDescription,
					profilePolicyName1,
					profilePolicyDescription1,
					profilePolicyName2,
					profilePolicyDescription2,
					profilePolicyNamePriority,
					profilePolicyName1Priority,
					profilePolicyName2Priority,
				),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// testAccCheckPolicyPriorities checks policy_priority by ID, safe for TypeSet
func testAccCheckPolicyPriorities(expected map[string]int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["britive_profile_policy_prioritization.new_priority"]
		if !ok {
			return fmt.Errorf("resource not found")
		}

		for idKey, expPriority := range expected {
			found := false
			for k, v := range rs.Primary.Attributes {
				if strings.HasSuffix(k, ".id") && v == idKey {
					index := strings.TrimSuffix(strings.TrimPrefix(k, "policy_priority."), ".id")
					priorityKey := fmt.Sprintf("policy_priority.%s.priority", index)
					if rs.Primary.Attributes[priorityKey] != fmt.Sprintf("%d", expPriority) {
						return fmt.Errorf("priority for %s: expected %d, got %s", idKey, expPriority, rs.Primary.Attributes[priorityKey])
					}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("policy %s not found in resource", idKey)
			}
		}
		return nil
	}
}

// testAccCheckBritiveProfilePolicyPrioritizationConfig returns the Terraform config as string
func testAccCheckBritiveProfilePolicyPrioritizationConfig(applicationName, profileName, policy1Name, policy1Desc, policy2Name, policy2Desc, policy3Name, policy3Desc string, priority1, priority2, priority3 int) string {
	membersJSON, _ := json.Marshal(map[string][]map[string]string{
		"users": {
			{"name": "britiveprovideracceptancetest"},
			{"name": "britiveprovideracceptancetest1"},
		},
	})

	return fmt.Sprintf(`
data "britive_application" "app" {
  name = "%s"
}

resource "britive_profile" "new_profile" {
	app_container_id = data.britive_application.app.id
	name = "%s"
	expiration_duration = "25m0s"
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
}

resource "britive_profile_policy" "new" {
  policy_name  = "%s"
  description  = "%s"
  profile_id   = britive_profile.new_profile.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode(%s)
}

resource "britive_profile_policy" "new_1" {
  policy_name  = "%s"
  description  = "%s"
  profile_id   = britive_profile.new_profile.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode(%s)
}

resource "britive_profile_policy" "new_2" {
  policy_name  = "%s"
  description  = "%s"
  profile_id   = britive_profile.new_profile.id
  access_type  = "Allow"
  consumer     = "papservice"
  is_active    = true
  is_draft     = false
  is_read_only = false
  members      = jsonencode(%s)
}

resource "britive_profile_policy_prioritization" "new_priority" {
  profile_id = britive_profile.new_profile.id

  policy_priority {
    id       = "profile_policy_new"
    priority = %d
  }
  policy_priority {
    id       = "profile_policy_new_1"
    priority = %d
  }
  policy_priority {
    id       = "profile_policy_new_2"
    priority = %d
  }
}
`, applicationName, profileName,
		policy1Name, policy1Desc,
		string(membersJSON),
		policy2Name, policy2Desc,
		string(membersJSON),
		policy3Name, policy3Desc,
		string(membersJSON),
		priority1, priority2, priority3,
	)
}

// testAccCheckBritiveProfilePolicyPrioritizationExists checks resource exists in state
func testAccCheckBritiveProfilePolicyPrioritizationExists(n string) resource.TestCheckFunc {
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
