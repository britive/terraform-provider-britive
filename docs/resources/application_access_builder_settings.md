---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_application_access_builder_settings Resource - britive"
description: |-
  Manages access builder settings and association approvers for an application for the Britive provider.
---

# britive_application_access_builder_settings Resource

The `britive_application_access_builder_settings` resource allows you to manage access requests
for a Britive application: whether users can request access, the timeouts, notification
mediums, manager approval, member rules, and the association approvers that map environments
to [`britive_application_approvers_group`](application_approvers_group.md) resources.

An application has one access builder settings record, so use one instance of this resource
per application. Only `application_id` is required. Everything else is optional and left
unmanaged when omitted, so you can, for example, manage only notification mediums or only
association approvers.

When `association_approver` blocks are configured, the resource manages every association
approver of the application, and orders the API calls itself: approvers are created before
access requests are enabled, and removed before new ones are created, so an environment can
move from one approver to another in a single apply.

~> Enabling access requests requires an approval timeout, a profile expiration timeout, at least
one association approver, and at least one notification medium. These can be configured here or
already exist on the application.

-> An environment or environment group can belong to only one association approver, including
parent and child environment groups. To add another approvers group to a covered environment,
add it to the `approvers_group_ids` of the association approver that already covers it.

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
}

resource "britive_application_access_builder_settings" "this" {
  application_id             = britive_application.example.id
  allow_access_request       = true
  approval_timeout           = "00:01:00"
  profile_expiration_timeout = 15

  notification_medium {
    name = "Email" # or: id = "<notification_medium_id>"
  }

  manager_approval_required  = true
  manager_approval_condition = "Any"

  association_approver {
    name = "production-approvers"

    association {
      type  = "Environment"
      value = "Production" # or: value = "<environment_id>"
    }

    approvers_group_ids = [
      britive_application_approvers_group.security.group_id
    ]
  }

  member_rule {
    type      = "Tag"
    name      = "Engineering" # or: id = "<tag_id>"
    condition = "Include"
  }
}
```

### Manage only notification mediums

```hcl
resource "britive_application_access_builder_settings" "notifications" {
  application_id = britive_application.example.id

  notification_medium {
    name = "Email"
  }
}
```

## Argument Reference

Only `application_id` is required. An optional argument or block that is omitted is left
unmanaged: the provider does not send it, and keeps whatever value the application already has.

* `application_id` - (Required) The ID of the application. Forces replacement if changed.
* `allow_access_request` - (Optional) Whether users can request access.
* `approval_timeout` - (Optional) How long a request waits for approval, in `DD:HH:MM` format. Must be from `00:00:05` (5 minutes) to `05:00:00` (5 days).
* `profile_expiration_timeout` - (Optional) How long approved access lasts, in minutes. Must be at least `1`; the valid range is enforced by the API.
* `manager_approval_required` - (Optional) Whether the requester's manager must approve.
* `manager_approval_condition` - (Optional) Which manager approval is required. One of `Any`, `All`, `Manager`.
* `notification_medium` - (Optional) A notification medium for access requests. Removing every `notification_medium` block sends an empty list to clear them; the API rejects this while access requests are enabled. Each block supports:
  * `id` - (Optional) The ID of the notification medium. Exactly one of `id` or `name` must be set.
  * `name` - (Optional) The name of the notification medium, for example `Email`. Exactly one of `id` or `name` must be set.
* `member_rule` - (Optional) A user or tag included in or excluded from access requests. Removing every `member_rule` block clears the member rules. Each block supports:
  * `type` - (Required) The member type. One of `User`, `Tag`.
  * `condition` - (Required) One of `Include`, `Exclude`.
  * `id` - (Optional) The ID of the user or tag. Exactly one of `id` or `name` must be set.
  * `name` - (Optional) The username of the user, or the name of the tag. Exactly one of `id` or `name` must be set.
* `association_approver` - (Optional) Maps environments or environment groups to approvers groups. Removing every `association_approver` block removes all association approvers. Each block supports:
  * `name` - (Required) The name of the association approver. Must be unique within the application.
  * `association` - (Required) An environment or environment group. At least one is required. Each block supports:
    * `type` - (Required) One of `Environment`, `EnvironmentGroup` (case-insensitive).
    * `value` - (Required) The environment or environment group, by name or ID.
  * `approvers_group_ids` - (Required) The IDs of the approvers groups. At least one is required.

The IDs of notification mediums, users, and tags can also be passed from the
[`britive_notification_medium`](../data-sources/notification_medium.md),
[`britive_user`](../data-sources/user.md), and [`britive_tag`](../data-sources/tag.md) data sources.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the access builder settings.

## Import

Access builder settings can be imported using the application ID or the composite ID. Import
brings in every association approver of the application:

```sh
terraform import britive_application_access_builder_settings.example <application_id>
terraform import britive_application_access_builder_settings.example apps/<application_id>/access-builder-settings
```

## Delete Behavior

The API has no delete operation for the settings record. Destroying this resource resets only
what it manages:

* Association approvers and member rules are removed, if configured.
* Manager approval is reset, if configured.
* Access requests are disabled, if `allow_access_request` is configured or association
  approvers are removed.

The approval timeout, profile expiration timeout, and notification mediums remain on the
application, because the API does not allow them to be cleared.
