package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDataApplication(t *testing.T) {
	resourceName := "data.britive_application.appDetails"
	applicationName := "Azure-ValueLabs"
	appId := "cl6vvfhzkjdxjcbtzr8j"
	env_ids := []string{"64738ffe-22fe-40fb-9380-8b5af077d244", "67e131ab-f3d7-4fed-b1f9-c8fe634bc3b5", "9d635ded-0fa9-4035-8767-c312e47ac537"}
	env_group_ids := []string{"cl6vvfhzkjdxjcbtzr8j", "6ed83eea-639b-44ef-a860-486761c02803", "development1", "POC1", "production1", "QA1", "stage1", "UI", "Jennifer"}
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataApplicationConfig(applicationName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", appId),
					testAccCheckDataApplicationSet(resourceName, "environment_ids", env_ids),
					testAccCheckDataApplicationSet(resourceName, "environment_group_ids", env_group_ids),
				),
			},
		},
	})
}

func testAccCheckDataApplicationConfig(applicationName string) string {
	return fmt.Sprintf(`
	data "britive_application" "appDetails" {
		name = "%s"
	}`, applicationName)
}

func testAccCheckDataApplicationSet(resourceName, attributeName string, expectedValues []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		setAttribute, ok := rs.Primary.Attributes[attributeName+".%"]
		if !ok {
			return fmt.Errorf("attribute not found: %s", attributeName)
		}

		setSize := len(expectedValues)
		if setAttribute != fmt.Sprintf("%d", setSize) {
			return fmt.Errorf("expected set size %d, got %s", setSize, setAttribute)
		}

		for _, value := range expectedValues {
			if _, ok := rs.Primary.Attributes[fmt.Sprintf("%s.%s", attributeName, value)]; !ok {
				return fmt.Errorf("expected value %s in set attribute %s", value, attributeName)
			}
		}

		return nil
	}
}
