---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_profile Resource - britive"
description: |-
  Manages resource manager profiles for the Britive provider.
---

# britive_resource_manager_profile Resource

The `britive_resource_manager_profile` resource allows you to create, update, and manage resource manager profiles in Britive.

## Example Usage

```hcl
resource "britive_resource_manager_profile" "example" {
    name                 = "example_profile"
    description          = "Profile for managing production resources"
    expiration_duration  = 3600000
    delegation_enabled   = true

    associations {
        label_key   = "environment"
        values = ["Production", "Development"]
    }
    associations {
        label_key   = "region"
        values = ["us-east-1", "eu-west-1"]
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the britive resource manager profile.
* `description` - (Optional) Description of britive resource manager profile.
* `expiration_duration` - (Required) Expiration duration of resource manager profile (in minutes).
* `delegation_enabled` - (Optional) Allow impersonation for profile.
* `associations` - (Optional) List of resource label associations. Each association block supports:
  * `label_key` - (Required) Resource label name for association.
  * `values` - (Required) List of values for the associated resource label. Must contain at least one value.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `status` - Status of resource manager profile.
* `resource_label_color_map` - List of resource label color mappings. Each mapping block contains:
  * `label_key` - Name of the resource label.
  * `color_code` - Color code of the resource label.

## Import

Resource manager profiles can be imported using their unique identifier:

```sh
terraform import britive_resource_manager_profile.example resource-manager/profile/{{profile_id}}
terraform import britive_resource_manager_profile.example resource-manager/profile/abc123def456
```
