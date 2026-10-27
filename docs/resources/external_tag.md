---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_external_tag Resource - britive"
description: |-
  Manages an external tag: a tag on a non-default identity provider that the identity provider connects through SCIM.
---

# britive_external_tag Resource

This resource manages an external tag. An external tag is a tag that belongs to an identity provider other than Britive's default provider (for example Okta, Entra ID or OneLogin) and whose membership is provisioned by that identity provider through SCIM.

The tag goes through two states, and this resource behaves differently in each:

| State | How it gets there | What Terraform manages |
| --- | --- | --- |
| **Placeholder** (`external = false`) | Created by this resource, or in the Britive UI, on a non-default identity provider. | Everything: `name`, `description`, `requestable`, `attributes`, members (via `britive_tag_member`) and deletion. |
| **Connected** (`external = true`) | The identity provider provisions a group with the same name through SCIM and Britive links it to the tag. | `requestable` and `attributes`. `description` only when the identity provider's **Allow tag name and description to be editable** setting is on. `name`, status, members and deletion belong to the identity provider. |

Creating the placeholder from Terraform lets you declare the tag, its policies and its owners in one apply, before the group exists in the identity provider. The tag's `id` does not change when the identity provider connects it, so references from policies and `britive_tag_owner` keep working.

-> Tags on Britive's default identity provider are managed with the [`britive_tag`](tag.md) resource. This resource refuses the default provider, and `britive_tag` refuses every other provider.

## Example Usage

```hcl
data "britive_identity_provider" "okta" {
    name = "Okta"
}

resource "britive_external_tag" "platform_admins" {
    name                 = "Platform-Admins"           # must match the group name in Okta
    description          = "Platform admins, provisioned from Okta"
    identity_provider_id = data.britive_identity_provider.okta.id
    requestable          = true

    attributes {
        attribute_name  = "Team"
        attribute_value = "Platform"
    }
}

resource "britive_tag_owner" "platform_admins" {
    tag_id = britive_external_tag.platform_admins.id

    user {
        username = "alice"
    }
}

resource "britive_policy" "platform_admins" {
    name        = "Platform-Admins"
    description = "Granted to the Okta group once it is connected"
    access_type = "Allow"
    members     = jsonencode({
        tags = [{ id = britive_external_tag.platform_admins.id }]
    })
    permissions = jsonencode([{ id = britive_permission.platform.id }])
    roles       = jsonencode([])
}
```

### Adopting a tag the identity provider created first

When the group was pushed from the identity provider before Terraform ran, the tag already exists in Britive and a plan for a new resource with that name fails with **External Tag Already Exists**, naming the tag id. Terraform does not take over existing tags on create. Adopt it with an import block (Terraform 1.5+):

```hcl
import {
  to = britive_external_tag.platform_admins
  id = "2utfgbi5320i2lnyhnmt"
}

resource "britive_external_tag" "platform_admins" {
    name                 = "Platform-Admins"
    identity_provider_id = data.britive_identity_provider.okta.id
    requestable          = true
}
```

or with the CLI:

```sh
terraform import britive_external_tag.platform_admins 2utfgbi5320i2lnyhnmt
```

After the import the resource behaves exactly as a connected tag created from Terraform.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the tag. It must match the group name the identity provider will provision. Once the tag is connected, changing it is refused at plan time; rename the group in the identity provider instead.

* `identity_provider_id` - (Required, Forces new resource) The unique identity of the non-default identity provider that owns the tag. Using Britive's default identity provider is refused at plan time; use `britive_tag` for those tags.

* `description` - (Optional) The description of the tag. Once the tag is connected, a change is refused at plan time unless the identity provider's **Allow tag name and description to be editable** setting (SCIM tab) is on.

* `requestable` - (Optional, Computed) Whether the tag is requestable. Writable in both states. When omitted, the value is determined by the Britive backend and stored in state.

* `attributes` - (Optional) One or more attribute blocks to associate with the tag. Writable in both states. Multiple blocks with the same `attribute_name` are supported for multi-valued attributes. Each block supports:
  * `attribute_name` - (Required) The name of the attribute.
  * `attribute_value` - (Required) The value of the attribute.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - The identity of the tag.
* `external` - `true` once the identity provider has connected the tag through SCIM, `false` while it is a placeholder.
* `external_id` - The identity provider's identifier for the connected group, when the provider supplies one. Empty for placeholders and for SCIM providers that do not send an id.

## Lifecycle notes

* **Members.** You can add members to a placeholder with `britive_tag_member`, but they do not survive the connection: when the identity provider pushes the group, SCIM replaces the tag's membership with the group's, and from then on the Britive API refuses membership changes (`UT-0004` on add, `UT-0006` on remove). A `britive_tag_member` left in the configuration is reported as replaced on the next refresh and must be removed from the configuration. Pre-wire policies and owners on a placeholder, not members.
* **Status.** This resource does not expose `disabled`. The Britive API refuses enabling or disabling an external tag.
* **Destroy of a placeholder** deletes the tag.
* **Destroy of a connected tag** cannot delete it: the Britive API refuses (`UT-0009`). Terraform removes the resource from state and prints a warning; the tag stays in Britive until the group is deleted in the identity provider. Nothing is recreated afterwards: if the resource is still in the configuration, the next plan fails with **External Tag Already Exists** and offers either to import the tag again or to remove the resource from the configuration. Add `lifecycle { prevent_destroy = true }` if you want the destroy itself to stop instead.
* **Deleted by the identity provider.** When the identity provider deletes a connected group, Britive deletes the tag. The next refresh removes the resource from state with a warning, and the following plan proposes recreating it as a placeholder. Remove the resource from the configuration if that is not what you want.
* **Same name on several providers.** Tag names are unique per identity provider only. The API refuses a duplicate name on the same provider (`UT-0001`) but allows it on another. When a policy refers to such a tag by name, the backend picks one; refer to tags by `id` in policy members (with roles and permissions also by id, since the API requires one form for the whole body).

## Import

You can import an external tag by id, or by name when the name is unique across identity providers:

```sh
terraform import britive_external_tag.new {{tag_id}}
terraform import britive_external_tag.new {{tag_name}}
```

Importing a tag that belongs to Britive's default identity provider is refused; use `britive_tag` for it.
