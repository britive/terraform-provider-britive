| subcategory  | layout    | page_title                    | description                                            |
| ------------ | --------- | ----------------------------- | ------------------------------------------------------ |
| Profile Identity   | Britive   | Britive: britive_profile_identity   | The Britive Profile Identity adds a identity to the Britive profile. |

# britive\_profile\_identity

Adds a identity to the Britive profile.

## Example Usage

```hcl
resource "britive_profile" "new" {
    # ...
}

resource "britive_profile_identity" "new" {
    profile_id = britive_profile.new.id
    username = "MyUser"
    access_period {
        start = "2020-11-01T06:00:00Z"
        end   = "2020-11-05T06:00:00Z"
    }
}
```

## Argument Reference

The following argument is supported:

* `profile_id` - (Required): The identifier of the profile.

  For example: `britive_profile.new.id`

* `username` - (Required): The name of the identity.

  For example: `My Tag`

* `access_period` - (Optional): The access period of identity in the profile. 

  The format is documented below.


### `access_period` block supports

* `start` - The start of the access period for the associated identity.

* `end` - The end of the access period for the associated identity.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with format `paps/{{profileID}}/users/{{userID}}`

## Import

Profile identity can be imported using any of these accepted formats:

```
$ terraform import britive_profile_identity.new paps/{{profile_name}}/users/{{username}}
$ terraform import britive_profile_identity.new {{profile_name}}/{{username}}
```