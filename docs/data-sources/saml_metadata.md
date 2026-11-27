---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_saml_metadata Data Source - britive"
description: |-
  Retrieves the Britive tenant's SAML identity provider metadata and certificates.
---

# britive_saml_metadata Data Source

Use this data source to retrieve the Britive tenant's SAML identity provider metadata XML, the SAML signing certificate and the OAuth2 certificate.

## Example Usage

```hcl
data "britive_saml_metadata" "this" {}

resource "aws_iam_saml_provider" "britive" {
  name                   = "Britive"
  saml_metadata_document = data.britive_saml_metadata.this.metadata
}
```

## Argument Reference

This data source takes no arguments. It reads the tenant's SAML configuration; a tenant has exactly one.

## Attribute Reference

The following attributes are exported:

* `id` - The identifier of the tenant's SAML configuration.
* `metadata` - The SAML 2.0 identity provider metadata XML.
* `issuer` - The SAML identity provider entity ID.
* `sign_in_url` - The SAML sign-in URL.
* `sign_out_url` - The SAML sign-out URL.
* `certificate_expiration_date` - The expiration date of the SAML signing certificate.
* `saml_certificate` - The PEM-encoded SAML signing certificate, the same certificate embedded in `metadata`.
* `oauth2_certificate` - The tenant's PEM-encoded OAuth2 certificate.
