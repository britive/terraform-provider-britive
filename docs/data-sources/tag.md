---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_tag Data Source - britive"
description: |-
  Retrieves information of a tag.
---

# britive_tag Data Source

Use this data source to retrieve information about a Britive tag by name or tag ID.

Tag names are unique per identity provider only. When a name matches tags on several identity providers, the lookup fails and lists the candidates; set `identity_provider_id` to pick one, or look the tag up by `tag_id`.

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

### Lookup by name on a specific identity provider

```hcl
data "britive_identity_provider" "okta" {
    name = "Okta"
}

data "britive_tag" "example" {
    name                 = "Platform-Admins"
    identity_provider_id = data.britive_identity_provider.okta.id
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

* `identity_provider_id` - (Optional) With `name`, restricts the lookup to the tag on this identity provider. Cannot be combined with `tag_id`.

## Attribute Reference

In addition to the above arguments, the following attributes are exported:

* `id` - An identifier for the tag (same as `tag_id`).

* `name` - The name of the tag.

* `tag_id` - The unique identifier of the tag.
* `identity_provider_id` - The identity provider the tag belongs to.
* `description` - The description of the tag.
* `disabled` - Whether the tag is disabled.
* `requestable` - Whether the tag is requestable.
* `external` - Whether the tag is connected to its identity provider through SCIM.
* `external_id` - The identity provider's identifier for a connected tag, when supplied.
* `attributes` - The attributes of the tag. Each entry has `attribute_name` and `attribute_value`.
