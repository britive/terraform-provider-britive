# britive_profile_session_attribute Resource

This resource allows you to add or remove session attributes from a Britive profile.

!> This resource is only supported from AWS Applications.  
If you try to add a session attribute using this resource for other than AWS applications, an error message will be displayed.

## Example Usage

```hcl
resource "britive_profile" "new" {
    # ...
}

resource "britive_profile_session_attribute" "new" {
  profile_id = britive_profile.new.id
  attribute_name = "Date Of Birth"
  mapping_name = "dob"
  transitive = true
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required) The identifier of the profile.

* `attribute_name` - (Required) The name of attribute.

* `mapping_name` - (Optional) The name for attribute mapping. If omitted, camelCase of `attribute_name` value will be used.

* `transitive` - (Optional) The Boolean flag that indicates whether the attribute is transitive or not. The default value is `false`

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profileID}}/session-attributes/{{sessionAttributeID}}`

## Import

You can import a Britive profile using any of these accepted formats:

```sh
terraform import britive_profile_session_attribute.new apps/{{app_name}}/paps/{{profile_name}}/session-attributes/{{attribute_name}}
terraform import britive_profile_session_attribute.new {{app_name}}/{{profile_name}}/{{attribute_name}}
```
