# britive_profile_identity Resource

Adds an identity to a Britive profile.

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

* `profile_id` (Required): The identifier of the profile.

  For example: `britive_profile.new.id`

* `username` (Required): The name of the identity.

  For example: `My Tag`

* `access_period` (Optional): The access period of the identity in a profile. 

  The format of an access_period is documented below.


### `access_period` block supports

* `start` - The start of the access period for the associated identity.

* `end` - The end of the access period for the associated identity.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profileID}}/users/{{userID}}`

## Import

You can import a profile using any of these accepted formats:

```
$ terraform import britive_profile.new apps/{{app_name}}/paps/{{profile_name}}/users/{{username}}
$ terraform import britive_profile.new {{app_name}}/{{profile_name}}/{{username}}
```