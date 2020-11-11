| subcategory | layout    | page_title             | description                             |
| ----------- | --------- | ---------------------- | --------------------------------------- |
| Profile         |  Britive  | Britive: britive_profile   | The Britive Profile creates a new user profile. |

# britive\_profile

Creates a new user profile.

## Example Usage

```hcl
data "britive_application" "app" {
    name = "My Application"
}

resource "britive_profile" "new" {
    app_container_id                 = data.britive_application.app.id
    name                             = "My Profile"
    description                      = "My Profile Description"
    expiration_duration              = "25m0s"
    extendable                       = true
    notification_prior_to_expiration = "10m0s"
    extension_duration               = "12m30s"
    extension_limit                  = 2
    associations {
      type  = "Environment"
      value = "QA Subscription"
    }
}
```

## Argument Reference

The following arguments are supported:

* `app_container_id` (Required): The id of the Britive application.

  For example: `bCBV60TkOopdu5JEE` or `data.britive_application.app.id`

* `name` (Required): The name of the profile.

  For example: `My Profile`

* `description` (Optional): A description of the profile.

  For example: `My Profile Description`

* `disabled` - (Optional): Default profile is enabled. To disable profile set `disabled = true`.

* `expiration_duration` (Required): The expiration time for the profile.

  For example: `25m0s`


* `extendable` (Optional): The Boolean flag that indicates whether profile expiry is extendable or not. The default value is `false`.

  For example: `true`

* `notification_prior_to_expiration`  (Optional): The profile expiry notification as a time value.

  For example: `10m0s`


* `extension_duration` - (Optional): The profile expiry extension as a time value.

  For example: `12m30s`


* `extension_limit` - (Optional): The repetition limit for extending the profile expiry.

  For example: `2`

* `associations` - (Required): The list of associations for the profile. 

  The format of an association is documented below.


### `associations` block supports

* `type` - The type of association, either Environment or Environment Group.

* `value` - The association value.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier for the resource.

## Import

You can import a profile using any of these accepted formats:

```
$ terraform import britive_profile.new apps/{{app_container_id}}/paps/{{profile_id}}
$ terraform import britive_profile.new {{app_container_id}}/{{profile_id}}
```