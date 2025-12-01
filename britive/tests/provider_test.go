package tests

import (
	"os"
	"regexp"
	"testing"

	"github.com/britive/terraform-provider-britive/britive"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"britive": providerserver.NewProtocol6WithError(britive.New()),
	}
)

func testAccPreCheck(t *testing.T) {
	if os.Getenv("BRITIVE_TENANT") == "" {
		t.Fatal("BRITIVE_TENANT must be set for acceptance tests")
	}
	if os.Getenv("BRITIVE_TOKEN") == "" {
		t.Fatal("BRITIVE_TOKEN must be set for acceptance tests")
	}
}

func TestProvider_Startup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Minimal working config
				Config: `
					provider "britive" {
						tenant = "https://example.com"
						token  = "dummy"
					}
				`,
			},
		},
	})
}

func TestAccProvider_InvalidTenant(t *testing.T) {

	config := `
	provider "britive" {
		tenant = "notaurl"
		token  = "dummy"
	}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Invalid Tenant URL"),
			},
		},
	})
}

func TestAccProvider_MissingToken(t *testing.T) {

	config := `
	provider "britive" {
		tenant = "https://example.com"
	}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Missing API Token"),
			},
		},
	})
}
