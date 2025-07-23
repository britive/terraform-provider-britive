package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceManagerProfilePermission(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Test_Resource_Type_Name_1"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Test_Resource_Type_1_Description"
	resourceResourceName := "AT-Britive_Resource_Manager_Test_Resource_Name_1"
	resourceResourceDescription := "AT-Britive_Resource_Manager_Test_Resource_1_Description"
	responseTemplateName := "AT-Britive_Resource_Manager_Test_Response_Template_Name_1"
	responseTemplateDescription := "AT-Britive_Resource_Manager_Test_Response_Template_1_Description"
	resourceTypePermissionName := "AT-Britive_Resource_Manager_Test_Resource_Type_Permission_Name_2"
	resourceTypePermissionDescription := "AT-Britive_Resource_Manager_Test_Resource_Type_Permission_1_Description"
	resourceManagerProfileName := "AT-Britive_Resource_Manager_Test_Resource_Profile_Name_1"
	resourceManagerProfileDescription := "AT-Britive_Resource_Manager_Test_Resource_Profile_1_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePermissionConfig(resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_profile_permission.profile_permission_1"),
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_resource_type.resource_type_1"),
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_resource.resource_1"),
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_response_template.response_template"),
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_resource_type_permission.type_permission_1"),
					testAccCheckBritiveResourceManagerProfilePermissionExists("britive_resource_manager_profile.profile_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfilePermissionConfig(resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_profile_permission" "profile_permission_1" {
		profile_id = britive_resource_manager_profile.profile_1.id
		name = britive_resource_manager_resource_type_permission.type_permission_1.name
		version = "LoCaL"

		variables {
			name = "test1"
			value = "t1"
			is_system_defined = false
		}
		variables {
			name = "test2"
			value = "t3"
			is_system_defined = false
		}
	}

	resource "britive_resource_manager_resource_type" "resource_type_1" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name = "testfield5"
			param_type = "StrinG"
			is_mandatory = true
		}
	}

	resource "britive_resource_manager_resource" "resource_1" {
		name = "%s"
		description = "%s"
		resource_type = britive_resource_manager_resource_type.resource_type_1.name
		parameter_values = {
			"testfield5" = "v5"
		}
	}

	resource "britive_resource_manager_response_template" "response_template" {
		name                      = "%s"
		description               = "%s"
		is_console_access_enabled = true
		show_on_ui                = false
		template_data             = "The user {{YS}} has the {{admin}}."
	}

	resource "britive_resource_manager_resource_type_permission" "type_permission_1" {
		name                = "%s"
		resource_type_id    = britive_resource_manager_resource_type.resource_type_1.id
		description         = "%s"
		checkin_time_limit  = 180
		checkout_time_limit = 360
		is_draft            = false
		show_orig_creds     = true
		variables           = ["test1", "test2"]
		code_language = "PyThon"
		checkin_code  = <<EOT
			#!/bin/bash
			echo "Running task 1"
			echo "Running task 2"
			EOT
		checkout_code = <<EOT
			#!/bin/bash
			echo "Running task 2"
			echo "Running task 3"
			EOT
		response_templates = [britive_resource_manager_response_template.response_template.name]
	}

	resource "britive_resource_manager_profile" "profile_1" {
		name = "%s"
		description = "%s"
		expiration_duration = 3600000

		associations {
			label_key = "Resource-Type"
			values = [britive_resource_manager_resource_type.resource_type_1.name]
		}
	}
	`, resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription)
}

func testAccCheckBritiveResourceManagerProfilePermissionExists(n string) resource.TestCheckFunc {
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
