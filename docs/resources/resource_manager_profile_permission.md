---
subcategory: "Resource Manager"
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
    # A variable with prompt_at_checkout = true never has "value" set here -
    # see the note on prompt_at_checkout below. This applies to any variable
    # type, not only password-type variables.
    variables {
        name               = "accessLevel"
        is_system_defined  = false
        regex_pattern      = "^[a-z]+$"
        description        = "Access level requested for the resource"
        prompt_at_checkout = true
    }
    # Password-type variable: prompt_at_checkout must be true for these.
    variables {
        name               = "dbPassword"
        is_system_defined  = false
        regex_pattern      = "^[a-zA-Z0-9]{8,}$"
        description        = "Password requested for the resource at checkout"
        prompt_at_checkout = true
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
  * `value` - (Optional) Value for the variable. The Britive platform does not allow a value to be set for a variable whose `prompt_at_checkout` is `true` (see note below).
  * `is_system_defined` - (Required) Boolean indicating if the variable is system defined.
  * `regex_pattern` - (Optional) Regex pattern used to validate the value supplied at checkout. The Britive platform only allows this to be set when `prompt_at_checkout` is `true` (see note below).
  * `description` - (Optional) Description shown to the user at checkout. The Britive platform only allows this to be set when `prompt_at_checkout` is `true` (see note below).
  * `prompt_at_checkout` - (Optional) Boolean indicating whether the variable's value is supplied by the user at checkout time instead of being configured here. Defaults to `false` when omitted from config. Setting this to `true` is independent of the variable's type - it is not limited to password-type variables. Must be set to `true` explicitly for a password-type variable (see note below); omitting it for such a variable will cause the apply to fail, since the backend requires `prompt_at_checkout = true` (or `is_system_defined = true`) for password-type variables.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `permission_id` - The ID of the profile permission.
* `description` - Description of the permission.
* `resource_type_id` - ID of the ResourceType associated with this permission.
* `resource_type_name` - Name of the ResourceType associated with this permission.
* `variables.type` - Type of the variable (e.g. `String` or `password`), as defined on the associated resource type permission.

-> To mark a variable as a password type, suffix its name with `:password` when defining `variables` on the associated `britive_resource_manager_resource_type_permission` resource (e.g. `variables = ["test1", "test2:password"]`). The `britive_resource_manager_profile_permission` resource then references the variable by its base name (e.g. `test2`).

-> A password-type variable must have `prompt_at_checkout` set to `true` - the backend enforces this, independent of a variable's type. `prompt_at_checkout` splits a variable's `value`, `regex_pattern`, and `description` into two mutually exclusive modes on the Britive platform: when it is `true`, the value is supplied by the user at checkout instead of in this resource, so the platform does not allow `value` to be set - only `regex_pattern` (to validate what's entered at checkout) and `description` (shown to the user at checkout) are allowed. When it is `false`, the opposite holds: `value` is configured directly here, and the platform does not allow `regex_pattern` or `description` to be set, since there is no checkout-time prompt for them to apply to. This provider does not validate these combinations locally - each variable block is sent to the platform as configured, and the platform is responsible for accepting or rejecting it.

-> As the maximum payload size is limited to 8 KB, the number of variables as well as the size of their values must collectively remain within this limit.

## Import

Profile permissions can be imported using their unique identifier:

```sh
terraform import britive_resource_manager_profile_permission.example resource-manager/profile/{{profile_id}}/permission/{{permission_id}}
terraform import britive_resource_manager_profile_permission.example resource-manager/profile/abc123def456/permission/xy123zjash7wg12w
```

