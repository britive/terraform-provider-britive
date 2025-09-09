---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource_label Resource - britive"
description: |-
  Manages resource labels for the Britive provider.
---

# britive_resource_manager_resource_label Resource

The `britive_resource_manager_resource_label` resource allows you to create, update, and manage resource labels in Britive. Each label can have multiple values, and each value can include additional metadata.

## Example Usage

```hcl
resource "britive_resource_manager_resource_label" "example" {
    name         = "example_label"
    description  = "This is an example resource label for categorizing resources by environment."
    label_color  = "#FF5733"
    values {
      name = "Production"
      description = "Resources used in the production environment"
    }
    values {
      name = "Development"
      description = "Resources used in the development environment"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the resource label. Only letters, numbers, hyphens (`-`), and underscores (`_`) are allowed, no other special characters. Used to uniquely identify the label within Britive.
* `description` - (Optional) A description for the resource label. Useful for providing context or usage information.
* `label_color` - (Optional) The color associated with the label, specified as a hex code in the format `#RRGGBB`. Only accepts values where `RR`, `GG`, and `BB` are hexadecimal digits (`0-9`, `a-f`, `A-F`). For example, `#FF5733`. Color is case-insensitive and helps visually distinguish labels in the UI.
* `values` - (Optional) A set of resource label values. Each value block supports the following:
  * `name` - (Required) The name of the label value. This is the actual value that can be assigned to resources.
  * `description` - (Optional) A description for the label value, providing additional context.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `internal` - Indicates if the label is internal to Britive.
* `values.value_id` - The unique identifier for each label value.
* `values.created_by` - The user ID who created the label value.
* `values.updated_by` - The user ID who last updated the label value.
* `values.created_on` - The timestamp when the label value was created.
* `values.updated_on` - The timestamp when the label value was last updated.

## Import

Resource labels can be imported using their `label_id`:

```sh
terraform import britive_resource_manager_resource_label.example resource-manager/resource-labels/{{label_id}}
terraform import britive_resource_manager_resource_label.example resource-manager/resource-labels/abc123def456
```
