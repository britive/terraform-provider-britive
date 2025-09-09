---
subcategory: ""
layout: "britive"
page_title: "britive_all_connections Data Source - britive"
description: |-
  Retrieves all connections.
---

# britive_all_connections Data Source

This data source retrieves a list of all available connections in Britive.

## Example Usage

```hcl
data "britive_all_connections" "all" {
  setting_type = "ITSM"
}

output "all_connections" {
  value = data.britive_all_connections.all.connections
}
```
## Argument Reference

The following argument is supported:

- `setting_type` (Optional) – The type of advanced setting. Defaults to 'ITSM' if not specified. Supported types are ITSM and IM.

## Attribute Reference

The following attributes are exported:

- `connections` – A set of all connections, each containing:
  - `id` – The unique identifier of the connection.
  - `name` – The name of the connection.
  - `type` – The type of the connection.
  - `auth_type` – The authentication type of the connection.