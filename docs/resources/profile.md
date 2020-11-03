| subcategory | layout    | page_title             | description                             |
| ----------- | --------- | ---------------------- | --------------------------------------- |
| Profile         |  Britive  | Britive: britive_profile   | The Britive Profile creates a new profile. |

# britive\_profile

Creates a new profile.

## Example Usage

```hcl
data "britive_application" "app" {
    name = "My Application"
}

resource "britive_profile" "new" {
    app_container_id                 = data.britive_application.app.id
    name                             = "My Profile"
    description                      = "My Profile Description"
    status                           = "active"
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

* `app_container_id` - (Required): The id of the Britive application.

  For example: `bCBV60TkOopdu5JEE` or `data.britive_application.app.id`

* `name` - (Required): The name of profile.

  For example: `My Profile`

* `description` - (Optional): A description of the profile.

  For example: `My Profile Description`

* `status` - (Required): The status of the profile.

  For example: `Active`

* `expiration_duration` - (Required): The expiration duration of the profile.

  For example: `25m0s`


* `extendable` - (Optional): Boolean flag whether profile expiration is extendable or not. Default to `false`.

  For example: `false`

* `notification_prior_to_expiration` - (Optional): The profile expiration notification as duration.

  For example: `10m0s`


* `extension_duration` - (Optional): The profile expiration extenstion as duration.

  For example: `12m30s`


* `extension_limit` - (Optional): The profile expiration extension repeat limit

  For example: `2`

* `associations` - (Required): The list of associations for the profile. 

  The format is documented below.


### `associations` block supports

* `type` - Type of association, either Environment or EnvironmentGroup.

* `value` - Association value.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - An identifier for the resource.

## Import

Profile can be imported using any of these accepted formats:

```
$ terraform import britive_profile.new user-profiles/{{profile_name}}
$ terraform import britive_profile.new {{profile_name}}
```