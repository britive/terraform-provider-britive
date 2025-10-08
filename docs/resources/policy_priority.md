---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_policy_priority Resource - britive"
description: |-
  Manages policy priority configuration for a Britive profile.
---

# britive_policy_priority Resource

This resource allows you to manage policy prioritization of policies for a Britive profile.

## Example Usage

```hcl
resource "britive_policy_priority" "new_priority" {
  profile_id = "abc123xyz"
  policy_priority {
    id       = "policy-001"
    priority = 0
  }
  policy_priority {
      id       = "policy-002"
      priority = 1
  }
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required) The identity of britive application profile.
* `policy_priority` - (Optional) The policy priority
    * `id` - (Required) The identity of britive profile policy.
    * `priority` - (Required) The priority order (integer), where 0 is the highest priority.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - The identity of the Britive policy priority.

-> When this resource is created, Terraform will automatically **enable policy prioritization** for the associated profile and prioritize policies accordingly. Conversely, when the resource is deleted, Terraform will **disable policy prioritization** for that profile.

## Import

You can import a profile using any of these accepted formats:

```sh
terraform import britive_policy_priority.new paps/{{profile_id}}/policies/priority
terraform import britive_policy_priority.new {{profile_id}}
```

-> - During import, all current policies and their priorities are **read and synced** into the Terraform state.