package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBritiveDataSourceNotificationMedium(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				data "britive_notification_medium" "email" {
					name = "Email"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.britive_notification_medium.email", "id"),
					resource.TestCheckResourceAttr("data.britive_notification_medium.email", "name", "Email"),
					resource.TestCheckResourceAttrSet("data.britive_notification_medium.email", "type"),
				),
			},
		},
	})
}
