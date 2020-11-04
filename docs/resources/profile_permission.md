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
    permission {
      name = "Application Developer"
      type = "role"
    }
}
```

## Argument Reference

The following argument is supported:

* `profile_id` (Required): The identifier of the profile.

  For example: `britive_profile.new.id`

* `permission` (Required): The permission that should be added to the profile. 

  The format of the permission is given below.


### `permission` block supports

* `name` - The name of permission.

* `type` - The type of permission.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profileID}}/permissions/{{permission_name}}/type/{{permission_type}}`
