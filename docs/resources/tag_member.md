# britive_tag_member Resource

Adds a user to the Britive tag.

This resource allows you to add or remove Tag member.

**Note:** A tag represents a group of users in the Britive system.

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

* `username` - The username of the user adding to the tag.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with format `tags/{{tagID}}/users/{{userID}}`

## Import

You can import the tag member using any of these accepted formats:

```sh
$ terraform import britive_tag_member.new tags/{{tag_name}}/users/{{username}}
$ terraform import britive_tag_member.new {{tag_name}}/{{username}}
```
