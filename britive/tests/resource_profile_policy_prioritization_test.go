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
	profileName := "AT - New Britive Profile Policy Test"
	profilePolicyName := "AT - New Britive Profile Policy Test"
	profilePolicyDescription := "AT - New Britive Profile Policy Test Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfilePolicyPrioritizationConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy.new"),
					testAccCheckBritiveProfilePolicyPrioritizationExists("britive_profile_policy_prioritization.new_priority"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfilePolicyPrioritizationConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription string) string {
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
		members      = jsonencode(
			{
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
	}

	resource "britive_profile_policy_prioritization" "new_priority" {
    	profile_id = britive_profile.new_profile.id
		policy_priority {
      		id = britive_profile_policy.new.id
      		priority =0
    	}
	}
		`, applicationName, profileName, profilePolicyName, profilePolicyDescription)

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
