package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDataApplication(t *testing.T) {
	resourceName := "data.britive_application.appDetails"
	applicationName := "AWS - Standalone - TFTest - DO NOT MODIFY"
	appId := "tgd5nbs7f6bv14cujd22"
	env_ids := []string{"k4pi3kuy2yzdczqk", "y6bd6eg8sf2i7bvn", "son96vpdo6o585dl"}
	env_group_ids := []string{"ljrf0zhhvldsu7o8", "czyjjidl2jz0a2k2", "1z684x4w9ylbmfy1", "xu7w30hkg210kk8d"}
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
			return fmt.Errorf("attribute not found: %s, value of setAttribute is %s", attributeName, setAttribute)
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
