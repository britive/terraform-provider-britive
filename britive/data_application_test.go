package britive

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataApplication(t *testing.T) {
	applicationName := "Azure-ValueLabs"
	appId := "cl6vvfhzkjdxjcbtzr8j"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataApplicationConfig(applicationName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.britive_application.appDetails", "id", appId),
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
