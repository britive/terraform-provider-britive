---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_tag Data Source - britive"
description: |-
  Retrieves information of a tag.
---

# britive_tag Data Source

Use this data source to retrieve information about a Britive tag by name or tag ID.

## Example Usage

### Lookup by name

```hcl
data "britive_tag" "example" {
    name = "My Tag"
}

output "tag_id" {
    value = data.britive_tag.example.tag_id
}
```

### Lookup by tag ID

```hcl
data "britive_tag" "example" {
    tag_id = "2utfgbi5320i2lnyhnmt"
}

output "tag_name" {
    value = data.britive_tag.example.name
}
```

### Use tag ID in another resource

```hcl
data "britive_tag" "owner_tag" {
    name = "My Owner Tag"
}

resource "britive_tag_owner" "example" {
    tag_id = britive_tag.my_tag.id

    tag {
        id = data.britive_tag.owner_tag.tag_id
    }
}
```

## Argument Reference

Exactly one of the following arguments must be provided:

* `name` - (Optional) The name of the tag.

* `tag_id` - (Optional) The unique identifier of the tag.

## Attribute Reference

In addition to the above arguments, the following attributes are exported:

* `id` - An identifier for the tag (same as `tag_id`).

* `name` - The name of the tag.

* `tag_id` - The unique identifier of the tag.
