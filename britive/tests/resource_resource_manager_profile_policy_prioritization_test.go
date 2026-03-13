_package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceManagerProfilePolicyPrioritization(t *testing.T) {
	resourceLabelName1 := "AT-Britive_Resource_Manager_Test_Resource_Label-1"
	resourceLabelDescription1 := "AT-Britive_Resource_Manager_Test_Resource_Label_1_Description"
	resourceLabelName2 := "AT-Britive_Resource_Manager_Test_Resource_Label-2"
	resourceLabelDescription2 := "AT-Britive_Resource_Manager_Test_Resource_Label_2_Description"
	resourceProfileName := "AT-Britive_Resource_Manager_Test_Resource_Profile-1"
	resourceProfileDescription := "AT-Britive_Resource_Manager_Test_Resource_Profile_1_Description"
	profilePolicyName := "AT - New Britive Resource Manager Profile Policy for Prioritization Test"
	profilePolicyDescription := "AT - New Britive Resource Manager Profile Policy for Prioritization Test Description"
	profilePolicyNamePriority := 0
	profilePolicyName1 := "AT - New Britive Resource Manager Profile Policy for Prioritization Test 1"
	profilePolicyDescription1 := "AT - New Britive Resource Manager Profile Policy for Prioritization Test 1 Description"
	profilePolicyName1Priority := 1
	profilePolicyName2 := "AT - New Britive Resource Manager Profile Policy for Prioritization Test 2"
	profilePolicyDescription2 := "AT - New Britive Resource Manager Profile Policy for Prioritization Test 2 Description"
	profilePolicyName2Priority := 2

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePolicyPrioritizationConfig(
					resourceLabelName1,
					resourceLabelDescription1,
					resourceLabelName2,
					resourceLabelDescription2,
					resourceProfileName,
					resourceProfileDescription,
					profilePolicyName,
					profilePolicyDescription,
					profilePolicyName1,
					profilePolicyDescription1,
					profilePolicyName2,
					profilePolicyDescription2,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePolicyPrioritizationExists("britive_resource_manager_profile_policy.resource_profile_policy"),
					testAccCheckBritiveResourceManagerProfilePolicyPrioritizationExists("britive_resource_manager_profile_policy.resource_profile_policy_1"),
					testAccCheckBritiveResourceManagerProfilePolicyPrioritizationExists("britive_resource_manager_profile_policy.resource_profile_policy_2"),
					testAccCheckBritiveResourceManagerProfilePolicyPrioritizationExists("britive_resource_manager_profile_policy_prioritization.priority_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfilePolicyPrioritizationConfig(applicationName, profileName, profilePolicyName, profilePolicyDescription, profilePolicyName1, profilePolicyDescription1, profilePolicyName2, profilePolicyDescription2 string, profilePolicyNamePriority, profilePolicyName1Priority, profilePolicyName2Priority int) string {
	return fmt.Sprintf(`
resource "britive_resource_manager_resource_label" "resource_label_1" {
		name         = "%s"
		description  = "%s"
		label_color  = "#abc123"

		values {
			name = "Production"
			description = "Production Desc"
		}
		values {
			name = "Development"
			description = "Development Desc"
		}
	}

	resource "britive_resource_manager_resource_label" "resource_label_2" {
		name         = "%s"
		description  = "%s"
		label_color  = "#1a2b3c"

		values {
			name = "us-east-1"
			description = "us-east-1 Desc"
		}
		values {
			name = "eu-west-1"
			description = "eu-west-1 Desc"
		}
	}

	resource "britive_resource_manager_profile" "resource_profile_1" {
		name                 = "%s"
		description          = "%s"
		expiration_duration  = 3600000
		allow_impersonation  = true

		associations {
			label_key   = britive_resource_manager_resource_label.resource_label_1.name
			values = ["Production", "Development"]
		}
		associations {
			label_key   = britive_resource_manager_resource_label.resource_label_2.name
			values = ["us-east-1", "eu-west-1"]
		}
	}

	resource "britive_resource_manager_profile_policy" "resource_profile_policy" {
		profile_id   = britive_resource_manager_profile.resource_profile_1.id
		policy_name  = "%s"
		description  = "%s"
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
		access_type  = "Allow"
		consumer     = "resourceprofile"   
		is_active    = true
		is_draft     = false
		is_read_only = false
		resource_labels {
			label_key = britive_resource_manager_resource_label.resource_label_2.name
			values = ["us-east-1"]
		}
	}
	resource "britive_resource_manager_profile_policy" "resource_profile_policy_1" {
		profile_id   = britive_resource_manager_profile.resource_profile_1.id
		policy_name  = "%s"
		description  = "%s"
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
		access_type  = "Allow"
		consumer     = "resourceprofile"   
		is_active    = true
		is_draft     = false
		is_read_only = false
		resource_labels {
			label_key = britive_resource_manager_resource_label.resource_label_2.name
			values = ["us-east-1"]
		}
	}
	resource "britive_resource_manager_profile_policy" "resource_profile_policy_2" {
		profile_id   = britive_resource_manager_profile.resource_profile_1.id
		policy_name  = "%s"
		description  = "%s"
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
		access_type  = "Allow"
		consumer     = "resourceprofile"   
		is_active    = true
		is_draft     = false
		is_read_only = false
		resource_labels {
			label_key = britive_resource_manager_resource_label.resource_label_2.name
			values = ["us-east-1"]
		}
	}
	resource "britive_resource_manager_profile_policy_prioritization" "priority_1" {
		profile_id = britive_resource_manager_profile.resource_profile_1.id
		policy_priority {
			id = britive_resource_manager_profile_policy.resource_profile_policy_2.id
			priority = 0
		}
		policy_priority {
			id = britive_resource_manager_profile_policy.resource_profile_policy.id
			priority = 1
		}
	}
`, resourceLabelName1, resourceLabelDescription, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription, profilePolicyName, profilePolicyDescription, profilePolicyName1, profilePolicyDescription1, profilePolicyName2, profilePolicyDescription2)
}

func testAccCheckBritiveResourceManagerProfilePolicyPrioritizationExists(n string) resource.TestCheckFunc {
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
