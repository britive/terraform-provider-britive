# britive_application Data Source

Gets information about the application.

**Note:** An application is any IAAS/SAAS application integrated with Britive. For example, AWS.

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

  For example, `Britive`

## Attributes Reference

In addition to the above arguments , the following attributes are exported:

* `id` - an identifier for the data source. 

