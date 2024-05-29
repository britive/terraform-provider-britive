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
```

## Argument Reference

The following argument is supported:

* `name` - (Required) The name of the application.

## Attribute Reference

In addition to the above argument, the following attribute is exported:

* `id` - An identifier for the application.
* `environment_ids` - A list of environment ids for the application.
* `environment_group_ids` - A list of environment group ids for the application.
