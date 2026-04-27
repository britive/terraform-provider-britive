package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestBritiveDataSourceUserByName looks up the acceptance-test service account by username.
func TestBritiveDataSourceUserByName(t *testing.T) {
	username := "britiveprovideracceptancetest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveDataSourceUserByNameConfig(username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.britive_user.by_name", "id"),
					resource.TestCheckResourceAttr("data.britive_user.by_name", "name", username),
					resource.TestCheckResourceAttrSet("data.britive_user.by_name", "user_id"),
				),
			},
		},
	})
}

func TestBritiveDataSourceUserByID(t *testing.T) {
	username := "britiveprovideracceptancetest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveDataSourceUserByIDConfig(username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.britive_user.by_id", "id"),
					resource.TestCheckResourceAttrSet("data.britive_user.by_id", "user_id"),
				),
			},
		},
	})
}

func testAccCheckBritiveDataSourceUserByNameConfig(username string) string {
	return fmt.Sprintf(`
	data "britive_user" "by_name" {
		name = "%s"
	}
	`, username)
}

func testAccCheckBritiveDataSourceUserByIDConfig(username string) string {
	return fmt.Sprintf(`
	data "britive_user" "by_name_first" {
		name = "%s"
	}

	data "britive_user" "by_id" {
		user_id = data.britive_user.by_name_first.user_id
	}
	`, username)
}
