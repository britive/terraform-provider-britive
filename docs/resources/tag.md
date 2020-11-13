| subcategory | layout    | page_title             | description                             |
| ----------- | --------- | ---------------------- | --------------------------------------- |
| Tag         |  Britive  | Britive: britive_tag   | The Britive Tag creates a new user tag. |

# britive\_tag

Creates a new user tag.

**Note:** A user tag represents a group of users in the Britive system.

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

  For example: `My Tag`

* `description` - (Optional): A description of the tag.

  For example: `My Tag Description`

* `disabled` - (Optional): Default tag is enabled. To disable tag set `disabled = true`.

* `user_tag_identity_providers` - (Required): The list of identity providers associated with the tag. 

  The format is documented below.

### `user_tag_identity_providers` block supports

* `identity_provider` - An identity provider. 

  The format is documented below.

### `identity_provider` block supports

* `id` - The unique ID of the identity-provider.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - An identifier for the resource.
* `external` - The boolean attribute that indicates whether the tag is external or not.

## Import

You can import the tag using any of these accepted formats:

```
$ terraform import britive_tag.new tags/{{tag_name}}
$ terraform import britive_tag.new {{tag_name}}
```