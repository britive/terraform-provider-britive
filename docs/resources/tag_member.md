| subcategory  | layout    | page_title                    | description                                            |
| ------------ | --------- | ----------------------------- | ------------------------------------------------------ |
| Tag Member   | Britive   | Britive: britive_tag_member   | The Britive Tag Member adds a user to the Britive tag. |

# britive\_tag\_member

Adds a user to the Britive tag.

**Note:** A user tag represents a group of users in the Britive system.

## Example Usage

```hcl
resource "britive_tag" "new" {
    # ...
}

resource "britive_tag_member" "new" {
    tag_id = britive_tag.new.id
    username = "MyMemberUserName"
}
```

## Argument Reference

The following arguments are supported:

* `tag_id` - (Required): The identifier of the tag.

  For example: `britive_tag.new.id`

* `username` - The username of the user who is added to the tag.

  For example: `NewUserOne`

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with format `user-tag/{{tagID}}/users/{{userID}}`

## Import

You can import the tag member using any of these accepted formats:

```
$ terraform import britive_tag_member.new tags/{{tag_name}}/users/{{username}}
$ terraform import britive_tag_member.new {{tag_name}}/{{username}}
```