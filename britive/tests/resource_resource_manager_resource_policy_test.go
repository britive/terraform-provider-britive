package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceResourcePolicy(t *testing.T) {
	resourceLabelName1 := "AT-Britive_Resource_Manager_Test_Resource_Label-1"
	resourceLabelDescription1 := "AT-Britive_Resource_Manager_Test_Resource_Label_1_Description"
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceResourcePolicyConfig(resourceLabelName1, resourceLabelDescription1, timeOfAccessFrom, timeOfAccessTo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceResourcePolicyExists("britive_resource_manager_resource_label.resource_label_1"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_resource_policy.resource_policy_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceResourcePolicyConfig(resourceLabelName1, resourceLabelDescription1, timeOfAccessFrom, timeOfAccessTo string) string {
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

	resource "britive_resource_manager_resource_policy" "resource_policy_1" {
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
		access_level = "manage"
		consumer     = "resourcemanager"   
		is_active    = true
		is_draft     = false
		is_read_only = false
		resource_labels {
			label_key = britive_resource_manager_resource_label.resource_label_1.name
			values = ["Production"]
		}
	}

	`, resourceLabelName1, resourceLabelDescription1, timeOfAccessFrom, timeOfAccessTo)
}

func testAccCheckBritiveResourceResourcePolicyExists(n string) resource.TestCheckFunc {
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
