---
subcategory: "Identity Provider"
layout: "Britive"
page_title: "Britive: britive_identity_provider"
description: |-
  Get information about identity provider.
---

# britive\_identity\_provider

Get information about identity provider.

## Example Usage

```hcl
data "britive_identity_provider" "britive" {
    name = "Britive"
}
resource "britive_tag" "new" {
    # ...

    user_tag_identity_providers {
        identity_provider {
            id = data.britive_identity_provider.idp.id
        }
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the identity provider.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - an identifier for the data source

