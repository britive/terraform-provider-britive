| subcategory  | layout    | page_title                    | description                                            |
| ------------ | --------- | ----------------------------- | ------------------------------------------------------ |
| Profile Permission   | Britive   | Britive: britive_profile_permission   | The Britive Profile Permission adds a permission to a Britive profile. |

# britive\_profile\_permission

Adds a permission to a Britive profile.

## Example Usage

```hcl
resource "britive_profile" "new" {
    # ...
}

resource "britive_profile_permission" "new" {
    profile_id = britive_profile.new.id
    permission_name = "Application Developer"
    permission_type = "role"
}
```

## Argument Reference

The following argument is supported:

* `profile_id` (Required): The identifier of the profile.

  For example: `britive_profile.new.id`


* `permission_name` (Required): The name of permission.

* `permission_type` (Required): The type of permission.


## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profileID}}/permissions/{{permission_name}}/type/{{permission_type}}`

## Import

You can import a profile using any of these accepted formats:

```
$ terraform import britive_profile.new apps/{{app_name}}/paps/{{profile_name}}/permissions/{{permission_name}}/type/{{permission_type}}
$ terraform import britive_profile.new {{app_name}}/{{profile_name}}/{{permission_name}}/{{permission_type}}
```