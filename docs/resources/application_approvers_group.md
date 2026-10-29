---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_application_approvers_group Resource - britive"
description: |-
  Manages an approvers group for application access requests for the Britive provider.
---

# britive_application_approvers_group Resource

The `britive_application_approvers_group` resource allows you to manage an approvers group for
a Britive application. An approvers group is a set of users and tags who approve access
requests. Approvers groups are assigned to environments through the `association_approver`
blocks of [`britive_application_access_builder_settings`](application_access_builder_settings.md).

## Example Usage

```hcl
resource "britive_application_approvers_group" "security" {
  application_id = britive_application.example.id
  name           = "security-approvers"
  condition      = "All"

  member {
    type = "User"
    name = "security.lead" # or: id = "<user_id>"
  }

  member {
    type = "Tag"
    name = "Security" # or: id = "<tag_id>"
  }
}
```

## Argument Reference

* `application_id` - (Required) The ID of the application. Forces replacement if changed.
* `name` - (Required) The name of the approvers group.
* `condition` - (Required) Whether `All` or `Any` of the members must approve.
* `member` - (Required) A user or tag in the approvers group. At least one is required. Each block supports:
  * `type` - (Required) The member type. One of `User`, `Tag`.
  * `id` - (Optional) The ID of the user or tag. Exactly one of `id` or `name` must be set.
  * `name` - (Optional) The username of the user, or the name of the tag. Exactly one of `id` or `name` must be set.

The IDs of users and tags can also be passed from the [`britive_user`](../data-sources/user.md)
and [`britive_tag`](../data-sources/tag.md) data sources.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the approvers group.
* `group_id` - The unique identifier of the approvers group.

## Import

Approvers groups can be imported using their composite ID:

```sh
terraform import britive_application_approvers_group.example apps/<application_id>/approvers-groups/<group_id>
```

## Delete Behavior

An approvers group cannot be deleted while an association approver uses it. Remove the group
from every `association_approver` block of `britive_application_access_builder_settings`
first. When both are managed in the same configuration, Terraform does this automatically,
because the settings resource references the group's ID.
