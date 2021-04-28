# britive_tag_member Resource

This resource allows you to add or remove a member to the Britive tag.

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

* `tag_id` - (Required, Forces new resource) The identifier of the Britive tag.

* `username` - (Required, Forces new resource) The username of the user added to the Britive tag.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with format `tags/{{tagID}}/users/{{userID}}`

## Import

You can import the Britive tag member using any of these accepted formats:

```sh
terraform import britive_tag_member.new tags/{{tag_name}}/users/{{username}}
terraform import britive_tag_member.new {{tag_name}}/{{username}}
```
