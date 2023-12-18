# britive_profile_permission Resource

This resource allows you to add or remove permissions from a Britive profile.

## Example Usage

```hcl
resource "britive_profile_permission" "new" {
    profile_id = britive_profile.new.id
    permission_name = "Application Developer"
    permission_type = "role"
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required, Forces new resource) The identifier of the profile.

* `permission_name` - (Required, Forces new resource) The name of permission.

* `permission_type` - (Required, Forces new resource) The type of permission. The value is case-sensitive and must be updated by getting the same from the API response for an import. (https://docs.britive.com/docs/manage-profile-permissions)

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profileID}}/permissions/{{permission_name}}/type/{{permission_type}}`

## Import

You can import a Britive profile using any of these accepted formats:

```sh
terraform import britive_profile_permission.new apps/{{app_name}}/paps/{{profile_name}}/permissions/{{permission_name}}/type/{{permission_type}}
terraform import britive_profile_permission.new {{app_name}}/{{profile_name}}/{{permission_name}}/{{permission_type}}
```
