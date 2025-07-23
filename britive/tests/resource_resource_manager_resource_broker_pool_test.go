package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestBritiveResourceResourceBrokerPool(t *testing.T) {
	resourceTypeName := "AT-Britive_Resource_Manager_Tests_Resource_Type_1"
	resourceTypeDescription := "AT-Britive_Resource_Manager_Tests_Resource_Type_1_Description"
	resourceLabelName1 := "AT-Britive_Resource_Manager_Test_Resource_Label_111"
	resourceLabelDescription1 := "AT-Britive_Resource_Manager_Test_Resource_Label_111_Description"
	resourceResourceName := "AT-Britive_Resource_Tests_Resource_1"
	resourceResourceDescription := "AT-Britive_Resource_Test_Resource_Description_1"
	resourceResourceBrokerPoolName := "DO NOT DELETE - BROKER POOL TF Plugin"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveResourceResourceBrokerPoolConfig(resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription, resourceResourceBrokerPoolName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveResourceResourceBrokerPoolExists("britive_resource_manager_resource_type.resource_type_1"),
					testAccCheckBritiveResourceResourceBrokerPoolExists("britive_resource_manager_resource_label.resource_label_1"),
					testAccCheckBritiveResourceResourceBrokerPoolExists("britive_resource_manager_resource.resource_1"),
					testAccCheckBritiveResourceResourceBrokerPoolExists("britive_resource_manager_resource_broker_pools.resource_broker_pool_1"),
				),
			},
		},
	})
}

func testAccCheckBritiveResourceResourceBrokerPoolConfig(resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription, resourceResourceBrokerPoolName string) string {
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
			"${britive_resource_manager_resource_label.resource_label_1.name}" = "Production,Development"
		}
	}

	resource "britive_resource_manager_resource_broker_pools" "resource_broker_pool_1" {
		resource_id = britive_resource_manager_resource.resource_1.id
		broker_pools = ["%s"]
	}

	`, resourceTypeName, resourceTypeDescription, resourceLabelName1, resourceLabelDescription1, resourceResourceName, resourceResourceDescription, resourceResourceBrokerPoolName)
}

func testAccCheckBritiveResourceResourceBrokerPoolExists(n string) resource.TestCheckFunc {
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
