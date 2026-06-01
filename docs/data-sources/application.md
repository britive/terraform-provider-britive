---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_application Data Source - britive"
description: |-
  Retrieves information of application.
---

# britive_application Data Source

Use this data source to retrieve information about the application.

-> An application is any IAAS/SAAS application integrated with Britive. For example, AWS.

## Example Usage

```hcl
data "britive_application" "my_app" {
    name = "My Application"
}

data "britive_application" "my_app_by_id" {
    app_container_id = "g9f2s7h1example"
}

output "britive_application_my_app" {
    value = data.britive_application.my_app.id
}

output "britive_application_my_app_env_ids" {
    value = data.britive_application.my_app.environment_ids
}

output "britive_application_my_app_env_group_ids" {
    value = data.britive_application.my_app.environment_group_ids
}

output "britive_application_my_app_env_ids_names" {
    value = data.britive_application.my_app.environment_ids_names
}

output "britive_application_my_app_env_group_ids_names" {
    value = data.britive_application.my_app.environment_group_ids_names
}
```

## Argument Reference

Exactly one of the following arguments must be provided:

* `name` - (Optional) The name of the application.

* `app_container_id` - (Optional) The unique identifier of the application.

## Attribute Reference

In addition to the above argument, the following attributes are exported:

* `id` - An identifier for the application.

* `app_container_id` - The unique identifier for the application (same as `id`).

* `environment_ids` - A set of environment ids for the application.

* `environment_group_ids` - A set of environment group ids for the application.

* `environment_ids_names` - A set of environment ids and their respective names for the application.

* `environment_group_ids_names` - A set of environment group ids and and their respective names for the application.
