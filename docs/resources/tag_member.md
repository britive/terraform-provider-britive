---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_tag_member Resource - britive"
description: |-
  Manages tag members for the Britive provider.
---

# britive_tag_member Resource

This resource allows you to add or remove a member to the Britive tag.

## Example Usage

```hcl
resource "britive_tag_member" "new" {
    tag_id = britive_tag.new.id
    username = "MyMemberUserName"
    # Optional but recommended to avoid username->ID lookup API call:
    # user_id = data.britive_user.member.user_id
}
```

## Argument Reference

The following arguments are supported:

* `tag_id` - (Required, ForceNew) The identifier of the Britive tag.

* `username` - (Required, ForceNew) The username of the user added to the Britive tag.

* `user_id` - (Optional, ForceNew) The identifier of the user added to the Britive tag. If omitted, the provider resolves it from `username`.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with format `tags/{{tagID}}/users/{{userID}}`

## Import

You can import the Britive tag member using any of these accepted formats:

```sh
terraform import britive_tag_member.new tags/{{tag_id}}/users/{{user_id}}
terraform import britive_tag_member.new tag-name/{{tag_name}}/username/{{username}}
terraform import britive_tag_member.new {{tag_name}}/{{username}}
```
