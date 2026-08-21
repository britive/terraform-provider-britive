package tests

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveResourceLabel(t *testing.T) {
	resourceLabelName := "AT-Britive_Resource_Manager_Test_Resource_Label"
	resourceLabelDescription := "AT-Britive_Resource_Manager_Test_Resource_Label_Description"
	resourceAddr := "britive_resource_manager_resource_label.resource_label_1"
	expectedValues := []map[string]string{
		{"name": "AT-Value-1", "description": "AT-Value-1_Description"},
		{"name": "AT-Value-2", "description": "AT-Value-2_Description"},
	}
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceLabelExists(resourceAddr),
				),
			},
			// Import via the documented "resource-manager/resource-labels/{id}" format.
			// "values" is excluded from ImportStateVerify and checked separately, order-independently:
			// the API's underlying value storage is unordered (see resource_label_resource.go),
			// so a fresh import's ordering isn't guaranteed to match the order set at create time.
			{
				ResourceName:            resourceAddr,
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceLabelImportStateId(resourceAddr, "resource-manager/resource-labels/"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values"},
				ImportStateCheck:        testAccCheckResourceLabelValuesImported(expectedValues),
			},
			// Import via the current "resource-manager/labels/{id}" format.
			{
				ResourceName:            resourceAddr,
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceLabelImportStateId(resourceAddr, "resource-manager/labels/"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values"},
				ImportStateCheck:        testAccCheckResourceLabelValuesImported(expectedValues),
			},
			// Import via a bare {id}.
			{
				ResourceName:            resourceAddr,
				ImportState:             true,
				ImportStateIdFunc:       testAccResourceLabelImportStateId(resourceAddr, ""),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values"},
				ImportStateCheck:        testAccCheckResourceLabelValuesImported(expectedValues),
			},
		},
	})
}

// testAccCheckResourceLabelValuesImported asserts that the imported "values" block contains
// exactly the expected {name, description} pairs, regardless of order.
func testAccCheckResourceLabelValuesImported(expected []map[string]string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported instance state, got %d", len(states))
		}
		attrs := states[0].Attributes

		count, err := strconv.Atoi(attrs["values.#"])
		if err != nil {
			return fmt.Errorf("could not parse values.# from imported state: %w", err)
		}
		if count != len(expected) {
			return fmt.Errorf("expected %d values, got %d", len(expected), count)
		}

		got := make([]map[string]string, 0, count)
		for i := 0; i < count; i++ {
			got = append(got, map[string]string{
				"name":        attrs[fmt.Sprintf("values.%d.name", i)],
				"description": attrs[fmt.Sprintf("values.%d.description", i)],
			})
		}

		for _, exp := range expected {
			found := false
			for _, g := range got {
				if g["name"] == exp["name"] && g["description"] == exp["description"] {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected value %v not found in imported values %v", exp, got)
			}
		}
		return nil
	}
}

func testAccResourceLabelImportStateId(resourceAddr, prefix string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return "", errs.NewNotFoundErrorf("%s in state", resourceAddr)
		}
		if rs.Primary.ID == "" {
			return "", errs.NewNotFoundErrorf("ID for %s in state", resourceAddr)
		}
		parts := strings.Split(rs.Primary.ID, "/")
		labelID := parts[len(parts)-1]
		return prefix + labelID, nil
	}
}

func testAccCheckBritiveResourceLabelConfig(resourceLabelName, resourceLabelDescription string) string {
	return fmt.Sprintf(`
	resource "britive_resource_manager_resource_label" "resource_label_1" {
		name         = "%s"
		description  = "%s"
		label_color  = "#abc123"

		values {
			name = "AT-Value-1"
			description = "AT-Value-1_Description"
		}
		values {
			name = "AT-Value-2"
			description = "AT-Value-2_Description"
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
