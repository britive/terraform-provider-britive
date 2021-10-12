# britive_identity_provider Data Source 

Use this data source to retrieve the identity provider information.

## Example Usage

```hcl
data "britive_identity_provider" "idp" {
    name = "My Identity Provider"
}

resource "britive_tag" "new" {
    # ...
    identity_provider_id = data.britive_identity_provider.idp.id
}
```

## Argument Reference

The following argument is supported:

* `name` - (Required) The name of the identity provider.

## Attribute Reference

In addition to the above argument, the following attribute is exported:

* `id` - An identifier for the identity provider.
