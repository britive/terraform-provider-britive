package britive

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"britive": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testAccPreCheck(t *testing.T) {
	configPath, _ := homedir.Expand("~/.britive/tf.config")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		return
	}
	if err := os.Getenv("BRITIVE_TENANT"); err == "" {
		t.Fatal("BRITIVE_TENANT must be set for acceptance tests")
	}
	if err := os.Getenv("BRITIVE_TOKEN"); err == "" {
		t.Fatal("BRITIVE_TOKEN must be set for acceptance tests")
	}
}
