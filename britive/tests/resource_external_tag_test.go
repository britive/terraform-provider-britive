package tests

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// External tag tests need a non-default (SAML) identity provider on the test
// tenant. Set BRITIVE_EXTERNAL_TAG_IDP_ID to its id to enable them.
func testAccExternalTagIdentityProvider(t *testing.T) string {
	id := os.Getenv("BRITIVE_EXTERNAL_TAG_IDP_ID")
	if id == "" {
		t.Skip("BRITIVE_EXTERNAL_TAG_IDP_ID not set; skipping external tag tests")
	}
	return id
}

// Placeholder lifecycle: create, update every writable field, drop the
// description (the API's "No Description" default must not drift), import.
func TestBritiveExternalTagPlaceholder(t *testing.T) {
	idpID := testAccExternalTagIdentityProvider(t)
	name := "AT - External Tag Placeholder"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExternalTagConfig(name, "AT description", idpID, "true", "v1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveExternalTagExists("britive_external_tag.new"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "name", name),
					resource.TestCheckResourceAttr("britive_external_tag.new", "external", "false"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "external_id", ""),
					resource.TestCheckResourceAttr("britive_external_tag.new", "requestable", "true"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "identity_provider_id", idpID),
					resource.TestCheckResourceAttr("britive_external_tag.new", "attributes.#", "1"),
				),
			},
			{
				// Rename, change description, requestable and the attribute value.
				Config: testAccExternalTagConfig(name+" Renamed", "AT description 2", idpID, "false", "v2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_external_tag.new", "name", name+" Renamed"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "description", "AT description 2"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "requestable", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("britive_external_tag.new", "attributes.*", map[string]string{
						"attribute_name":  "TagAttr1",
						"attribute_value": "v2",
					}),
				),
			},
			{
				// Omitting description must not produce drift against the API default.
				Config: testAccExternalTagConfigNoDescription(name+" Renamed", idpID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("britive_external_tag.new", "description"),
				),
			},
			{
				Config:   testAccExternalTagConfigNoDescription(name+" Renamed", idpID),
				PlanOnly: true,
			},
			{
				ResourceName:      "britive_external_tag.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// The default identity provider is refused at plan time.
func TestBritiveExternalTagRefusesDefaultIdentityProvider(t *testing.T) {
	testAccExternalTagIdentityProvider(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
	data "britive_identity_provider" "default" {
		name = "Britive"
	}

	resource "britive_external_tag" "bad" {
		name                 = "AT - External Tag On Default IdP"
		identity_provider_id = data.britive_identity_provider.default.id
	}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Not An External Tag`),
			},
		},
	})
}

// britive_tag refuses the non-default provider and points at this resource.
func TestBritiveTagRefusesExternalIdentityProvider(t *testing.T) {
	idpID := testAccExternalTagIdentityProvider(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
	resource "britive_tag" "bad" {
		name                 = "AT - Britive Tag On SAML IdP"
		identity_provider_id = %q
	}`, idpID),
				ExpectError: regexp.MustCompile(`britive_external_tag`),
			},
		},
	})
}

// A placeholder can take members through britive_tag_member.
func TestBritiveExternalTagPlaceholderMember(t *testing.T) {
	idpID := testAccExternalTagIdentityProvider(t)
	username := os.Getenv("BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME")
	if username == "" {
		t.Skip("BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME not set; skipping member test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
	resource "britive_external_tag" "new" {
		name                 = "AT - External Tag With Member"
		identity_provider_id = %q
	}

	resource "britive_tag_member" "new" {
		tag_id   = britive_external_tag.new.id
		username = %q
	}`, idpID, username),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveExternalTagExists("britive_external_tag.new"),
					resource.TestCheckResourceAttrPair("britive_tag_member.new", "tag_id", "britive_external_tag.new", "id"),
				),
			},
		},
	})
}

// The data source refuses an ambiguous name and resolves it with a provider id.
func TestBritiveTagDataSourceDisambiguation(t *testing.T) {
	idpID := testAccExternalTagIdentityProvider(t)
	name := "AT - Duplicate Tag Name"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Same name on the default provider (britive_tag) and the SAML provider.
				Config: testAccDuplicateTagNameConfig(name, idpID),
			},
			{
				Config: testAccDuplicateTagNameConfig(name, idpID) + fmt.Sprintf(`
	data "britive_tag" "ambiguous" {
		name = %q
		depends_on = [britive_tag.dup, britive_external_tag.dup]
	}`, name),
				ExpectError: regexp.MustCompile(`Ambiguous Tag Name`),
			},
			{
				Config: testAccDuplicateTagNameConfig(name, idpID) + fmt.Sprintf(`
	data "britive_tag" "resolved" {
		name                 = %q
		identity_provider_id = %q
		depends_on           = [britive_tag.dup, britive_external_tag.dup]
	}`, name, idpID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.britive_tag.resolved", "id", "britive_external_tag.dup", "id"),
					resource.TestCheckResourceAttr("data.britive_tag.resolved", "external", "false"),
				),
			},
		},
	})
}

// A second resource for a name that already exists on the same provider is
// refused at plan time with import guidance (the identity-provider-first case).
func TestBritiveExternalTagAlreadyExists(t *testing.T) {
	idpID := testAccExternalTagIdentityProvider(t)
	name := "AT - External Tag Already Exists"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExternalTagConfigNoDescription(name, idpID),
			},
			{
				Config: testAccExternalTagConfigNoDescription(name, idpID) + fmt.Sprintf(`
	resource "britive_external_tag" "second" {
		name                 = %q
		identity_provider_id = %q
		depends_on           = [britive_external_tag.new]
	}`, name, idpID),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`External Tag Already Exists`),
			},
		},
	})
}

// Adding a member to a connected external tag is refused at plan time. Needs
// an existing SCIM-connected tag id in BRITIVE_EXTERNAL_TAG_CONNECTED_ID.
func TestBritiveTagMemberRefusesConnectedExternalTag(t *testing.T) {
	testAccExternalTagIdentityProvider(t)
	connectedID := os.Getenv("BRITIVE_EXTERNAL_TAG_CONNECTED_ID")
	username := os.Getenv("BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME")
	if connectedID == "" || username == "" {
		t.Skip("BRITIVE_EXTERNAL_TAG_CONNECTED_ID or BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME not set; skipping")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
	resource "britive_tag_member" "connected" {
		tag_id   = %q
		username = %q
	}`, connectedID, username),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Cannot Add Member To Connected External Tag`),
			},
		},
	})
}

func testAccExternalTagConfig(name, description, idpID, requestable, attrValue string) string {
	return fmt.Sprintf(`
	resource "britive_external_tag" "new" {
		name                 = %q
		description          = %q
		identity_provider_id = %q
		requestable          = %s

		attributes {
			attribute_name  = "TagAttr1"
			attribute_value = %q
		}
	}`, name, description, idpID, requestable, attrValue)
}

func testAccExternalTagConfigNoDescription(name, idpID string) string {
	return fmt.Sprintf(`
	resource "britive_external_tag" "new" {
		name                 = %q
		identity_provider_id = %q
		requestable          = false

		attributes {
			attribute_name  = "TagAttr1"
			attribute_value = "v2"
		}
	}`, name, idpID)
}

func testAccDuplicateTagNameConfig(name, idpID string) string {
	return fmt.Sprintf(`
	data "britive_identity_provider" "default" {
		name = "Britive"
	}

	resource "britive_tag" "dup" {
		name                 = %q
		identity_provider_id = data.britive_identity_provider.default.id
	}

	resource "britive_external_tag" "dup" {
		name                 = %q
		identity_provider_id = %q
	}`, name, name, idpID)
}

func testAccCheckBritiveExternalTagExists(n string) resource.TestCheckFunc {
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
