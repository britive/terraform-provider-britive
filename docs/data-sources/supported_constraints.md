---
subcategory: ""
layout: "britive"
page_title: "britive_supported_constraints Data Source - britive"
description: |-
  Retrieves information of supported constraints.
---

# britive_supported_constraints Data Source

Use this data source to retrieve the supported constraint types for a profile permission.

The permission should be associated to the profile, to fetch the supported constraint types.

-> This data source is only supported for GCP and Okta Applications.

## Example Usage

```hcl
data "britive_application" "my_app" {
    name = "My Application"
}

resource "britive_profile" "new" {
    app_container_id                 = data.britive_application.my_app.id
    name                             = "My Profile"
    description                      = "My Profile Description"
    disabled            = false
    expiration_duration = "1h0m0s"
    extendable          = false
    associations {
        type  = "EnvironmentGroup"
        value = "Banking"
    }
}

resource "britive_profile_permission" "new" {
    profile_id = britive_profile.new.id
    permission_name = "BigQuery Data Owner"
    permission_type = "role"
}

data "britive_supported_constraints" "new" {
  profile_id = britive_profile.new.id
  permission_name = britive_profile_permission.new.permission_name
  permission_type = britive_profile_permission.new.permission_type
}

output "britive_supported_constraints_output" {
    value = data.britive_supported_constraints.new.constraint_types
}

data "britive_supported_constraints" "test" {
  profile_id = "h59ih6p1537xxxxxxxxx"
  permission_name = "BigQuery Data Owner"
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required) The identifier of the profile.
* `permission_name` - (Required) Name of the permission associated with the profile.
* `permission_type` - (Optional) The type of permission. Default: `"role"`.

## Attribute Reference

In addition to the above arguments, the following attribute is exported:

* `constraint_types` - A set of constraints supported for the given profile permission
