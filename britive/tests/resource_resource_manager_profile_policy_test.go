package tests

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePolicyConfig(resourceLabelName1, resourceLabelDescription1, resourceLabelName2, resourceLabelDescription2, resourceProfileName, resourceProfileDescription, timeOfAccessFrom, timeOfAccessTo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_resource_label.resource_label_1"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_resource_label.resource_label_2"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_profile.resource_profile_1"),
					testAccCheckBritiveResourceManagerProfilePolicyExists("britive_resource_manager_profile_policy.resource_profile_policy_1"),
					// policy_name contains spaces - positive case for the
					// AlphanumericWithSpaces validator, which must accept them.
					resource.TestCheckResourceAttr("britive_resource_manager_profile_policy.resource_profile_policy_1", "policy_name", "AT-Britive Resource Manager Test Resource Profile-Policy-1"),
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
		name                             = "%s"
		description                      = "%s"
		expiration_duration              = 10800000
		extendable                       = true
		notification_prior_to_expiration = "1h0m0s"
		extension_duration               = "2h0m0s"
		extension_limit                  = 2
		allow_impersonation              = true

		associations {
			label_key = britive_resource_manager_resource_label.resource_label_1.name
			values    = ["Production", "Development"]
		}
		associations {
			label_key = britive_resource_manager_resource_label.resource_label_2.name
			values    = ["us-east-1", "eu-west-1"]
		}
	}

	resource "britive_resource_manager_profile_policy" "resource_profile_policy_1" {
		profile_id   = britive_resource_manager_profile.resource_profile_1.id
		policy_name  = "AT-Britive Resource Manager Test Resource Profile-Policy-1"
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
				aiIdentities      = [
        			{
          				name : "DO_NOT_DELETE_AI_IDENTITY"
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

// TestBritiveResourceManagerProfilePolicyInvalidName is a negative test: policy_name
// values with characters outside the allowed set (letters, numbers, spaces, hyphens,
// underscores) must be rejected at plan time by the AlphanumericWithSpaces schema
// validator, before any API call is made. profile_id is a placeholder since the
// plan never reaches apply.
func TestBritiveResourceManagerProfilePolicyInvalidName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckBritiveResourceManagerProfilePolicyInvalidNameConfig(),
				ExpectError: regexp.MustCompile(`(?s).*Invalid Characters.*`),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfilePolicyInvalidNameConfig() string {
	return `
	resource "britive_resource_manager_profile_policy" "invalid_name" {
		profile_id  = "placeholder-profile-id"
		policy_name = "Sudo Session - Recorded (Directory)"
	}
	`
}

// TestBritiveResourceManagerProfilePolicyMembersByID creates a resource manager profile
// policy whose members reference the tag and user by ID. Create must leave members as
// configured (no "Provider produced inconsistent result after apply") with an empty plan
// after refresh, and the policy must import with both import ID formats.
func TestBritiveResourceManagerProfilePolicyMembersByID(t *testing.T) {
	n := "britive_resource_manager_profile_policy.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: append([]resource.TestStep{
			{
				Config: testAccResourceManagerProfilePolicyByIDConfig("AT-Britive_RM_Profile_Policy_By_ID_Label", "AT-Britive_RM_Profile_Policy_By_ID_Profile", "AT-Britive RM Profile Policy By ID"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAttrContainsAttrOf(n, "members", "data.britive_tag.policy_tag", "tag_id"),
					testAccCheckAttrContainsAttrOf(n, "members", "data.britive_user.policy_user", "user_id"),
				),
			},
			// profile_id is configured as the profile's resource ID ("resource-manager/profile/{id}")
			// and kept as configured, while import only has the bare profile ID.
		}, testAccImportSteps(n, testAccResourceManagerProfilePolicyImportID, map[string]string{}, "profile_id")...),
	})
}

func testAccResourceManagerProfilePolicyByIDConfig(labelName, profileName, policyName string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_label" "policy_label" {
		name        = %q
		description = "Label for the by-ID profile policy test"
		label_color = "#abc123"

		values {
			name = "Production"
		}
	}

	resource "britive_resource_manager_profile" "policy_profile" {
		name                = %q
		description         = "Profile for the by-ID profile policy test"
		expiration_duration = 10800000
		associations {
			label_key = britive_resource_manager_resource_label.policy_label.name
			values    = ["Production"]
		}
	}

	resource "britive_resource_manager_profile_policy" "test" {
		profile_id   = britive_resource_manager_profile.policy_profile.id
		policy_name  = %q
		description  = "By ID profile policy description"
		members      = jsonencode(%s)
		condition    = jsonencode({ ipAddress = "192.162.0.0/16,10.10.0.10" })
		access_type  = "Allow"
		consumer     = "resourceprofile"
		is_active    = true
		is_draft     = false
		is_read_only = false
		resource_labels {
			label_key = britive_resource_manager_resource_label.policy_label.name
			values    = ["Production"]
		}
	}
	%s`, labelName, profileName, policyName, testAccPolicyMembersByID, testAccPolicyByIDDataSources())
}
