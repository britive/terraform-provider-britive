package tests

import (
	"os"
	"testing"

	"github.com/britive/terraform-provider-britive/britive"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/mitchellh/go-homedir"
)

const testVersion = "test"

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"britive": providerserver.NewProtocol6WithError(britive.New(testVersion)()),
}

// testAccPreCheckFramework validates that the environment is properly configured
// for Plugin Framework acceptance tests
func testAccPreCheckFramework(t *testing.T) {
	configPath, _ := homedir.Expand("~/.britive/tf.config")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		return
	}
	if tenant := os.Getenv("BRITIVE_TENANT"); tenant == "" {
		t.Fatal("BRITIVE_TENANT must be set for acceptance tests")
	}
	if token := os.Getenv("BRITIVE_TOKEN"); token == "" {
		t.Fatal("BRITIVE_TOKEN must be set for acceptance tests")
	}
}

// TestProviderFramework validates that the Framework provider can be instantiated
func TestProviderFramework(t *testing.T) {
	provider := britive.New(testVersion)()

	if provider == nil {
		t.Fatal("Provider should not be nil")
	}
}
