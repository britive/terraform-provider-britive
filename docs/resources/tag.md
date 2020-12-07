# britive_tag Resource

Creates a Tag.

This resource allows you to create and configure a Tag.

__A tag represents a group of users in the Britive system.__

## Example Usage

```hcl
data "britive_identity_provider" "idp" {
    name = "My Identity Provider"
}

resource "britive_tag" "new" {
    name = "My Tag"
    description = "My Tag Description"
    user_tag_identity_providers {
        identity_provider {
            id = data.britive_identity_provider.idp.id
        }
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required): The name of tag.

* `description` - (Optional): A description of the tag.

* `disabled` - (Optional): Default tag is enabled. To disable tag set `disabled = true`.

* `user_tag_identity_providers` - (Required): The list of identity providers associated with the tag.

  The format of `user_tag_identity_providers` is documented below.

### `user_tag_identity_providers` block supports

* `identity_provider` - An identity provider.

   The format of `identity_provider` is documented below.

### `identity_provider` block supports

* `id` - The unique ID of the identity provider.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - The ID of the Tag.
* `external` - The boolean attribute that indicates whether the tag is external or not.

## Import

You can import the tag using any of these accepted formats:

```sh
$ terraform import britive_tag.new tags/{{tag_name}}
$ terraform import britive_tag.new {{tag_name}}
```
