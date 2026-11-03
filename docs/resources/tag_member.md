---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_tag_member Resource - britive"
description: |-
  Manages tag members for the Britive provider.
---

# britive_tag_member Resource

This resource allows you to add or remove a member to the Britive tag.

-> **External tags.** A member can be added to an external tag (`britive_external_tag`) only while the tag is an unconnected placeholder, and it does not survive the connection: when the identity provider pushes the group, SCIM replaces the tag's membership with the group's. From then on Britive reserves membership for the identity provider. A plan that adds a member to a connected tag fails with **Cannot Add Member To Connected External Tag**; a member that was replaced by the sync is reported with a warning on refresh and must be removed from the configuration. Manage membership in the identity provider's group instead.

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

In addition to the above arguments, the following attributes are exported.

* `id` - An identifier of the resource with format `tags/{{tagID}}/users/{{userID}}`

* `tag_name` - The name of the Britive tag.

## Import

You can import the Britive tag member using any of these accepted formats:

```sh
terraform import britive_tag_member.new tags/{{tag_id}}/users/{{user_id}}
terraform import britive_tag_member.new tag-name/{{tag_name}}/username/{{username}}
terraform import britive_tag_member.new {{tag_name}}/{{username}}
```
