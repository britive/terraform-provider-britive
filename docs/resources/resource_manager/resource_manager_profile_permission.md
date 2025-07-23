---
subcategory: "resource_manager"
layout: "britive"
page_title: "britive_resource_manager_profile_permission Resource - britive"
description: |-
  Manages resource manager profile permissions for the Britive provider.
---

# britive_resource_manager_profile_permission Resource

The `britive_resource_manager_profile_permission` resource allows you to create, update, and manage permissions associated with a resource manager profile in Britive.

## Example Usage

```hcl
resource "britive_resource_manager_profile_permission" "example" {
    profile_id   = "abc123def456"
    name         = "PermissionName"
    version      = "5"

    variables {
        name              = "resourceId"
        value             = "prod-001"
        is_system_defined = false
    }
    variables {
        name              = "accessLevel"
        value             = "read"
        is_system_defined = false
    }
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required) The ID of the resource manager profile.
* `name` - (Required) Name of the permission to associate with the profile.
* `version` - (Required) Version of the permission. You can specify any version number (e.g., `"1"`), `"latest"` for the most recent version, or `"local"` for the local version.
* `variables` - (Optional) List of variables for the permission. Each variable block supports:
  * `name` - (Required) Name of the variable.
  * `value` - (Required) Value for the variable.
  * `is_system_defined` - (Required) Boolean indicating if the variable is system defined.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `permission_id` - The ID of the profile permission.
* `description` - Description of the permission.
* `resource_type_id` - ID of the ResourceType associated with this permission.
* `resource_type_name` - Name of the ResourceType associated with this permission.

-> As the maximum payload size is limited to 8 KB, the number of variables as well as the size of their values must collectively remain within this limit.

## Import

Profile permissions can be imported using their unique identifier:

```sh
terraform import britive_resource_manager_profile_permission.example resource-manager/profile/{{profile_id}}/permission/{{permission_id}}
terraform import britive_resource_manager_profile_permission.example resource-manager/profile/abc123def456/permission/xy123zjash7wg12w
```

