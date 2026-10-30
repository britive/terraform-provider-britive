package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
					resource.TestCheckTypeSetElemNestedAttrs("britive_resource_manager_profile_permission.profile_permission_1", "variables.*", map[string]string{
						"name":               "test1",
						"value":              "t1",
						"is_system_defined":  "false",
						"prompt_at_checkout": "false",
						"type":               "String",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("britive_resource_manager_profile_permission.profile_permission_1", "variables.*", map[string]string{
						"name":               "test2",
						"is_system_defined":  "false",
						"regex_pattern":      "^[a-z]{3}$",
						"description":        "test regex pattern",
						"prompt_at_checkout": "true",
						"type":               "password",
					}),
				),
			},
			// Re-plan with the same, unchanged config: this must produce an
			// empty plan, confirming prompt_at_checkout's config-driven
			// default (false when test1 omits it) is stable across plans.
			{
				Config:   testAccCheckBritiveResourceManagerProfilePermissionConfig(resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription),
				PlanOnly: true,
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
			is_system_defined = false
			regex_pattern = "^[a-z]{3}$"
			description = "test regex pattern"
			prompt_at_checkout = true
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
		variables           = ["test1", "test2:password"]
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
		name                             = "%s"
		description                      = "%s"
		expiration_duration              = 10800000
		extendable                       = true
		notification_prior_to_expiration = "1h0m0s"
		extension_duration               = "2h0m0s"
		extension_limit                  = 2

		associations {
			label_key = "Resource-Type"
			values    = [britive_resource_manager_resource_type.resource_type_1.name]
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

// TestBritiveResourceManagerProfilePermissionUnknownVariables is a regression test: the
// "variables" dynamic block is gated by
// britive_resource_manager_resource_type.resource_type_1.id, an attribute only known
// after apply for a newly created resource type. This keeps the variables Set unknown
// all the way through PlanResourceChange, which previously crashed ModifyPlan's
// req.Plan.Get into []PermissionVariableModel.
func TestBritiveResourceManagerProfilePermissionUnknownVariables(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Test_Resource_Type_Name_Unknown"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Test_Resource_Type_Unknown_Description"
	resourceResourceName := "AT-Britive_Resource_Manager_Test_Resource_Name_Unknown"
	resourceResourceDescription := "AT-Britive_Resource_Manager_Test_Resource_Unknown_Description"
	responseTemplateName := "AT-Britive_Resource_Manager_Test_Response_Template_Name_Unknown"
	responseTemplateDescription := "AT-Britive_Resource_Manager_Test_Response_Template_Unknown_Description"
	resourceTypePermissionName := "AT-Britive_Resource_Manager_Test_Resource_Type_Permission_Name_Unknown"
	resourceTypePermissionDescription := "AT-Britive_Resource_Manager_Test_Resource_Type_Permission_Unknown_Description"
	resourceManagerProfileName := "AT-Britive_Resource_Manager_Test_Resource_Profile_Name_Unknown"
	resourceManagerProfileDescription := "AT-Britive_Resource_Manager_Test_Resource_Profile_Unknown_Description"
	resourceName := "britive_resource_manager_profile_permission.unknown_vars"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceManagerProfilePermissionUnknownVariablesConfig(resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfilePermissionUnknownVariablesConfig(resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_profile_permission" "unknown_vars" {
		profile_id = britive_resource_manager_profile.profile_1.id
		name       = britive_resource_manager_resource_type_permission.type_permission_1.name
		version    = "LoCaL"

		dynamic "variables" {
			for_each = britive_resource_manager_resource_type.resource_type_1.id == "" ? [] : [{ name = "test1" }]
			content {
				name              = variables.value.name
				is_system_defined = false
			}
		}
	}

	resource "britive_resource_manager_resource_type" "resource_type_1" {
		name        = "%s"
		description = "%s"
		parameters {
			param_name   = "testfield5"
			param_type   = "StrinG"
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
		variables           = ["test1"]
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
		name                             = "%s"
		description                      = "%s"
		expiration_duration              = 10800000
		extendable                       = true
		notification_prior_to_expiration = "1h0m0s"
		extension_duration               = "2h0m0s"
		extension_limit                  = 2

		associations {
			label_key = "Resource-Type"
			values    = [britive_resource_manager_resource_type.resource_type_1.name]
		}
	}
	`, resourceTypeName, resourceTypeDescription, resourceResourceName, resourceResourceDescription, responseTemplateName, responseTemplateDescription, resourceTypePermissionName, resourceTypePermissionDescription, resourceManagerProfileName, resourceManagerProfileDescription)
}

// TestBritiveResourceManagerProfilePermissionVariableDefaults covers variables that have a
// default value on the resource type permission: they may be left out of the profile
// permission (the platform applies the default), set to override the default, and the
// override removed again. A variable without a default may be left out too, but a variable
// the permission doesn't declare is rejected.
func TestBritiveResourceManagerProfilePermissionVariableDefaults(t *testing.T) {
	namePrefix := "AT-Britive_Resource_Manager_Test_Profile_Permission_Var_Defaults"
	resourceName := "britive_resource_manager_profile_permission.var_defaults"

	config := func(profileVariables string) string {
		return testAccCheckBritiveResourceManagerProfilePermissionVariableDefaultsConfig(namePrefix, `["test1", "test2", "test3:password"]`, `
		variable_defaults {
			name  = "test1"
			value = "resource.name"
		}
		variable_defaults {
			name  = "test2"
			value = "user.email"
		}`, profileVariables)
	}
	// test3 has no default, so it must always be set.
	test3 := `
		variables {
			name               = "test3"
			is_system_defined  = false
			prompt_at_checkout = true
			regex_pattern      = "^[a-z]{3}$"
			description        = "test regex pattern"
		}`
	emptyPlan := resource.ConfigPlanChecks{
		PostApplyPostRefresh: []plancheck.PlanCheck{
			plancheck.ExpectEmptyPlan(),
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// test1 and test2 are left out and fall back to their defaults.
			{
				Config:           config(test3),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"name":               "test3",
						"prompt_at_checkout": "true",
						"type":               "password",
					}),
				),
			},
			// Override test1's default for this profile.
			{
				Config: config(test3 + `
		variables {
			name              = "test1"
			value             = "t1"
			is_system_defined = false
		}`),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "variables.*", map[string]string{
						"name":               "test1",
						"value":              "t1",
						"prompt_at_checkout": "false",
					}),
				),
			},
			// Remove the override: test1 falls back to its default again.
			{
				Config:           config(test3),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "1"),
				),
			},
			// test3 has no default, but the platform still accepts leaving it out.
			{
				Config:           config(""),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "0"),
				),
			},
			// A variable the permission doesn't declare is rejected.
			{
				Config: config(`
		variables {
			name              = "not_declared"
			value             = "x"
			is_system_defined = false
		}`),
				ExpectError: regexp.MustCompile(`variable\s+'not_declared'\s+is\s+not\s+valid`),
			},
		},
	})
}

// TestBritiveResourceManagerProfilePermissionAllVariablesDefaulted covers a permission whose
// variables all have defaults: the profile permission can be created with no variables, and
// removing the last configured variable must send an empty list so it is cleared on the
// backend (a stale value would show as a non-empty plan after refresh).
func TestBritiveResourceManagerProfilePermissionAllVariablesDefaulted(t *testing.T) {
	namePrefix := "AT-Britive_Resource_Manager_Test_Profile_Permission_All_Defaulted"
	resourceName := "britive_resource_manager_profile_permission.var_defaults"

	config := func(profileVariables string) string {
		return testAccCheckBritiveResourceManagerProfilePermissionVariableDefaultsConfig(namePrefix, `["test1"]`, `
		variable_defaults {
			name  = "test1"
			value = "resource.name"
		}`, profileVariables)
	}
	emptyPlan := resource.ConfigPlanChecks{
		PostApplyPostRefresh: []plancheck.PlanCheck{
			plancheck.ExpectEmptyPlan(),
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:           config(""),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "0"),
				),
			},
			// Import must read back no variables: the defaulted variable shows only in the
			// API's effectiveVariables, not its variables. Import sets profile_id to the bare
			// profile ID and version to the resolved version number, so those differ from config.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"profile_id",
					"version",
				},
			},
			{
				Config: config(`
		variables {
			name              = "test1"
			value             = "t1"
			is_system_defined = false
		}`),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "1"),
				),
			},
			{
				Config:           config(""),
				ConfigPlanChecks: emptyPlan,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "variables.#", "0"),
				),
			},
		},
	})
}

// TestBritiveResourceManagerProfilePermissionDefaultChange covers changing a default on the
// resource type permission while a profile permission on version "latest" leaves that
// variable out. The permission is updated in place (a new version), and the profile
// permission must not change or drift.
func TestBritiveResourceManagerProfilePermissionDefaultChange(t *testing.T) {
	namePrefix := "AT-Britive_Resource_Manager_Test_Profile_Permission_Default_Change"
	resourceName := "britive_resource_manager_profile_permission.var_defaults"
	permissionResourceName := "britive_resource_manager_resource_type_permission.var_defaults"

	config := func(defaultValue string) string {
		return testAccCheckBritiveResourceManagerProfilePermissionVariableDefaultsConfig(namePrefix, `["test1"]`, fmt.Sprintf(`
		variable_defaults {
			name  = "test1"
			value = "%s"
		}`, defaultValue), "")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("resource.name"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "0"),
				),
			},
			{
				Config: config("user.email"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(permissionResourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(permissionResourceName, "variable_defaults.*", map[string]string{
						"name":  "test1",
						"value": "user.email",
					}),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "0"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceManagerProfilePermissionVariableDefaultsConfig(namePrefix, permissionVariables, variableDefaults, profileVariables string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "var_defaults" {
		name        = "%[1]s_Resource_Type"
		description = "%[1]s_Resource_Type_Description"
		parameters {
			param_name   = "testfield5"
			param_type   = "string"
			is_mandatory = true
		}
	}

	resource "britive_resource_manager_resource" "var_defaults" {
		name          = "%[1]s_Resource"
		description   = "%[1]s_Resource_Description"
		resource_type = britive_resource_manager_resource_type.var_defaults.name
		parameter_values = {
			"testfield5" = "v5"
		}
	}

	resource "britive_resource_manager_resource_type_permission" "var_defaults" {
		name               = "%[1]s_Permission"
		resource_type_id   = britive_resource_manager_resource_type.var_defaults.id
		description        = "%[1]s_Permission_Description"
		is_draft           = false
		show_orig_creds    = true
		response_templates = []
		variables          = %[2]s
		code_language      = "Text"
		checkin_code       = "checkin"
		checkout_code      = "checkout"
		%[3]s
	}

	resource "britive_resource_manager_profile" "var_defaults" {
		name                = "%[1]s_Profile"
		description         = "%[1]s_Profile_Description"
		expiration_duration = 10800000
		extendable          = false

		associations {
			label_key = "Resource-Type"
			values    = [britive_resource_manager_resource_type.var_defaults.name]
		}
	}

	resource "britive_resource_manager_profile_permission" "var_defaults" {
		profile_id = britive_resource_manager_profile.var_defaults.id
		name       = britive_resource_manager_resource_type_permission.var_defaults.name
		version    = "latest"
		%[4]s
	}`, namePrefix, permissionVariables, variableDefaults, profileVariables)
}

// TestBritiveResourceManagerPermissionsProviderUpgrade checks that state written by the
// last release, before variable_defaults existed and while every declared variable had to
// be set, plans clean with this version, and that the new features then work on it.
// As in TestBritiveTagProviderMigration, no TestCase-level providers are set so each
// step uses only its own provider.
func TestBritiveResourceManagerPermissionsProviderUpgrade(t *testing.T) {
	namePrefix := "AT-Britive_Resource_Manager_Test_Permissions_Upgrade"
	resourceName := "britive_resource_manager_profile_permission.var_defaults"
	permissionResourceName := "britive_resource_manager_resource_type_permission.var_defaults"

	allVariables := `
		variables {
			name              = "test1"
			value             = "t1"
			is_system_defined = false
		}
		variables {
			name              = "test2"
			value             = "t2"
			is_system_defined = false
		}`
	oldConfig := testAccCheckBritiveResourceManagerProfilePermissionVariableDefaultsConfig(namePrefix, `["test1", "test2"]`, "", allVariables)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheckFramework(t) },
		Steps: []resource.TestStep{
			// Create with the last release.
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"britive": {
						Source:            "britive/britive",
						VersionConstraint: "= 3.0.4",
					},
				},
				Config: oldConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceManagerProfilePermissionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "2"),
				),
			},
			// The same config must plan clean with this version.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   oldConfig,
				PlanOnly:                 true,
			},
			// The new features work on the upgraded state: add a default to the permission
			// and leave that variable out of the profile permission.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: testAccCheckBritiveResourceManagerProfilePermissionVariableDefaultsConfig(namePrefix, `["test1", "test2"]`, `
		variable_defaults {
			name  = "test1"
			value = "resource.name"
		}`, `
		variables {
			name              = "test2"
			value             = "t2"
			is_system_defined = false
		}`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(permissionResourceName, "variable_defaults.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "variables.#", "1"),
				),
			},
		},
	})
}
