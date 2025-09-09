package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceTypePermission(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_Description"
	responseTemplateName := "AT-Britive_Resource_Manager_Tests_Response_Template"
	responseTemplateDescription := "AT-Britive_Resource_Manager_Tests_Response_Template_Description"
	permissionName := "AT-Britive_Resource_Manager_Tests_Resource_Type_Permission"
	permissionDescription := "At-Britive_Resource_Manager_Tests_ResourceType_Permision_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypePermissionConfig(resourceTypeName, resourceTypeDescription, responseTemplateName, responseTemplateDescription, permissionName, permissionDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypePermissionExists("britive_resource_manager_resource_type.new_resource_type_1"),
					testAccCheckBritiveResourceTypePermissionExists("britive_resource_manager_response_template.new_response_template_1"),
					testAccCheckBritiveResourceTypePermissionExists("britive_resource_manager_resource_type_permission.new_resource_type_permission_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceTypePermissionConfig(resourceTypeName, resourceTypeDescription, responseTemplateName, responseTemplateDescription, permissionName, permissionDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_1" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name = "testfield1"
			param_type = "password"
			is_mandatory = true
		}
		parameters {
			param_name = "testfield2"
			param_type = "Password"
			is_mandatory = false
		}
		parameters {
			param_name = "testfield3"
			param_type = "string"
			is_mandatory = true
		}
		parameters {
			param_name = "testfield4"
			param_type = "String"
			is_mandatory = true
		}
	}
		
	resource "britive_resource_manager_response_template" "new_response_template_1" {
    	name        = "%s"
    	description = "%s"
    	template_data = "The user {{name}} for the role {{role}}."
    	is_console_access_enabled = true
    	show_on_ui = false
	}
		
	resource "britive_resource_manager_resource_type_permission" "new_resource_type_permission_1" {
		name                = "%s"
		resource_type_id    = britive_resource_manager_resource_type.new_resource_type_1.id
		description         = "%s"
		checkin_time_limit  = 160
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
		response_templates = [britive_resource_manager_response_template.new_response_template_1.name]
	}`, resourceTypeName, resourceTypeDescription, responseTemplateName, responseTemplateDescription, permissionName, permissionDescription)
}

func testAccCheckBritiveResourceTypePermissionExists(n string) resource.TestCheckFunc {
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
