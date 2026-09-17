package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveScheduleScanDaily(t *testing.T) {
	resourceTypeName := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Daily"
	resourceTypeDescription := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Daily_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveScheduleScanDailyConfig(resourceTypeName, resourceTypeDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_ss_daily"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_daily"),
				),
			},
		},
	})
}

func TestBritiveScheduleScanWeekly(t *testing.T) {
	resourceTypeName := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Weekly"
	resourceTypeDescription := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Weekly_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveScheduleScanWeeklyConfig(resourceTypeName, resourceTypeDescription, "Monday", "11:00", "Val1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_ss_weekly"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_weekly"),
				),
			},
			// In-place update: day_of_week (abbreviation this time), start_time, and
			// resource_labels all change without replacing the resource.
			{
				Config: testAccCheckBritiveScheduleScanWeeklyConfig(resourceTypeName, resourceTypeDescription, "Fri", "23:45", "Val2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_ss_weekly"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_weekly"),
				),
			},
		},
	})
}

func TestBritiveScheduleScanMonthly(t *testing.T) {
	resourceTypeName := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Monthly"
	resourceTypeDescription := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Monthly_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveScheduleScanMonthlyConfig(resourceTypeName, resourceTypeDescription, 6),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_ss_monthly"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_monthly"),
				),
			},
			// In-place update: day_of_month changes without replacing the resource.
			{
				Config: testAccCheckBritiveScheduleScanMonthlyConfig(resourceTypeName, resourceTypeDescription, 15),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_ss_monthly"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_monthly"),
				),
			},
		},
	})
}

// TestBritiveResourceTypeScanEnabled exercises scan_enabled on
// britive_resource_manager_resource_type across an Update (false -> true), alongside a
// sibling britive_resource_manager_resource_type_schedule_scan.
func TestBritiveResourceTypeScanEnabled(t *testing.T) {
	resourceTypeName := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Scan_Enabled"
	resourceTypeDescription := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Scan_Enabled_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypeScanEnabledConfig(resourceTypeName, resourceTypeDescription, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_scan_enabled"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_for_enabled"),
					resource.TestCheckResourceAttr("britive_resource_manager_resource_type.new_resource_type_scan_enabled", "scan_enabled", "false"),
				),
			},
			{
				Config: testAccCheckBritiveResourceTypeScanEnabledConfig(resourceTypeName, resourceTypeDescription, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_scan_enabled"),
					resource.TestCheckResourceAttr("britive_resource_manager_resource_type.new_resource_type_scan_enabled", "scan_enabled", "true"),
				),
			},
		},
	})
}

// TestBritiveResourceTypeScanEnabledSingleApply proves the specific capability this backend
// change unlocked: scan_enabled = true set directly in the same apply that creates both the
// resource type and its first schedule scan. Previously the resource type's scan task
// service was only created lazily on the first schedule scan, making this combination fail
// outright - now the task service is registered as part of resource type creation itself.
func TestBritiveResourceTypeScanEnabledSingleApply(t *testing.T) {
	resourceTypeName := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Scan_Enabled_Single_Apply"
	resourceTypeDescription := "AT-Britive_Schedule_Scan_Tests_Resource_Type_Scan_Enabled_Single_Apply_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypeScanEnabledConfig(resourceTypeName, resourceTypeDescription, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type.new_resource_type_scan_enabled"),
					testAccCheckBritiveScheduleScanExists("britive_resource_manager_resource_type_schedule_scan.new_schedule_scan_for_enabled"),
					resource.TestCheckResourceAttrSet("britive_resource_manager_resource_type.new_resource_type_scan_enabled", "task_service_id"),
					resource.TestCheckResourceAttr("britive_resource_manager_resource_type.new_resource_type_scan_enabled", "scan_enabled", "true"),
				),
			},
		},
	})
}

func testAccCheckBritiveScheduleScanDailyConfig(resourceTypeName, resourceTypeDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_ss_daily" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_schedule_scan" "new_schedule_scan_daily" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_ss_daily.id
		name              = "AT-daily-scan"
		frequency_type    = "Daily"
		start_time        = "06:30"
	}`, resourceTypeName, resourceTypeDescription)
}

func testAccCheckBritiveScheduleScanWeeklyConfig(resourceTypeName, resourceTypeDescription, dayOfWeek, startTime, selectedValue string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_ss_weekly" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_label" "schedule_scan_label" {
		name        = "AT_Schedule_Scan_Label"
		description = "AT_Schedule_Scan_Label_Description"

		values {
			name = "Val1"
		}
		values {
			name = "Val2"
		}
	}

	resource "britive_resource_manager_resource_type_schedule_scan" "new_schedule_scan_weekly" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_ss_weekly.id
		name              = "AT-weekly-scan"
		description       = "AT-weekly-scan-description"
		frequency_type    = "Weekly"
		day_of_week       = "%s"
		start_time        = "%s"

		resource_labels {
			label_key = britive_resource_manager_resource_label.schedule_scan_label.name
			values    = ["%s"]
		}
	}`, resourceTypeName, resourceTypeDescription, dayOfWeek, startTime, selectedValue)
}

func testAccCheckBritiveScheduleScanMonthlyConfig(resourceTypeName, resourceTypeDescription string, dayOfMonth int) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_ss_monthly" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_schedule_scan" "new_schedule_scan_monthly" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_ss_monthly.id
		name              = "AT-monthly-scan"
		frequency_type    = "Monthly"
		day_of_month      = %d
		start_time        = "03:30"
	}`, resourceTypeName, resourceTypeDescription, dayOfMonth)
}

func testAccCheckBritiveResourceTypeScanEnabledConfig(resourceTypeName, resourceTypeDescription string, scanEnabled bool) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_scan_enabled" {
		name         = "%s"
		description  = "%s"
		scan_enabled = %t
	}

	resource "britive_resource_manager_resource_type_schedule_scan" "new_schedule_scan_for_enabled" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_scan_enabled.id
		name              = "AT-scan-enabled-scan"
		frequency_type    = "Daily"
		start_time        = "06:30"
	}`, resourceTypeName, resourceTypeDescription, scanEnabled)
}

func testAccCheckBritiveScheduleScanExists(n string) resource.TestCheckFunc {
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
