package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

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
			// Step 1: Create profile and policies with initial priorities
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
			// Step 2: Update policy priorities
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
					2,
					0,
					1,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"britive_profile_policy_prioritization.new_priority", "policy_priority.0.priority",
						"britive_profile_policy.new", "id",
					),
					resource.TestCheckResourceAttrPair(
						"britive_profile_policy_prioritization.new_priority", "policy_priority.1.priority",
						"britive_profile_policy.new_1", "id",
					),
					resource.TestCheckResourceAttrPair(
						"britive_profile_policy_prioritization.new_priority", "policy_priority.2.priority",
						"britive_profile_policy.new_2", "id",
					),
				),
			},
			// Step 3: Plan-only check (should be no diff)
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

func testAccCheckBritiveProfilePolicyPrioritizationConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription, profilePolicyName1, profilePolicyDescription1, profilePolicyName2, profilePolicyDescription2 string, profilePolicyNamePriority, profilePolicyName1Priority, profilePolicyName2Priority int) string {
	return fmt.Sprintf(`
data "britive_application" "app" {
  name = "%s"
}

resource "britive_profile" "new_profile" {
  app_container_id   = data.britive_application.app.id
  name               = "%s"
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
  members      = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" }
    ]
  })
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
  members      = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" }
    ]
  })
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
  members      = jsonencode({
    users = [
      { name = "britiveprovideracceptancetest" },
      { name = "britiveprovideracceptancetest1" }
    ]
  })
}

resource "britive_profile_policy_prioritization" "new_priority" {
  profile_id = britive_profile.new_profile.id

  policy_priority {
    id       = britive_profile_policy.new.id
    priority = %d
  }

  policy_priority {
    id       = britive_profile_policy.new_1.id
    priority = %d
  }

  policy_priority {
    id       = britive_profile_policy.new_2.id
    priority = %d
  }
}
`, applicationName, profileName, profilePolicyName, profilePolicyDescription, profilePolicyName1, profilePolicyDescription1, profilePolicyName2, profilePolicyDescription2, profilePolicyNamePriority, profilePolicyName1Priority, profilePolicyName2Priority)
}

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
