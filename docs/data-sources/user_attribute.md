---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_user_attribute Data Source - britive"
description: |-
  Retrieves information of a user attribute.
---

# britive_user_attribute Data Source

Use this data source to retrieve information about a Britive user attribute by name or attribute schema ID.

## Example Usage

### Lookup by name

```hcl
data "britive_user_attribute" "department" {
    name = "department"
}

output "attribute_schema_id" {
    value = data.britive_user_attribute.department.attribute_schema_id
}
```

### Lookup by attribute schema ID

```hcl
data "britive_user_attribute" "department" {
    attribute_schema_id = "0h3f6k7example"
}

output "attribute_name" {
    value = data.britive_user_attribute.department.name
}
```

## Argument Reference

Exactly one of the following arguments must be provided:

* `name` - (Optional) The name of the user attribute.

* `attribute_schema_id` - (Optional) The unique identifier of the user attribute.

## Attribute Reference

In addition to the above arguments, the following attributes are exported:

* `id` - An identifier for the user attribute (same as `attribute_schema_id`).

* `name` - The name of the user attribute.

* `attribute_schema_id` - The unique identifier of the user attribute.
