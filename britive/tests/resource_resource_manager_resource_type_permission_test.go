package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveResourceTypePermission(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_Description"
	responseTemplateName := "AT-Britive_Resource_Manager_Tests_Response_Template"
	responseTemplateDescription := "AT-Britive_Resource_Manager_Tests_Response_Template_Description"
	permissionName := "AT-Britive_Resource_Manager_Tests_Resource_Type_Permission"
	permissionDescription := "At-Britive_Resource_Manager_Tests_ResourceType_Permision_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// TestBritiveResourceTypePermissionForEachUnknownSets is a regression test: count on
// the resource with "variables" assigned directly from count.index previously crashed
// ValidateConfig's req.Config.Get into []types.String, since the un-expanded resource is
// validated with count.index left unknown.
//
// count (not for_each) is used here deliberately: terraform-plugin-testing's legacy
// state shim used by TestCheckResourceAttr/RootModule().Resources lookups does not
// support for_each string-keyed instances ("unexpected index type (string) ...,
// for_each is not supported"), only count's integer-keyed instances.
//
// response_templates is set explicitly (rather than left unset) to sidestep a separate,
// pre-existing provider behavior where an omitted response_templates plans as null but
// applies as an empty set, which is unrelated to what this test is regression-checking.
func TestBritiveResourceTypePermissionForEachUnknownSets(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type_ForEach"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_ForEach_Description"
	permissionName := "AT-Britive_Resource_Manager_Tests_Resource_Type_Permission_ForEach"
	permissionDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_Permission_ForEach_Description"
	resourceName := "britive_resource_manager_resource_type_permission.iterated.0"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypePermissionForEachUnknownSetsConfig(resourceTypeName, resourceTypeDescription, permissionName, permissionDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypePermissionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "2"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceTypePermissionForEachUnknownSetsConfig(resourceTypeName, resourceTypeDescription, permissionName, permissionDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_1" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name   = "testfield1"
			param_type   = "string"
			is_mandatory = true
		}
	}

	locals {
		permissions = [
			{
				name      = "%s"
				variables = ["test1", "test2"]
			}
		]
	}

	resource "britive_resource_manager_resource_type_permission" "iterated" {
		count = length(local.permissions)

		name                = local.permissions[count.index].name
		resource_type_id    = britive_resource_manager_resource_type.new_resource_type_1.id
		description         = "%s"
		show_orig_creds     = true
		response_templates  = []
		variables           = local.permissions[count.index].variables
	}`, resourceTypeName, resourceTypeDescription, permissionName, permissionDescription)
}

// TestBritiveResourceTypePermissionUnknownSetsFromDependency is a regression test:
// "variables" is gated by britive_resource_manager_resource_type.trigger.id, an
// attribute only known after apply since "trigger" is created in the same operation.
// This keeps the Set unknown all the way through PlanResourceChange (unlike the
// for_each+each.value case, which resolves by plan time), which previously crashed
// ModifyPlan's req.Plan.Get into []types.String.
func TestBritiveResourceTypePermissionUnknownSetsFromDependency(t *testing.T) {
	triggerResourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type_Unknown_Trigger"
	triggerResourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_Unknown_Trigger_Description"
	resourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type_Unknown"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_Unknown_Description"
	permissionName := "AT-Britive_Resource_Manager_Tests_Resource_Type_Permission_Unknown"
	permissionDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_Permission_Unknown_Description"
	resourceName := "britive_resource_manager_resource_type_permission.unknown_vars"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceTypePermissionUnknownSetsFromDependencyConfig(triggerResourceTypeName, triggerResourceTypeDescription, resourceTypeName, resourceTypeDescription, permissionName, permissionDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceTypePermissionExists(resourceName),
					testAccCheckBritiveResourceTypePermissionExists("britive_resource_manager_resource_type.trigger"),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceTypePermissionUnknownSetsFromDependencyConfig(triggerResourceTypeName, triggerResourceTypeDescription, resourceTypeName, resourceTypeDescription, permissionName, permissionDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "trigger" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name   = "trigfield"
			param_type   = "string"
			is_mandatory = true
		}
	}

	resource "britive_resource_manager_resource_type" "new_resource_type_1" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name   = "testfield1"
			param_type   = "string"
			is_mandatory = true
		}
	}

	resource "britive_resource_manager_resource_type_permission" "unknown_vars" {
		name                = "%s"
		resource_type_id    = britive_resource_manager_resource_type.new_resource_type_1.id
		description         = "%s"
		show_orig_creds     = true
		response_templates  = []
		variables           = britive_resource_manager_resource_type.trigger.id == "" ? [] : ["test1"]
	}`, triggerResourceTypeName, triggerResourceTypeDescription, resourceTypeName, resourceTypeDescription, permissionName, permissionDescription)
}
