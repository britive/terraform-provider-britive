---
subcategory: "resource_manager"
layout: "britive"
page_title: "britive_resource_manager_profile_permissions Data Source - britive"
description: |-
  Retrieves information of available permissions.
---

# britive_resource_manager_profile_permissions Data Source

This data source enables you to retrieve the permissions available for a specific profile.

## Example Usage

```hcl
data "britive_resource_manager_profile_permissions" "new_permission" {
  profile_id = "96sdysds-y8w7eyh-dwdyaus-9as7ydy"
}

output "permissions" {
  value = data.britive_resource_manager_profile_permissions.new_permission.permissions
}
```

## Argument Reference

The following argument is supported:

- `profile_id` (Required) – The unique identifier of the profile.

## Attribute Reference

The following attributes are exported:

- `permissions` – A list of permissions available for the specified profile.
    - `name` – The name of the permission.
    - `permission_id` – The unique identifier of the permission.
    - `version` – A list of available versions of the permission.