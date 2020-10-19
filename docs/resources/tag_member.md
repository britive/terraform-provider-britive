---
subcategory: "Tag Member"
layout: "Britive"
page_title: "Britive: britive_tag_member"
description: |-
  Add member to the Britive Tag.
---

# britive\_tag\_member

Manage Britive Tag Member.

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

The following argument is supported:

* `tag_id` - (Required) The identifier of the tag.

* `username` - The username of the member to add to tag.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - an identifier for the resource with format `user-tag/{{tagID}}/users/{{userID}}`
