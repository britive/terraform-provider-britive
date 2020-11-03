| subcategory  | layout    | page_title                    | description                                            |
| ------------ | --------- | ----------------------------- | ------------------------------------------------------ |
| Profile Permission   | Britive   | Britive: britive_profile_permission   | The Britive Profile Permission adds a permission to the Britive profile. |

# britive\_profile\_permission

Adds a permission to the Britive profile.

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

* `profile_id` - (Required): The identifier of the profile.

  For example: `britive_profile.new.id`

* `permission` - (Required): The permission to add to the profile. 

  The format is documented below.


### `permission` block supports

* `name` - The name of permission.

* `type` - The type of permission.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with format `paps/{{profileID}}/permissions/{{permission_name}}/type/{{permission_type}}`

## Import

Profile tag can be imported using any of these accepted formats:

```
$ terraform import britive_profile_permission.new paps/{{profile_name}}/permissions/{{permission_name}}/type/{{permission_type}}
$ terraform import britive_profile_permission.new {{profile_name}}/{{permission_name}}/{{permission_type}}
```