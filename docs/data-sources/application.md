# britive_application Data Source

Use this data source to retrieve information about the application.

-> An application is any IAAS/SAAS application integrated with Britive. For example, AWS.

## Example Usage

```hcl
data "britive_application" "my_app" {
    name = "My Application"
}

out "britive_application_my_app" {
    value = data.britive_application.my_app.id
}
```

## Argument Reference

The following argument is supported:

* `name` - (Required): The name of the application.

## Attribute Reference

In addition to the above argument, the following attribute is exported:

* `id` - An identifier for the application.
