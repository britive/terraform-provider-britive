package britive

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveConstraint(t *testing.T) {
	applicationName := "DO NOT DELETE - GCP TF Plugin"
	profileName := "AT - New Britive Constraint Test"
	profileDescription := "AT - New Britive Constraint Test Description"
	associationValue := "britive-gdev-cis.net"
	permissionName := "BigQuery Data Owner"
	permissionType := "role"
	constraintType := "bigquery.datasets"
	constraintName := "my-first-project-310615.dataset2"
	permissionConditionName := "Storage Admin"
	constraintConditionType := "condition"
	constraintTitle := "ConditionConstraintType"
	constraintDescription := "Condition Constraint Type Description"
	constraintExpression := "request.time < timestamp('" + time.Now().AddDate(0, 0, 2).Format("2006-01-02T15:04:05Z07:00") + "')"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveConstraintConfig(applicationName, profileName, profileDescription, associationValue, permissionName, permissionType, constraintType, constraintName, permissionConditionName, constraintConditionType, constraintTitle, constraintDescription, constraintExpression),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveConstraintExists("britive_constraint.new"),
					testAccCheckBritiveConstraintExists("britive_constraint.new_condition"),
				),
			},
		},
	})
}

func testAccCheckBritiveConstraintConfig(applicationName, profileName, profileDescription, associationValue, permissionName, permissionType, constraintType, constraintName, permissionConditionName, constraintConditionType, constraintTitle, constraintDescription, constraintExpression string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "EnvironmentGroup"
			value = "%s"
		}
	}

	resource "britive_profile_permission" "new" {
		profile_id = britive_profile.new.id
		permission_name = "%s"
		permission_type = "%s"
	}

	resource "britive_constraint" "new" {
		profile_id = britive_profile.new.id
  		permission_name = britive_profile_permission.new.permission_name
		permission_type = britive_profile_permission.new.permission_type
  		constraint_type = "%s"
  		name = "%s"
	}

	resource "britive_profile_permission" "new_condition" {
		profile_id = britive_profile.new.id
		permission_name = "%s"
		permission_type = "%s"
	}

	resource "britive_constraint" "new_condition" {
    	profile_id      = britive_profile.new.id
		permission_name = britive_profile_permission.new_condition.permission_name
    	permission_type = britive_profile_permission.new_condition.permission_type
		constraint_type = "%s"
		title           = "%s"
    	description     = "%s"
    	expression      = "%s"
	}`, applicationName, profileName, profileDescription, associationValue, permissionName, permissionType, constraintType, constraintName, permissionConditionName, permissionType, constraintConditionType, constraintTitle, constraintDescription, constraintExpression)

}

func testAccCheckBritiveConstraintExists(n string) resource.TestCheckFunc {
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
