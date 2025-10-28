---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_profile Resource - britive"
description: |-
  Manages profiles for the Britive provider.
---

# britive_profile Resource

This resource allows you to create and configure a Britive Profile.

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
    delegation_enabled               = true
    associations {
      type  = "Environment"
      value = "QA Subscription"
    },
    destination_url                  = "https://console.aws.amazon.com"
}
```

## Argument Reference

The following arguments are supported:

* `app_container_id` - (Required, ForceNew) The identity of the Britive application.

* `name` - (Required) The name of the Britive profile.

* `description` - (Optional) A description of the Britive profile.

* `disabled` - (Optional) The status of the Britive profile. By default, the Britive profile is enabled. To disable a Britive profile, set `disabled = true`.

* `expiration_duration` - (Required) The expiration time for the Britive profile. For example, `25m0s`

* `destination_url` - (Optional) The console URL where the user will be redirected upon checking out the profile. For example: `https://console.aws.amazon.com`

* `associations` - (Required) The list of associations for the Britive profile.

The following arguments are supported, except for AWS profiles:

* `extendable` - (Optional) The Boolean flag that indicates whether profile expiry is extendable or not. Default: `false`.

* `notification_prior_to_expiration` - (Optional) The Britive profile expiry notification as a time value. For example, `10m0s`

* `extension_duration` - (Optional) The Britive profile expiry extension duration as a time value. For example: `12m30s`

* `delegation_enabled` - (Optional) Allow impersonation for profile.

* `extension_limit` - (Optional) The Britive profile expiry extension limit. For example: `2`

The format of `associations` is documented below.

### `associations` block supports

* `type` - (Required) The type of association, should be one of [Environment, EnvironmentGroup, ApplicationResource].

* `value` - (Required) The association value. For AWS applications, one of the following should be used: EnvironmentID, EnvironmentName, or AccountID.

* `parent_name` - (Optional) The parent name of the resource. Required only if the association type is ApplicationResource.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - The identity of the Britive profile.

## Import

You can import a profile using any of these accepted formats:

```sh
terraform import britive_profile.new apps/{{app_name}}/paps/{{name}}
terraform import britive_profile.new {{app_name}}/{{name}}
```
