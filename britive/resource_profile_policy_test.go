package britive

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveProfilePolicy(t *testing.T) {
	applicationName := "AWS-ValueLabs"
	profileName := "AT - Britive Profile Test"
	profilePolicyName := "AT - Britive Profile Policy Test"
	profilePolicyDescription := "AT - Britive Profile Policy Test Description"
	timeOfAccessFrom := time.Now().AddDate(0, 0, 2).Format("2006-01-02 15:04:05")
	timeOfAccessTo := time.Now().AddDate(0, 0, 7).Format("2006-01-02 15:04:05")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
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
		associations {
			type  = "EnvironmentGroup"
			value = "QA"
		}
	}

	resource "britive_profile_policy" "new" {
		policy_name  = "%s"
		description  = "%s"
		profile_id   = britive_profile.new.id
		access_type  = "Allow"
		condition    = jsonencode(
			{
				ipAddress    = "192.168.0.0/16,10.10.0.10"
				timeOfAccess = {
					from = "%s"
					to   = "%s"
				}
				approval     = {
					approvers          = {
						userIds = [
							"britiveprovideracceptancetest",
						]
						tags    = [
							"britiveProviderAcceptanceTestTag",
						]
					}
					timeToApprove      = 30
					notificationMedium = "Email"
					validFor = 240
					isValidForInDays = false
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
	}`, applicationName, profileName, profilePolicyName, profilePolicyDescription, timeOfAccessFrom, timeOfAccessTo)

}

func testAccCheckBritiveProfilePolicyExists(n string) resource.TestCheckFunc {
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
