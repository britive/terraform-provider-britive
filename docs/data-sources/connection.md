---
subcategory: ""
layout: "britive"
page_title: "britive_connection Data Source - britive"
description: |-
  Retrieves information of connection.
---

# britive_connection Data Source

This data source allows you to retrieve information about a specific connection required for configuring advanced settings in Britive.

## Example Usage

```hcl
data "britive_connection" "my_conn" {
  name = "BD-Jira-0601-1"
  setting_type = "ITSM"
}

output "connection_id" {
  value = data.britive_connection.my_conn.id
}

output "connection_name" {
  value = data.britive_connection.my_conn.name
}

output "connection_type" {
  value = data.britive_connection.my_conn.type
}

output "connection_auth_type" {
  value = data.britive_connection.my_conn.auth_type
}
```

## Argument Reference

The following argument is supported:

- `name` (Required) – The name of the connection to retrieve.
- `setting_type` (Optional) – The type of advanced setting. Defaults to 'ITSM' if not specified. Supported types are ITSM and IM.

## Attribute Reference

The following attributes are exported:

- `id` – The unique identifier of the connection.
- `name` – The name of the connection.
- `type` – The type of the connection.
- `auth_type` – The authentication type of the connection.