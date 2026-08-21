package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveResourceResource(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type_1"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_1_Description"
	resourceLabelName1 := "AT-Britive_Resource_Manager_Test_Resource_Label_111"
	resourceLabelDescription1 := "AT-Britive_Resource_Manager_Test_Resource_Label_111_Description"
	resourceResourceName := "AT-Britive_Resource_Tests_Resource_1"
	resourceResourceDescription := "AT-Britive_Resource_Test_Resource_Description_1"
	resourceResourceName2 := "AT-Britive_Resource_Tests_Resource_2"
	resourceResourceDescription2 := "AT-Britive_Resource_Test_Resource_Description_2"
	resourceName1 := "britive_resource_manager_resource.resource_1"
	resourceName2 := "britive_resource_manager_resource.resource_2"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceResourceConfigWithSecondResource(resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription, resourceResourceName2, resourceResourceDescription2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceResourceExists("britive_resource_manager_resource_type.resource_type_1"),
					testAccCheckBritiveResourceResourceExists("britive_resource_manager_resource_label.resource_label_1"),
					testAccCheckBritiveResourceResourceExists(resourceName1),
					testAccCheckBritiveResourceResourceExists(resourceName2),
				),
			},
			// Import via the documented "resources/{name}" format.
			{
				ResourceName:      resourceName1,
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("resources/%s", resourceResourceName),
				ImportStateVerify: true,
			},
			// Import via the legacy "resource-manager/resources/{id}" format.
			{
				ResourceName:      resourceName1,
				ImportState:       true,
				ImportStateIdFunc: testAccResourceManagerResourceImportStateId(resourceName1, "resource-manager/resources/"),
				ImportStateVerify: true,
			},
			// Import via a bare token, which must fall back from an ID lookup to a name lookup.
			{
				ResourceName:      resourceName1,
				ImportState:       true,
				ImportStateId:     resourceResourceName,
				ImportStateVerify: true,
			},
			// Import a resource with no parameter_values/resource_labels set. This guards against the
			// "resource_labels" / "parameter_values" nil-typed Map regression on import.
			{
				ResourceName:      resourceName2,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResourceManagerResourceImportStateId(resourceAddr, prefix string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", resourceAddr)
		}
		if rs.Primary.ID == "" {
			return "", errs.NewNotFoundErrorf("ID for %s in state", resourceAddr)
		}
		return prefix + rs.Primary.ID, nil
	}
}

func testAccCheckBritiveResourceResourceConfig(resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_type" "resource_type_1" {
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
	}

	resource "britive_resource_manager_resource_label" "resource_label_1" {
		name         = "%s"
		description  = "%s"
		label_color  = "#abc123"

		values {
			name = "Production"
			description = "Production Desc"
		}
		values {
			name = "Development"
			description = "Development Desc"
		}
	}

	resource "britive_resource_manager_resource" "resource_1" {
		name = "%s"
		description = "%s"
		resource_type = britive_resource_manager_resource_type.resource_type_1.name
		parameter_values = {
			"testfield1" = "v1"
			"testfield2" = "v2"
		}
		resource_labels = {
			// Alphabetically ordered to match the canonical order Read/ImportState normalize to
			// when there's no prior state order to preserve (see sameValueSet/sort.Strings usage
			// in resource_resource.go), so ImportStateVerify doesn't see a spurious ordering diff.
			"${britive_resource_manager_resource_label.resource_label_1.name}" = "Development,Production"
		}
	}

	`, resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription)
}

// testAccCheckBritiveResourceResourceConfigWithSecondResource extends the base config with a second
// resource type that has no mandatory parameters and a second server access resource built from it
// with no parameter_values/resource_labels set, to exercise the "resource_labels"/"parameter_values"
// nil-typed Map regression on import.
func testAccCheckBritiveResourceResourceConfigWithSecondResource(resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription, resourceResourceName2, resourceResourceDescription2 string) string {
	return testAccCheckBritiveResourceResourceConfig(resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription) + fmt.Sprintf(`

	resource "britive_resource_manager_resource_type" "resource_type_2" {
		name        = "%s_Type"
		description = "%s_Type_Description"
	}

	resource "britive_resource_manager_resource" "resource_2" {
		name = "%s"
		description = "%s"
		resource_type = britive_resource_manager_resource_type.resource_type_2.name
	}
	`, resourceResourceName2, resourceResourceName2, resourceResourceName2, resourceResourceDescription2)
}

func testAccCheckBritiveResourceResourceExists(n string) resource.TestCheckFunc {
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
