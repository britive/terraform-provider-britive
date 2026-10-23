package tests

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/britive/terraform-provider-britive/britive-client-go"
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

// testAccBritiveClient builds an API client from BRITIVE_TENANT and
// BRITIVE_TOKEN, so tests can check what Terraform left behind in Britive.
func testAccBritiveClient(t *testing.T) *britive.Client {
	tenant := os.Getenv("BRITIVE_TENANT")
	token := os.Getenv("BRITIVE_TOKEN")
	if tenant == "" || token == "" {
		t.Skip("BRITIVE_TENANT and BRITIVE_TOKEN not set; skipping test that checks tags through the API")
	}
	if !strings.Contains(tenant, "://") {
		tenant = "https://" + tenant
	}
	client, err := britive.NewClient(fmt.Sprintf("%s/api", strings.TrimSuffix(tenant, "/")), token, testVersion, 0)
	if err != nil {
		t.Fatalf("creating Britive client: %v", err)
	}
	return client
}

// Connected-tag tests need an existing SCIM-connected tag; set
// BRITIVE_EXTERNAL_TAG_CONNECTED_ID to its id. The tag is read up front so the
// configs can mirror its current values: importing it then plans no changes,
// and the tests never alter the shared tag by accident.
func testAccConnectedExternalTag(t *testing.T) (*britive.Client, *britive.Tag) {
	id := os.Getenv("BRITIVE_EXTERNAL_TAG_CONNECTED_ID")
	if id == "" {
		t.Skip("BRITIVE_EXTERNAL_TAG_CONNECTED_ID not set; skipping connected external tag test")
	}
	client := testAccBritiveClient(t)
	tag, err := client.GetTag(id)
	if err != nil {
		t.Fatalf("reading connected tag %s: %v", id, err)
	}
	if !tag.External || len(tag.UserTagIdentityProviders) == 0 {
		t.Fatalf("tag %s is not connected through SCIM; BRITIVE_EXTERNAL_TAG_CONNECTED_ID must be a connected external tag", id)
	}
	return client, tag
}

// The connected-tag configs always declare requestable = true, so the tag must
// already be requestable; otherwise the first apply would change it.
func testAccRequireRequestable(t *testing.T, tag *britive.Tag) {
	if !tag.Requestable {
		t.Skipf("connected tag %s is not requestable; these tests keep requestable = true and must not change it", tag.ID)
	}
}

// Placeholder lifecycle: create, update every writable field, drop the
// description (the API's "No Description" default must not drift), import,
// and confirm destroy deletes the tag from Britive.
func TestBritiveExternalTagPlaceholder(t *testing.T) {
	idpID := testAccExternalTagIdentityProvider(t)
	client := testAccBritiveClient(t)
	name := "AT - External Tag Placeholder"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBritiveExternalTagDestroy(client),
		Steps: []resource.TestStep{
			{
				Config: testAccExternalTagConfig(name, "AT description", idpID, "v1"),
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
				// Rename, change description and the attribute value.
				Config: testAccExternalTagConfig(name+" Renamed", "AT description 2", idpID, "v2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_external_tag.new", "name", name+" Renamed"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "description", "AT description 2"),
					resource.TestCheckResourceAttr("britive_external_tag.new", "requestable", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("britive_external_tag.new", "attributes.*", map[string]string{
						"attribute_name":  "Owner",
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
				ExpectError: regexp.MustCompile(`(?s)External Tag Already Exists.*terraform\s+import`),
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

// A connected tag can be imported by id with a clean plan, and destroying it
// only removes it from state: the tag must still exist and stay connected.
func TestBritiveExternalTagImportConnected(t *testing.T) {
	client, tag := testAccConnectedExternalTag(t)
	testAccRequireRequestable(t, tag)
	config := testAccConnectedExternalTagConfig(tag, tag.Name, tag.Attributes)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConnectedExternalTagNotDeleted(client, tag.ID),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_external_tag.connected", "id", tag.ID),
					resource.TestCheckResourceAttr("britive_external_tag.connected", "name", tag.Name),
					resource.TestCheckResourceAttr("britive_external_tag.connected", "external", "true"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// The identity provider owns a connected tag's name, so a rename is refused at
// plan time.
func TestBritiveExternalTagRenameConnected(t *testing.T) {
	client, tag := testAccConnectedExternalTag(t)
	testAccRequireRequestable(t, tag)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConnectedExternalTagNotDeleted(client, tag.ID),
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedExternalTagConfig(tag, tag.Name, tag.Attributes),
			},
			{
				Config:      testAccConnectedExternalTagConfig(tag, tag.Name+" Renamed", tag.Attributes),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Cannot Rename Connected External Tag`),
			},
		},
	})
}

// With the identity provider's "Allow tag name and description to be editable"
// flag off, attributes stay writable on a connected tag. The last step puts
// the original attributes back; a cleanup does the same through the API in case
// a step fails first.
func TestBritiveExternalTagUpdateConnected(t *testing.T) {
	client, tag := testAccConnectedExternalTag(t)
	testAccRequireRequestable(t, tag)
	idp, err := client.GetIdentityProvider(tag.UserTagIdentityProviders[0].IdentityProvider.ID)
	if err != nil {
		t.Fatalf("reading identity provider of tag %s: %v", tag.ID, err)
	}
	if idp.AllowEditingExternalTags {
		t.Skipf("identity provider %s allows editing external tags; this test covers the flag-off case", idp.Name)
	}
	t.Cleanup(func() { testAccRestoreTagOverlay(t, client, tag) })

	updatedAttributes := []britive.TagAttribute{{AttributeName: "Owner", AttributeValue: "AT connected update"}}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckConnectedExternalTagNotDeleted(client, tag.ID),
		Steps: []resource.TestStep{
			{
				Config: testAccConnectedExternalTagConfig(tag, tag.Name, tag.Attributes),
			},
			{
				Config: testAccConnectedExternalTagConfig(tag, tag.Name, updatedAttributes),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_external_tag.connected", "requestable", "true"),
					resource.TestCheckResourceAttr("britive_external_tag.connected", "attributes.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("britive_external_tag.connected", "attributes.*", map[string]string{
						"attribute_name":  "Owner",
						"attribute_value": "AT connected update",
					}),
					resource.TestCheckResourceAttr("britive_external_tag.connected", "external", "true"),
				),
			},
			{
				Config: testAccConnectedExternalTagConfig(tag, tag.Name, tag.Attributes),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_external_tag.connected", "requestable", "true"),
					resource.TestCheckResourceAttr("britive_external_tag.connected", "attributes.#", fmt.Sprintf("%d", len(tag.Attributes))),
				),
			},
		},
	})
}

// Owners stay Terraform-manageable on a connected tag. The tag's current owners
// are imported first, the test adds a user owner and a tag owner next to them,
// then removes only those two. The last step drops the resource from state
// with a removed block instead of destroying it, because destroying
// britive_tag_owner clears the whole owner list. Every owner change must also
// leave the tag's requestable flag and attributes untouched.
func TestBritiveTagOwnerConnectedExternalTag(t *testing.T) {
	client, tag := testAccConnectedExternalTag(t)
	username := os.Getenv("BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME")
	if username == "" {
		t.Skip("BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME not set; skipping connected tag owner test")
	}
	current, err := client.GetTagWithOwners(tag.ID)
	if err != nil {
		t.Fatalf("reading owners of tag %s: %v", tag.ID, err)
	}
	existing := current.Relationships.Owners
	var existingUsers, existingTags int
	for _, o := range existing {
		switch o.RelatedEntityType {
		case "User":
			if strings.EqualFold(o.RelatedEntityName, username) {
				t.Skipf("user %s is already an owner of connected tag %s; set BRITIVE_EXTERNAL_TAG_MEMBER_USERNAME to a user that is not", username, tag.ID)
			}
			existingUsers++
		case "Tag":
			existingTags++
		default:
			// britive_tag_owner only knows users and tags; applying it would drop this owner.
			t.Skipf("connected tag %s has a %q owner, which britive_tag_owner cannot manage", tag.ID, o.RelatedEntityType)
		}
	}
	t.Cleanup(func() { testAccRestoreTagOwners(t, client, tag, existing) })

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckFramework(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckTagOwnersEqual(client, tag.ID, existing),
			testAccCheckTagOverlayUnchanged(client, tag),
		),
		Steps: []resource.TestStep{
			{
				// Import the current owners unchanged.
				Config: testAccConnectedTagOwnerConfig(tag.ID, existing, "", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBritiveTagOwnerExists("britive_tag_owner.connected"),
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "user.#", fmt.Sprintf("%d", existingUsers)),
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "tag.#", fmt.Sprintf("%d", existingTags)),
					testAccCheckTagOwnersEqual(client, tag.ID, existing),
				),
			},
			{
				// Add a user owner.
				Config: testAccConnectedTagOwnerConfig(tag.ID, existing, username, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "user.#", fmt.Sprintf("%d", existingUsers+1)),
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "tag.#", fmt.Sprintf("%d", existingTags)),
					testAccCheckTagOverlayUnchanged(client, tag),
				),
			},
			{
				// Add a tag owner as well.
				Config: testAccConnectedTagOwnerConfig(tag.ID, existing, username, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "user.#", fmt.Sprintf("%d", existingUsers+1)),
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "tag.#", fmt.Sprintf("%d", existingTags+1)),
					testAccCheckTagOverlayUnchanged(client, tag),
				),
			},
			{
				// Remove both added owners; the existing ones must be untouched.
				Config: testAccConnectedTagOwnerConfig(tag.ID, existing, "", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "user.#", fmt.Sprintf("%d", existingUsers)),
					resource.TestCheckResourceAttr("britive_tag_owner.connected", "tag.#", fmt.Sprintf("%d", existingTags)),
					testAccCheckTagOwnersEqual(client, tag.ID, existing),
					testAccCheckTagOverlayUnchanged(client, tag),
				),
			},
			{
				// Hand the owners back without clearing them.
				Config: `
	removed {
		from = britive_tag_owner.connected

		lifecycle {
			destroy = false
		}
	}`,
			},
		},
	})
}

func testAccExternalTagConfig(name, description, idpID, attrValue string) string {
	return fmt.Sprintf(`
	resource "britive_external_tag" "new" {
		name                 = %q
		description          = %q
		identity_provider_id = %q
		requestable          = true

		attributes {
			attribute_name  = "Owner"
			attribute_value = %q
		}
	}`, name, description, idpID, attrValue)
}

func testAccExternalTagConfigNoDescription(name, idpID string) string {
	return fmt.Sprintf(`
	resource "britive_external_tag" "new" {
		name                 = %q
		identity_provider_id = %q
		requestable          = true

		attributes {
			attribute_name  = "Owner"
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

// testAccConnectedExternalTagConfig imports the connected tag and declares it
// with the given name and attributes and requestable = true; description and
// identity provider are taken from the tag so they never show as a change.
func testAccConnectedExternalTagConfig(tag *britive.Tag, name string, attributes []britive.TagAttribute) string {
	var b strings.Builder
	fmt.Fprintf(&b, `
	import {
		to = britive_external_tag.connected
		id = %q
	}

	resource "britive_external_tag" "connected" {
		name                 = %q
		identity_provider_id = %q
		requestable          = true
`, tag.ID, name, tag.UserTagIdentityProviders[0].IdentityProvider.ID)
	if tag.Description != "" && tag.Description != "No Description" {
		fmt.Fprintf(&b, "\t\tdescription          = %q\n", tag.Description)
	}
	for _, a := range attributes {
		fmt.Fprintf(&b, `
		attributes {
			attribute_name  = %q
			attribute_value = %q
		}
`, a.AttributeName, a.AttributeValue)
	}
	b.WriteString("\t}\n")
	return b.String()
}

// testAccRestoreTagOverlay puts the tag's original requestable flag and
// attributes back through the API.
func testAccRestoreTagOverlay(t *testing.T, client *britive.Client, tag *britive.Tag) {
	requestable := tag.Requestable
	attributes := make([]britive.TagAttribute, 0, len(tag.Attributes))
	for _, a := range tag.Attributes {
		attributes = append(attributes, britive.TagAttribute{AttributeName: a.AttributeName, AttributeValue: a.AttributeValue})
	}
	_, err := client.UpdateTagAttributes(tag.ID, britive.TagAttributesUpdateRequest{
		UserTagID:   tag.ID,
		Name:        tag.Name,
		Description: tag.Description,
		Requestable: &requestable,
		Attributes:  attributes,
	})
	if err != nil {
		t.Errorf("restoring requestable and attributes of tag %s: %v", tag.ID, err)
	}
}

func testAccCheckBritiveExternalTagDestroy(client *britive.Client) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "britive_external_tag" {
				continue
			}
			_, err := client.GetTag(rs.Primary.ID)
			if errors.Is(err, britive.ErrNotFound) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking tag %s after destroy: %w", rs.Primary.ID, err)
			}
			return fmt.Errorf("tag %s still exists in Britive after destroy", rs.Primary.ID)
		}
		return nil
	}
}

func testAccCheckConnectedExternalTagNotDeleted(client *britive.Client, tagID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		tag, err := client.GetTag(tagID)
		if err != nil {
			return fmt.Errorf("connected tag %s should still exist after destroy: %w", tagID, err)
		}
		if !tag.External {
			return fmt.Errorf("tag %s is no longer connected after destroy", tagID)
		}
		return nil
	}
}

// testAccConnectedTagOwnerConfig imports the tag's owners and declares the
// existing ones by id, plus the user named newUser and a test-created tag owner
// when asked.
func testAccConnectedTagOwnerConfig(tagID string, existing []britive.TagOwnerEntity, newUser string, newTagOwner bool) string {
	var b strings.Builder
	if newTagOwner {
		b.WriteString(`
	data "britive_identity_provider" "default" {
		name = "Britive"
	}

	resource "britive_tag" "owner" {
		name                 = "AT - Connected External Tag Owner"
		identity_provider_id = data.britive_identity_provider.default.id
	}
`)
	}
	fmt.Fprintf(&b, `
	import {
		to = britive_tag_owner.connected
		id = %q
	}

	resource "britive_tag_owner" "connected" {
		tag_id = %q
`, tagID, tagID)
	for _, o := range existing {
		block := "user"
		if o.RelatedEntityType == "Tag" {
			block = "tag"
		}
		fmt.Fprintf(&b, `
		%s {
			id = %q
		}
`, block, o.RelatedEntityID)
	}
	if newUser != "" {
		fmt.Fprintf(&b, `
		user {
			name = %q
		}
`, newUser)
	}
	if newTagOwner {
		b.WriteString(`
		tag {
			id = britive_tag.owner.id
		}
`)
	}
	b.WriteString("\t}\n")
	return b.String()
}

// testAccRestoreTagOwners puts the tag's original owners back through the API.
func testAccRestoreTagOwners(t *testing.T, client *britive.Client, tag *britive.Tag, owners []britive.TagOwnerEntity) {
	requestable := tag.Requestable
	attributes := make([]britive.TagAttribute, 0, len(tag.Attributes))
	for _, a := range tag.Attributes {
		attributes = append(attributes, britive.TagAttribute{AttributeName: a.AttributeName, AttributeValue: a.AttributeValue})
	}
	_, err := client.UpdateTagOwners(britive.TagWithOwners{
		TagID:         tag.ID,
		Name:          tag.Name,
		Description:   tag.Description,
		Requestable:   &requestable,
		Attributes:    attributes,
		Relationships: britive.TagOwnerRelationships{Owners: owners},
	})
	if err != nil {
		t.Errorf("restoring owners of tag %s: %v", tag.ID, err)
	}
}

// testAccCheckTagOwnersEqual fails when the tag's owners in Britive differ
// from want, compared by type and id.
func testAccCheckTagOwnersEqual(client *britive.Client, tagID string, want []britive.TagOwnerEntity) resource.TestCheckFunc {
	return func(*terraform.State) error {
		current, err := client.GetTagWithOwners(tagID)
		if err != nil {
			return fmt.Errorf("reading owners of tag %s: %w", tagID, err)
		}
		got := current.Relationships.Owners
		if len(got) != len(want) {
			return fmt.Errorf("tag %s has %d owner(s), want %d: %v", tagID, len(got), len(want), got)
		}
		counts := make(map[string]int, len(want))
		for _, o := range want {
			counts[o.RelatedEntityType+"/"+o.RelatedEntityID]++
		}
		for _, o := range got {
			k := o.RelatedEntityType + "/" + o.RelatedEntityID
			if counts[k] == 0 {
				return fmt.Errorf("tag %s has unexpected owner %s (%s)", tagID, k, o.RelatedEntityName)
			}
			counts[k]--
		}
		return nil
	}
}

// testAccCheckTagOverlayUnchanged fails when the tag's requestable flag or
// attributes in Britive differ from want.
func testAccCheckTagOverlayUnchanged(client *britive.Client, want *britive.Tag) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, err := client.GetTag(want.ID)
		if err != nil {
			return fmt.Errorf("reading tag %s: %w", want.ID, err)
		}
		if got.Requestable != want.Requestable {
			return fmt.Errorf("requestable of tag %s changed from %t to %t", want.ID, want.Requestable, got.Requestable)
		}
		if !sameTagAttributes(got.Attributes, want.Attributes) {
			return fmt.Errorf("attributes of tag %s changed from %v to %v", want.ID, want.Attributes, got.Attributes)
		}
		return nil
	}
}

func sameTagAttributes(a, b []britive.TagAttribute) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, x := range a {
		counts[x.AttributeName+"="+x.AttributeValue]++
	}
	for _, x := range b {
		k := x.AttributeName + "=" + x.AttributeValue
		if counts[k] == 0 {
			return false
		}
		counts[k]--
	}
	return true
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
