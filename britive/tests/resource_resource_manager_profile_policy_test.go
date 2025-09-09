package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceManagerProfilePolicy(t *testing.T) {
	resourceLabelName1 := "AT-Britive_Resource_Manager_Test_Resource_Label-1"
	resourceLabelDescription1 := "AT-Britive_Resource_Manager_Test_Resource_Label_1_Description"
	resourceLabelName2 := "AT-Britive_Resource_Manager_Test_Resource_Label-2"
	resourceLabelDescription2 := "AT-Britive_Resource_Manager_Test_Resource_Label_2_Description"
	resourceProfileName := "AT-Britive_Resource_Manager_Test_Resource_Profile-1"
	resourceProfileDescription := "AT-Britive_Resource_Manager_Test_Resource_Profile_1_Description"
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePolicyConfig(resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription, timeOfAccessFrom, timeOfAccessTo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_resource_label.resource_label_1"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_resource_label.resource_label_2"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_profile.resource_profile_1"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_profile_policy.resource_profile_policy_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfilePolicyConfig(resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription, timeOfAccessFrom, timeOfAccessTo string) string {
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

		associations {
			label_key   = britive_resource_manager_resource_label.resource_label_1.name
			values = ["Production", "Development"]
		}
		associations {
			label_key   = britive_resource_manager_resource_label.resource_label_2.name
			values = ["us-east-1", "eu-west-1"]
		}
	}

	resource "britive_resource_manager_profile_policy" "resource_profile_policy_1" {
		profile_id   = britive_resource_manager_profile.resource_profile_1.id
		policy_name  = "AT-Britive_Resource_Manager_Test_Resource_Profile-Policy-1"
		description  = "AT-Britive_Resource_Manager_Test_Resource_Profile_Policy_Description-1"
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
			}
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

	`, resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription, timeOfAccessFrom, timeOfAccessTo)
}

func testAccCheckBritiveResourceManagerProfilePolicyExists(n string) resource.TestCheckFunc {
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
