package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritivePolicyPriority(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritivePolicyPriorityConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritivePolicyPriorityExists("britive_policy_priority.new_priority"),
				),
			},
		},
	})
}

func testAccCheckBritivePolicyPriorityConfig() string {
	return fmt.Sprintf(`
		
	resource "britive_policy_priority" "new_priority" {
    	profile_id = "g5ukwnd0j4lfiuardkzs"
		policy_priority {
      		id = "e50e2efd-fecc-4515-82e9-1411d4cbb0f9"
      		priority =0
    	}
	}
		`)

}

func testAccCheckBritivePolicyPriorityExists(n string) resource.TestCheckFunc {
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
