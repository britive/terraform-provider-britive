package tests

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestBritiveDataSourceSAMLMetadata reads the tenant's SAML IdP metadata and certificates.
func TestBritiveDataSourceSAMLMetadata(t *testing.T) {
	pemCertificate := regexp.MustCompile(`^-----BEGIN CERTIFICATE-----`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "britive_saml_metadata" "this" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.britive_saml_metadata.this", "id"),
					resource.TestMatchResourceAttr("data.britive_saml_metadata.this", "metadata", regexp.MustCompile(`EntityDescriptor`)),
					resource.TestCheckResourceAttrSet("data.britive_saml_metadata.this", "issuer"),
					resource.TestCheckResourceAttrSet("data.britive_saml_metadata.this", "sign_in_url"),
					resource.TestCheckResourceAttrSet("data.britive_saml_metadata.this", "sign_out_url"),
					resource.TestCheckResourceAttrSet("data.britive_saml_metadata.this", "certificate_expiration_date"),
					resource.TestMatchResourceAttr("data.britive_saml_metadata.this", "saml_certificate", pemCertificate),
					resource.TestMatchResourceAttr("data.britive_saml_metadata.this", "oauth2_certificate", pemCertificate),
				),
			},
		},
	})
}
