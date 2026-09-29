package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveRotationTemplateLocal(t *testing.T) {
	resourceTypeName := "AT-Britive_Rotation_Template_Tests_Resource_Type_Local"
	resourceTypeDescription := "AT-Britive_Rotation_Template_Tests_Resource_Type_Local_Description"
	templateName := "AT-Britive_Rotation_Template_Tests_Local"
	templateDescription := "AT-Britive_Rotation_Template_Tests_Local_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRotationTemplateLocalConfig(resourceTypeName, resourceTypeDescription, templateName, templateDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type.new_resource_type_rt_local"),
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type_rotation_template.new_rotation_template_local"),
				),
			},
		},
	})
}

func TestBritiveRotationTemplateInlineFile(t *testing.T) {
	resourceTypeName := "AT-Britive_Rotation_Template_Tests_Resource_Type_Inline"
	resourceTypeDescription := "AT-Britive_Rotation_Template_Tests_Resource_Type_Inline_Description"
	templateName := "AT-Britive_Rotation_Template_Tests_Inline"
	templateDescription := "AT-Britive_Rotation_Template_Tests_Inline_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRotationTemplateInlineFileConfig(resourceTypeName, resourceTypeDescription, templateName, templateDescription, "python"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type.new_resource_type_rt_inline"),
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type_rotation_template.new_rotation_template_inline"),
				),
			},
			// Insert Code -> Local, updated in place (no replace), mirroring the captured
			// UI flow of switching YSTestTemplate2 from python inline code to Local.
			{
				Config: testAccCheckBritiveRotationTemplateLocalUpdateConfig(resourceTypeName, resourceTypeDescription, templateName, templateDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type.new_resource_type_rt_inline"),
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type_rotation_template.new_rotation_template_inline"),
				),
			},
		},
	})
}

// TestBritiveRotationTemplateForEachDynamicVariables is a regression test for a bug where
// combining `for_each` on britive_resource_manager_resource_type_rotation_template with a
// nested `dynamic "variables"` block whose own for_each is derived from each.value made
// Terraform represent the whole variables block collection as unknown during validate/plan
// (Terraform core can't statically resolve a dynamic block's repetition count in that
// combination). The provider's variables model field used to be a plain
// []RotationTemplateVariableModel, which can't hold an unknown value, and calling Config.Get
// on it in ValidateConfig crashed with "Value Conversion Error ... Suggested Type:
// basetypes.SetValue". Now fixed by modeling variables as types.Set instead.
func TestBritiveRotationTemplateForEachDynamicVariables(t *testing.T) {
	resourceTypeName := "AT-Britive_Rotation_Template_Tests_Resource_Type_ForEach"
	resourceTypeDescription := "AT-Britive_Rotation_Template_Tests_Resource_Type_ForEach_Description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveRotationTemplateForEachDynamicVariablesConfig(resourceTypeName, resourceTypeDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveRotationTemplateExists("britive_resource_manager_resource_type.new_resource_type_rt_for_each"),
					testAccCheckBritiveRotationTemplateExists(`britive_resource_manager_resource_type_rotation_template.new_rotation_template_for_each["a"]`),
					testAccCheckBritiveRotationTemplateExists(`britive_resource_manager_resource_type_rotation_template.new_rotation_template_for_each["b"]`),
				),
			},
		},
	})
}

func testAccCheckBritiveRotationTemplateForEachDynamicVariablesConfig(resourceTypeName, resourceTypeDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_rt_for_each" {
		name        = "%s"
		description = "%s"
	}

	locals {
		rotation_templates_for_each = {
			a = {
				name     = "AT-for-each-rt-a"
				var_name = "var_a"
			}
			b = {
				name     = "AT-for-each-rt-b"
				var_name = "var_b"
			}
		}
	}

	resource "britive_resource_manager_resource_type_rotation_template" "new_rotation_template_for_each" {
		for_each         = local.rotation_templates_for_each
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_rt_for_each.id
		name             = each.value.name
		time_limit       = 5
		template_type    = "Local"

		dynamic "variables" {
			for_each = [each.value]
			content {
				name         = variables.value.var_name
				type         = "String"
				multi_valued = false
			}
		}
	}`, resourceTypeName, resourceTypeDescription)
}

func testAccCheckBritiveRotationTemplateLocalConfig(resourceTypeName, resourceTypeDescription, templateName, templateDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_rt_local" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_rotation_template" "new_rotation_template_local" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_rt_local.id
		name             = "%s"
		description      = "%s"
		time_limit       = 5
		template_type    = "Local"

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
		variables {
			name         = "var3"
			type         = "Date"
			multi_valued = false
		}
	}`, resourceTypeName, resourceTypeDescription, templateName, templateDescription)
}

func testAccCheckBritiveRotationTemplateInlineFileConfig(resourceTypeName, resourceTypeDescription, templateName, templateDescription, scriptLanguage string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_rt_inline" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_rotation_template" "new_rotation_template_inline" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_rt_inline.id
		name             = "%s"
		description      = "%s"
		time_limit       = 10
		template_type    = "InlineFile"
		script_language  = "%s"
		script_content   = <<EOT
			import log
			log.info("rotating credential")
		EOT

		variables {
			name         = "username"
			type         = "String"
			multi_valued = false
		}
	}`, resourceTypeName, resourceTypeDescription, templateName, templateDescription, scriptLanguage)
}

func testAccCheckBritiveRotationTemplateLocalUpdateConfig(resourceTypeName, resourceTypeDescription, templateName, templateDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "new_resource_type_rt_inline" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_type_rotation_template" "new_rotation_template_inline" {
		resource_type_id = britive_resource_manager_resource_type.new_resource_type_rt_inline.id
		name             = "%s"
		description      = "%s"
		time_limit       = 10
		template_type    = "Local"

		variables {
			name         = "username"
			type         = "String"
			multi_valued = false
		}
	}`, resourceTypeName, resourceTypeDescription, templateName, templateDescription)
}

func testAccCheckBritiveRotationTemplateExists(n string) resource.TestCheckFunc {
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
