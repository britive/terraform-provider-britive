---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_user Data Source - britive"
description: |-
  Retrieves information of a user.
---

# britive_user Data Source

Use this data source to retrieve information about a Britive user by username or user ID.

## Example Usage

### Lookup by username

```hcl
data "britive_user" "example" {
    name = "jdoe"
}

output "user_id" {
    value = data.britive_user.example.user_id
}
```

### Lookup by user ID

```hcl
data "britive_user" "example" {
    user_id = "lba4rcrtpd3nldszfeds"
}

output "username" {
    value = data.britive_user.example.name
}
```

## Argument Reference

Exactly one of the following arguments must be provided:

* `name` - (Optional) The username of the user.

* `user_id` - (Optional) The unique identifier of the user.

## Attribute Reference

In addition to the above arguments, the following attributes are exported:

* `id` - An identifier for the user (same as `user_id`).

* `name` - The username of the user.

* `user_id` - The unique identifier of the user.
