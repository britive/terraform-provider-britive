---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_custom_template Resource - britive"
description: |-
  Manages custom template.
---

# britive_custom_template Resource

This resource allows you to manage custom template for application.

## Example Usage

```hcl
resource "britive_custom_template" "new" {
    template_name = "Custom_Template.baml"
    template = file("Custom_Template.baml")
}
```

## Argument Reference

The following arguments are supported:

* `template_name` - (Required) Template file name.

* `permission_name` - (Required) Template file.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with the format `apps`
