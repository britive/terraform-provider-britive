---
subcategory: "Tag"
layout: "Britive"
page_title: "Britive: britive_tag"
description: |-
  Creates a Britive Tag.
---

# britive\_tag

Manages Britive Tag.

## Example Usage

```hcl
data "britive_identity_provider" "idp" {
    name = "My Identity Provider"
}

resource "britive_tag" "new" {
    name = "My Tag"
    description = "My Tag Description"
    status = "Active"
    user_tag_identity_providers {
        identity_provider {
            id = data.britive_identity_provider.idp.id
        }
    }
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of tag.
* `description` - (Optional) A description of the tag.
* `status` - (Required) The status of the tag.
* `user_tag_identity_providers` - (Required) The list of identity providers associated with the tag. Structure is documented below.

### `user_tag_identity_providers` block supports

* `identity_provider` - An identity provider. Structure is documented below.

### `identity_provider` block supports

* `id` - The unique ID of the identity-provider.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - An identifier for the resource.
* `external` - Boolean - If the tag is external or not.
* `user_count` - The number of users associated with the tag.



