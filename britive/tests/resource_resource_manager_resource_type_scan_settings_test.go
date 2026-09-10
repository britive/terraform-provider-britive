package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveScanSettingsLocal(t *testing.T) {
	resourceTypeName := "AT-Britive_Scan_Settings_Tests_Resource_Type_Local"
	resourceTypeDescription := "AT-Britive_Scan_Settings_Tests_Resource_Type_Local_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveScanSettingsLocalConfig(resourceTypeName, resourceTypeDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScanSettingsExists("britive_resource_manager_resource_type.new_resource_type_ss_local"),
					testAccCheckBritiveScanSettingsExists("britive_resource_manager_resource_type_scan_settings.new_scan_settings_local"),
				),
			},
		},
	})
}

func TestBritiveScanSettingsInlineFile(t *testing.T) {
	resourceTypeName := "AT-Britive_Scan_Settings_Tests_Resource_Type_Inline"
	resourceTypeDescription := "AT-Britive_Scan_Settings_Tests_Resource_Type_Inline_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveScanSettingsInlineFileConfig(resourceTypeName, resourceTypeDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScanSettingsExists("britive_resource_manager_resource_type.new_resource_type_ss_inline"),
					testAccCheckBritiveScanSettingsExists("britive_resource_manager_resource_type_scan_settings.new_scan_settings_inline"),
				),
			},
			// InlineFile -> Local, updated in place (no replace), mirroring the captured
			// UI flow of switching scan settings from inline text back to Local.
			{
				Config: testAccCheckBritiveScanSettingsLocalUpdateConfig(resourceTypeName, resourceTypeDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveScanSettingsExists("britive_resource_manager_resource_type.new_resource_type_ss_inline"),
					testAccCheckBritiveScanSettingsExists("britive_resource_manager_resource_type_scan_settings.new_scan_settings_inline"),
				),
			},
		},
	})
}

func testAccCheckBritiveScanSettingsLocalConfig(resourceTypeName, resourceTypeDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_ss_local" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_scan_settings" "new_scan_settings_local" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_ss_local.id
		time_limit        = 20
		template_type     = "Local"

		variables {
			name         = "var1"
			type         = "String"
			multi_valued = false
		}
		variables {
			name         = "var2"
			type         = "Number"
			multi_valued = true
		}
	}`, resourceTypeName, resourceTypeDescription)
}

func testAccCheckBritiveScanSettingsInlineFileConfig(resourceTypeName, resourceTypeDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_ss_inline" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_scan_settings" "new_scan_settings_inline" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_ss_inline.id
		time_limit        = 20
		template_type     = "InlineFile"
		script_language   = "text"
		script_content    = <<EOT
			Test script here
			hii
			byyy
		EOT

		variables {
			name         = "username"
			type         = "String"
			multi_valued = false
		}
	}`, resourceTypeName, resourceTypeDescription)
}

func testAccCheckBritiveScanSettingsLocalUpdateConfig(resourceTypeName, resourceTypeDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_ss_inline" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_scan_settings" "new_scan_settings_inline" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_ss_inline.id
		time_limit        = 25
		template_type     = "Local"

		variables {
			name         = "username"
			type         = "String"
			multi_valued = false
		}
	}`, resourceTypeName, resourceTypeDescription)
}

func testAccCheckBritiveScanSettingsExists(n string) resource.TestCheckFunc {
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
