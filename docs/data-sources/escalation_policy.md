# britive_escalation_policy Data Source

This data source allows you to retrieve information about a specific escalation policy required for configuring IM settings in Britive.

## Example Usage

```hcl
data "britive_escalation_policy" "new_policy" {
  name = "TF_IM_Connection"
  im_connection_id = "yastd87awd-8q6wtd-as86dt-aw8we7khhd"
}

output "policy_id" {
  value = data.britive_escalation_policy.new_policy.id
}
```

## Argument Reference

The following argument is supported:

- `name` (Required) – The name of the escalation to retrieve.
- `im_connection_id` (Required) – Id of IM connection.

## Attribute Reference

The following attributes are exported:

- `id` – The unique identifier of the escalation policy.