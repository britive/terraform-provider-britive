---
subcategory: "Identity Management"
layout: "britive"
page_title: "britive_tag Resource - britive"
description: |-
  Manages tags for the Britive provider.
---

# britive_tag Resource

This resource allows you to create and configure a Britive tag.

!> This resource does not allow you to add external tags. External tags are tags managed by Identity Providers other than the Britive Default Identity Provider.  
If you try to add an external tag using this resource, an error message will be displayed.

## Example Usage

```hcl
data "britive_identity_provider" "idp" {
    name = "My Identity Provider"
}

resource "britive_tag" "new" {
    name = "My Tag"
    description = "My Tag Description"
    identity_provider_id = data.britive_identity_provider.idp.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of Britive tag.

* `description` - (Optional) A description of the Britive tag.

* `identity_provider_id` - (Required) The unique identity of the identity provider associated with the Britive tag.

* `disabled` - (Optional) The status of the Britive tag. By default, the Britive tag is enabled. To disable a Britive tag, set `disabled = true`.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `id` - The identity of the Britive tag.
* `external` - The boolean attribute that indicates whether the tag is external or not.

## Import

You can import the Britive tag using any of these accepted formats:

```sh
terraform import britive_tag.new tags/{{tag_name}}
terraform import britive_tag.new {{tag_name}}
```
