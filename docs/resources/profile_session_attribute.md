# britive_profile_session_attribute Resource

This resource allows you to add or remove session attributes from a Britive profile.

!> This resource is only supported for AWS Applications.  
If you try to add a session attribute using this resource for other than AWS applications, an error message will be displayed.

## Example Usage

```hcl
# Static Attribute
resource "britive_profile_session_attribute" "static_new" {
  profile_id = britive_profile.new.id
  attribute_type = "Static"
  attribute_value = "IT"
  mapping_name = "department"
  transitive = false
}

# User Attribute
resource "britive_profile_session_attribute" "user_new" {
  profile_id = britive_profile.new.id
  attribute_type = "Identity"  
  attribute_name = "Date Of Birth"
  mapping_name = "dob"
  transitive = false
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required, ForceNew) The identifier of the profile.

* `attribute_type` - (Optional, ForceNew) The type of attribute, should be one of [Static, Identity]. Default: `"Identity"`.

* `attribute_name` - (Optional, Required when `attribute_type` is Identity, ForceNew) The name of attribute.

* `attribute_value` - (Optional, Required when `attribute_type` is Static) The value of attribute.

* `mapping_name` - (Required) The name for attribute mapping.

* `transitive` - (Optional) The Boolean flag that indicates whether the attribute is transitive or not. Default: `false`.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profileID}}/session-attributes/{{sessionAttributeID}}`

## Import

You can import a Britive profile using any of these accepted formats:

```sh
terraform import britive_profile_session_attribute.new apps/{{app_name}}/paps/{{profile_name}}/session-attributes/type/{{attribute_type}}/mapping-name/{{mapping_name}}
terraform import britive_profile_session_attribute.new {{app_name}}/{{profile_name}}/{{attribute_type}}/{{mapping_name}}
```
