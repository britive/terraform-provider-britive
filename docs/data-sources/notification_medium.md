---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_notification_medium Data Source - britive"
description: |-
  Retrieves information of a notification medium.
---

# britive_notification_medium Data Source

Use this data source to retrieve information about a Britive notification medium by name.

## Example Usage

```hcl
data "britive_notification_medium" "email" {
    name = "Email"
}

resource "britive_application_access_builder_settings" "this" {
    # ...

    notification_medium {
        id = data.britive_notification_medium.email.id
    }
}
```

## Argument Reference

* `name` - (Required) The name of the notification medium, for example `Email`.

## Attribute Reference

In addition to the above arguments, the following attributes are exported:

* `id` - The unique identifier of the notification medium.

* `type` - The type of the notification medium, for example `email`.

* `description` - The description of the notification medium.
