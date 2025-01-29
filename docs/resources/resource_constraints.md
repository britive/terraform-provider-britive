# britive_constraint Resource

This resource allows you to create and configure a constraint on the permission associated to a profile.

-> This resource is only supported for GCP and Okta Applications.

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

resource "britive_constraint" "new" {
  profile_id = britive_profile.new.id
  permission_name = britive_profile_permission.new.permission_name
  permission_type = britive_profile_permission.new.permission_type
  constraint_type = "bigquery.datasets"
  name = "pivy-231191.dataset"
}

resource "britive_constraint" "new1" {
    profile_id      = "h59ih6p1537xxxxxxxxx"
    permission_name = "Storage Admin"
    permission_type = "role"
    constraint_type = "condition"
    title           = "Condition Constraint"
    description     = "Condition Constraint Description"
    expression      = "request.time < timestamp('2025-01-11 19:12:24')"
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required, Forces new resource) The identifier of the profile.

* `permission_name` - (Required, Forces new resource) Name of the permission associated with the profile.

* `permission_type` - (Optional, Forces new resource) The type of permission. Defaults to "role". The value is case-sensitive and must be updated by getting the same from the API response for an import. (https://docs.britive.com/docs/manage-profile-permissions)

* `constraint_type` - (Required, Forces new resource) The constraint type for a given profile permission. The value is case-sensitive and must be updated by getting the same from the britive_supported_constraints data source for an import.

* `name` - (Optional, Forces new resource) Name of the constraint. If `name` is set, then `title`, `expression`, and `description` cannot be set, and vice versa.

* `title` - (Optional, Forces new resource) Title of the condition constraint. Used only with the `condition` constraint type.

* `expression` - (Optional, Forces new resource) Expression of the condition constraint. Used only with the `condition` constraint type.

* `description` - (Optional, Forces new resource) Description of the condition constraint. Used only with the `condition` constraint type.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profile_id}}/permissions/{{permission_name}}/{{permission_type}}/constraints/{{constraint_type}}/{{constraint_name or constraint_title}}`

## Import

You can import a Britive profile using any of these accepted formats:

```sh
terraform import britive_constraint.new paps/{{profile_id}}/permissions/{{permission_name}}/{{permission_type}}/constraints/{{constraint_type}}/{{constraint_name or constraint_title}}
terraform import britive_constraint.new {{profile_id}}/{{permission_name}}/{{permission_type}}/{{constraint_type}}/{{constraint_name or constraint_title}}
```
