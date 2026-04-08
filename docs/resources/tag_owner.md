---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_tag_owner Resource - britive"
description: |-
  Manages tag owners for the Britive provider.
---

# britive_tag_owner Resource

This resource allows you to manage owners of a Britive tag. Both users and other tags can be configured as owners. Each `user` or `tag` block must specify either `id` or `name`, not both.

## Example Usage

```hcl
data "britive_identity_provider" "idp" {
    name = "Britive"
}

resource "britive_tag" "new" {
    name                 = "My Tag"
    description          = "My Tag Description"
    identity_provider_id = data.britive_identity_provider.idp.id
}

resource "britive_tag" "owner_tag" {
    name                 = "Owner Tag"
    description          = "Owner Tag Description"
    identity_provider_id = data.britive_identity_provider.idp.id
}

resource "britive_tag_owner" "new" {
    tag_id = britive_tag.new.id

    user {
        name = "myusername"
    }

    user {
        id = "user-id-123"
    }

    tag {
        id = britive_tag.owner_tag.id
    }

    tag {
        name = "tag-name-456"
    }
}
```

## Argument Reference

The following arguments are supported:

* `tag_id` - (Required, ForceNew) The identifier of the Britive tag whose owners are being managed.

* `user` - (Optional) One or more blocks defining user owners. Each block supports:
  * `id` - (Optional) The identifier of the user. Specify either `id` or `name`, not both.
  * `name` - (Optional) The username of the user. Specify either `id` or `name`, not both.

* `tag` - (Optional) One or more blocks defining tag owners. Each block supports:
  * `id` - (Optional) The identifier of the tag. Specify either `id` or `name`, not both.
  * `name` - (Optional) The name of the tag. Specify either `id` or `name`, not both.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with format `tags/{{tagID}}/owners`

## Import

You can import the Britive tag owner using any of these accepted formats:

```sh
terraform import britive_tag_owner.new tags/{{tagID}}/owners
terraform import britive_tag_owner.new {{tagID}}
```
