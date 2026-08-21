package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveResourceLabel(t *testing.T) {
	resourceLabelName := "AT-Britive_Resource_Manager_Test_Resource_Label"
	resourceLabelDescription := "AT-Britive_Resource_Manager_Test_Resource_Label_Description"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceLabelExists("britive_resource_manager_resource_label.resource_label_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_label" "resource_label_1" {
		name         = "%s"
		description  = "%s"
		label_color  = "#abc123"

		values {
			name = "YS Val"
			description = "YS Val Desc"
		}
		values {
			name = "YS Val 1"
			description = "YS Val Desc1"
		}
	}
	`, resourceLabelName, resourceLabelDescription)
}

func testAccCheckBritiveResourceLabelExists(n string) resource.TestCheckFunc {
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

// TestBritiveResourceLabelUnknownValuesFromDependency is a regression test: the
// "values" dynamic block is gated by britive_resource_manager_resource_label.trigger.id,
// an attribute only known after apply since "trigger" is created in the same operation.
// This keeps the values List unknown all the way through PlanResourceChange, which
// previously crashed ModifyPlan's req.Plan.Get into []ResourceLabelValueModel.
func TestBritiveResourceLabelUnknownValuesFromDependency(t *testing.T) {
	triggerName := "AT-Britive_Resource_Manager_Test_Resource_Label_Trigger"
	triggerDescription := "AT-Britive_Resource_Manager_Test_Resource_Label_Trigger_Description"
	resourceLabelName := "AT-Britive_Resource_Manager_Test_Resource_Label_Unknown"
	resourceLabelDescription := "AT-Britive_Resource_Manager_Test_Resource_Label_Unknown_Description"
	resourceName := "britive_resource_manager_resource_label.unknown_values"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelUnknownValuesFromDependencyConfig(triggerName, triggerDescription, resourceLabelName, resourceLabelDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceLabelExists(resourceName),
					testAccCheckBritiveResourceLabelExists("britive_resource_manager_resource_label.trigger"),
					resource.TestCheckResourceAttr(resourceName, "values.#", "1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceLabelUnknownValuesFromDependencyConfig(triggerName, triggerDescription, resourceLabelName, resourceLabelDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_label" "trigger" {
		name        = "%s"
		description = "%s"
	}

	resource "britive_resource_manager_resource_label" "unknown_values" {
		name        = "%s"
		description = "%s"

		dynamic "values" {
			for_each = britive_resource_manager_resource_label.trigger.id == "" ? [] : [{ name = "YS Val", description = "YS Val Desc" }]
			content {
				name        = values.value.name
				description = values.value.description
			}
		}
	}
	`, triggerName, triggerDescription, resourceLabelName, resourceLabelDescription)
}
