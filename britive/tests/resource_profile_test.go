package tests

import (
	"fmt"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBritiveProfile(t *testing.T) {
	name := "AT - New Britive Profile Test"
	description := "AT - New Britive Profile Test Description"
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	resourceName := "britive_profile.new"
	associationType := "EnvironmentGroup"
	associationValue := "QA"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileConfig(name, description, applicationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
			{
				Config: testAccCheckBritiveProfileConfigAddAssociations(name, description, applicationName, associationType, associationValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "associations.0.type", associationType),
					resource.TestCheckResourceAttr(resourceName, "associations.0.value", associationValue),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileConfig(name string, description string, applicationName string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		expiration_duration = "25m0s"
		allow_impersonation  = true
		associations {
			type  = "Environment"
			value = "Subscription 1"
		}
	}`, applicationName, name, description)
}

func testAccCheckBritiveProfileConfigAddAssociations(name, description, applicationName, associationType, associationValue string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id = data.britive_application.app.id
		name = "%s"
		description = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "%s"
			value = "%s"
		}
	}`, applicationName, name, description, associationType, associationValue)
}

func TestBritiveProfileTagAssociations(t *testing.T) {
	name := "AT - New Britive Profile Tag Association Test"
	description := "AT - New Britive Profile Tag Association Test Description"
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	resourceName := "britive_profile.new"
	tagKey := "team"
	tagValue := "engineering"
	associationType := "EnvironmentGroup"
	associationValue := "QA"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileConfigWithTagAssociations(name, description, applicationName, tagKey, tagValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "tag_associations.#", "1"),
				),
			},
			{
				Config: testAccCheckBritiveProfileConfigWithBothAssociations(name, description, applicationName, associationType, associationValue, tagKey, tagValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "associations.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tag_associations.#", "1"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileConfigWithTagAssociations(name, description, applicationName, tagKey, tagValue string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id    = data.britive_application.app.id
		name                = "%s"
		description         = "%s"
		expiration_duration = "25m0s"
		tag_associations {
			key    = "%s"
			values = ["%s"]
		}
	}`, applicationName, name, description, tagKey, tagValue)
}

func testAccCheckBritiveProfileConfigWithBothAssociations(name, description, applicationName, associationType, associationValue, tagKey, tagValue string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "new" {
		app_container_id    = data.britive_application.app.id
		name                = "%s"
		description         = "%s"
		expiration_duration = "25m0s"
		associations {
			type  = "%s"
			value = "%s"
		}
		tag_associations {
			key    = "%s"
			values = ["%s"]
		}
	}`, applicationName, name, description, associationType, associationValue, tagKey, tagValue)
}

func testAccCheckBritiveProfileExists(n string) resource.TestCheckFunc {
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

// TestBritiveProfileForEachDynamicAssociations is a regression test: count on the
// resource combined with a dynamic "associations" block driven by count.index caused
// Terraform to validate the un-expanded resource with count.index left unknown. A Set
// containing an unknown element collapses to a wholly unknown value, which previously
// crashed ValidateConfig's req.Config.Get into []ProfileAssociationModel.
//
// count (not for_each) is used here deliberately: terraform-plugin-testing's legacy
// state shim used by TestCheckResourceAttr/RootModule().Resources lookups does not
// support for_each string-keyed instances ("unexpected index type (string) ...,
// for_each is not supported"), only count's integer-keyed instances.
func TestBritiveProfileForEachDynamicAssociations(t *testing.T) {
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	name := "AT - New Britive Profile ForEach Dynamic Test"
	resourceName := "britive_profile.iterated.0"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileForEachDynamicAssociationsConfig(applicationName, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "associations.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "associations.0.type", "Environment"),
					resource.TestCheckResourceAttr(resourceName, "associations.0.value", "Subscription 1"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileForEachDynamicAssociationsConfig(applicationName, name string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	locals {
		profiles = [
			{
				name         = "%s"
				associations = [{ type = "Environment", value = "Subscription 1" }]
			}
		]
	}

	resource "britive_profile" "iterated" {
		count = length(local.profiles)

		app_container_id    = data.britive_application.app.id
		name                 = local.profiles[count.index].name
		expiration_duration  = "25m0s"

		dynamic "associations" {
			for_each = local.profiles[count.index].associations
			content {
				type  = associations.value.type
				value = associations.value.value
			}
		}
	}`, applicationName, name)
}

// TestBritiveProfileUnknownAssociationsFromDependency is a regression test: the
// "associations" dynamic block is gated by britive_profile.trigger.id, an attribute
// only known after apply since "trigger" is created in the same operation. This keeps
// the associations Set unknown all the way through PlanResourceChange (unlike the
// for_each+each.value case, which resolves by plan time), which previously crashed
// ModifyPlan's req.Plan.Get into []ProfileAssociationModel.
func TestBritiveProfileUnknownAssociationsFromDependency(t *testing.T) {
	applicationName := "DO NOT DELETE - Azure TF Plugin"
	triggerName := "AT - New Britive Profile Dependency Trigger"
	name := "AT - New Britive Profile Unknown Associations Test"
	resourceName := "britive_profile.modplan"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBritiveProfileUnknownAssociationsFromDependencyConfig(applicationName, triggerName, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveProfileExists(resourceName),
					testAccCheckBritiveProfileExists("britive_profile.trigger"),
					resource.TestCheckResourceAttr(resourceName, "associations.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "associations.0.type", "Environment"),
					resource.TestCheckResourceAttr(resourceName, "associations.0.value", "Subscription 1"),
				),
			},
		},
	})
}

func testAccCheckBritiveProfileUnknownAssociationsFromDependencyConfig(applicationName, triggerName, name string) string {
	return fmt.Sprintf(`
	data "britive_application" "app" {
		name = "%s"
	}

	resource "britive_profile" "trigger" {
		app_container_id    = data.britive_application.app.id
		name                 = "%s"
		expiration_duration  = "25m0s"
	}

	resource "britive_profile" "modplan" {
		app_container_id    = data.britive_application.app.id
		name                 = "%s"
		expiration_duration  = "25m0s"

		dynamic "associations" {
			for_each = britive_profile.trigger.id == "" ? [] : [{ type = "Environment", value = "Subscription 1" }]
			content {
				type  = associations.value.type
				value = associations.value.value
			}
		}
	}`, applicationName, triggerName, name)
}
