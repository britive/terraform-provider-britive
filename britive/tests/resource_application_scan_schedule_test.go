package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// All tests in this file use application_type = "Britive" as the backing application: it's
// the lightest-weight application type already exercised by resource_application_test.go (no
// external credentials/keys needed), and supports a root environment group like the other
// application types britive_application_scan_schedule's associations/org_scan logic cares
// about. These tests deliberately don't exercise associations/org_scan, since support for
// those varies per application type (see application_scan_schedule.md's note on
// supportsEnvironmentScanning/requiresHierarchicalModel) and isn't confirmed for this type -
// they only exercise the scheduling fields (frequency_type and friends) and scan_enabled,
// which are supported unconditionally.

func TestBritiveApplicationScanScheduleDaily(t *testing.T) {
	appName := "AT-App_Scan_Schedule_Daily"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationScanScheduleDailyConfig(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_daily"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.daily"),
				),
			},
		},
	})
}

func TestBritiveApplicationScanScheduleWeekly(t *testing.T) {
	appName := "AT-App_Scan_Schedule_Weekly"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationScanScheduleWeeklyConfig(appName, "Monday", "11:00"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_weekly"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.weekly"),
				),
			},
			// In-place update: day_of_week (abbreviation this time) and start_time change
			// without replacing the resource.
			{
				Config: testAccCheckBritiveApplicationScanScheduleWeeklyConfig(appName, "Fri", "23:45"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_weekly"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.weekly"),
				),
			},
		},
	})
}

func TestBritiveApplicationScanScheduleMonthly(t *testing.T) {
	appName := "AT-App_Scan_Schedule_Monthly"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationScanScheduleMonthlyConfig(appName, 6),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_monthly"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.monthly"),
				),
			},
			// In-place update: day_of_month changes without replacing the resource.
			{
				Config: testAccCheckBritiveApplicationScanScheduleMonthlyConfig(appName, 15),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_monthly"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.monthly"),
				),
			},
		},
	})
}

// TestBritiveApplicationScanScheduleHourly exercises hour_interval, the one frequency_type
// with no resourcemanager.ScheduleScanResource equivalent.
func TestBritiveApplicationScanScheduleHourly(t *testing.T) {
	appName := "AT-App_Scan_Schedule_Hourly"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationScanScheduleHourlyConfig(appName, 5),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_hourly"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.hourly"),
				),
			},
			// In-place update: hour_interval changes without replacing the resource.
			{
				Config: testAccCheckBritiveApplicationScanScheduleHourlyConfig(appName, 8),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_schedule_app_hourly"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.hourly"),
				),
			},
		},
	})
}

// TestBritiveApplicationScanEnabled exercises scan_enabled on britive_application across an
// Update (false -> true), alongside a sibling britive_application_scan_schedule - mirrors
// resourcemanager's TestBritiveResourceTypeScanEnabled.
func TestBritiveApplicationScanEnabled(t *testing.T) {
	appName := "AT-App_Scan_Enabled"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationScanEnabledConfig(appName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_enabled_app"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.for_enabled"),
					resource.TestCheckResourceAttr("britive_application.scan_enabled_app", "scan_enabled", "false"),
				),
			},
			{
				Config: testAccCheckBritiveApplicationScanEnabledConfig(appName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_enabled_app"),
					resource.TestCheckResourceAttr("britive_application.scan_enabled_app", "scan_enabled", "true"),
				),
			},
		},
	})
}

// TestBritiveApplicationScanEnabledSingleApply proves scan_enabled = true can be set in the
// same apply that creates both the application and its first scan schedule - mirrors
// resourcemanager's TestBritiveResourceTypeScanEnabledSingleApply.
func TestBritiveApplicationScanEnabledSingleApply(t *testing.T) {
	appName := "AT-App_Scan_Enabled_Single_Apply"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveApplicationScanEnabledConfig(appName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveApplicationScanScheduleExists("britive_application.scan_enabled_app"),
					testAccCheckBritiveApplicationScanScheduleExists("britive_application_scan_schedule.for_enabled"),
					resource.TestCheckResourceAttrSet("britive_application.scan_enabled_app", "task_service_id"),
					resource.TestCheckResourceAttr("britive_application.scan_enabled_app", "scan_enabled", "true"),
				),
			},
		},
	})
}

func testAccCheckBritiveApplicationScanScheduleDailyConfig(appName string) string {
	return fmt.Sprintf(`
	resource "britive_application" "scan_schedule_app_daily" {
		application_type = "Britive"
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_scan_schedule" "daily" {
		application_id = britive_application.scan_schedule_app_daily.id
		name           = "AT-daily-scan"
		frequency_type = "Daily"
		start_time     = "06:30"
	}`, appName, appName)
}

func testAccCheckBritiveApplicationScanScheduleWeeklyConfig(appName, dayOfWeek, startTime string) string {
	return fmt.Sprintf(`
	resource "britive_application" "scan_schedule_app_weekly" {
		application_type = "Britive"
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_scan_schedule" "weekly" {
		application_id = britive_application.scan_schedule_app_weekly.id
		name           = "AT-weekly-scan"
		frequency_type = "Weekly"
		day_of_week    = "%s"
		start_time     = "%s"
	}`, appName, appName, dayOfWeek, startTime)
}

func testAccCheckBritiveApplicationScanScheduleMonthlyConfig(appName string, dayOfMonth int) string {
	return fmt.Sprintf(`
	resource "britive_application" "scan_schedule_app_monthly" {
		application_type = "Britive"
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_scan_schedule" "monthly" {
		application_id = britive_application.scan_schedule_app_monthly.id
		name           = "AT-monthly-scan"
		frequency_type = "Monthly"
		day_of_month   = %d
		start_time     = "03:30"
	}`, appName, appName, dayOfMonth)
}

func testAccCheckBritiveApplicationScanScheduleHourlyConfig(appName string, hourInterval int) string {
	return fmt.Sprintf(`
	resource "britive_application" "scan_schedule_app_hourly" {
		application_type = "Britive"
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_scan_schedule" "hourly" {
		application_id = britive_application.scan_schedule_app_hourly.id
		name           = "AT-hourly-scan"
		frequency_type = "Hourly"
		hour_interval  = %d
	}`, appName, appName, hourInterval)
}

func testAccCheckBritiveApplicationScanEnabledConfig(appName string, scanEnabled bool) string {
	return fmt.Sprintf(`
	resource "britive_application" "scan_enabled_app" {
		application_type = "Britive"
		scan_enabled      = %t
		user_account_mappings {
			name        = "Mobile"
			description = "Mobile"
		}
		properties {
			name  = "displayName"
			value = "%s"
		}
		properties {
			name  = "description"
			value = "%s Description"
		}
		properties {
			name  = "maxSessionDurationForProfiles"
			value = 604800
		}
	}

	resource "britive_application_scan_schedule" "for_enabled" {
		application_id = britive_application.scan_enabled_app.id
		name           = "AT-scan-enabled-scan"
		frequency_type = "Daily"
		start_time     = "06:30"
	}`, scanEnabled, appName, appName)
}

func testAccCheckBritiveApplicationScanScheduleExists(n string) resource.TestCheckFunc {
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
