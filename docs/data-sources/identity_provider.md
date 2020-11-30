# britive_identity_provider Data Source

Gets information about the identity provider.

## Example Usage

```hcl
data "britive_identity_provider" "idp" {
    name = "My Identity Provider"
}
resource "britive_tag" "new" {
    # ...

    user_tag_identity_providers {
        identity_provider {
            id = data.britive_identity_provider.idp.id
        }
    }
}
```

## Argument Reference

The following argument is supported:

* `name` - (Required): The name of the identity provider.

  For example, `Britive`

## Attributes Reference

In addition to the above arguments, the following attribute is exported:

* `id` - an identifier for the data source. 