# britive_connection and britive_all_connections Data Sources

These data sources allow you to retrieve information about connections required for configuring advanced settings in Britive.

## britive_connection

The `britive_connection` data source retrieves details for a specific connection by name.

### Example Usage

```hcl
data "britive_connection" "my_conn" {
  name = "BD-Jira-0601-1"
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

### Argument Reference

The following argument is supported:

- `name` (Required) – The name of the connection to retrieve.

### Attribute Reference

The following attributes are exported:

- `id` – The unique identifier of the connection.
- `name` – The name of the connection.
- `type` – The type of the connection.
- `auth_type` – The authentication type of the connection.

---

## britive_all_connections

The `britive_all_connections` data source retrieves a list of all available connections.

### Example Usage

```hcl
data "britive_all_connections" "all" {}

output "all_connections" {
  value = data.britive_all_connections.all.connections
}
```

### Attribute Reference

The following attributes are exported:

- `connections` – A set of all connections, each containing:
  - `id` – The unique identifier of the connection.
  - `name` – The name of the connection.
  - `type` – The type of the connection.
  - `auth_type` – The authentication type of the connection.
