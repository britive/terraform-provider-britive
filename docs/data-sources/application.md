# britive_application Data Source

Use this data source to retrieve information about the application.

-> An application is any IAAS/SAAS application integrated with Britive. For example, AWS.

## Example Usage

```hcl
data "britive_application" "my_app" {
    name = "My Application"
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

The following argument is supported:

* `name` - (Required) The name of the application.

## Attribute Reference

In addition to the above argument, the following attributes are exported:

* `id` - An identifier for the application.

* `environment_ids` - A set of environment ids for the application.

* `environment_group_ids` - A set of environment group ids for the application.

* `environment_ids_names` - A set of environment ids and their respective names for the application.

* `environment_group_ids_names` - A set of environment group ids and and their respective names for the application.
