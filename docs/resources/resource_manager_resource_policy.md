---
subcategory: "resource_manager"
layout: "britive"
page_title: "britive_resource_manager_resource_policy Resource - britive"
description: |-
  Manages resource manager resource policy for the Britive provider.
---

# britive_resource_manager_resource_policy Resource

This resource allows you to create and manage policies associated with a resource manager profile in Britive.

## Example Usage

```hcl
resource "britive_resource_manager_resource_policy" "example" {
    policy_name  = "New_Policy"
    description  = "New_Policy Description"
    members      = jsonencode(
        {
            serviceIdentities = [
                {
                    name = "service-identity-99"
                },
                {
                    name = "service-identity-test"
                },
            ]
            tags              = [
                {
                    name = "tag_005"
                },
                {
                    name = "tag_001"
                },
            ]
            users             = [
                {
                    name = "lfox"
                },
                {
                    name = "jgordon"
                },
            ]
        }
    )
    condition    = jsonencode(
        {
            ipAddress = "192.162.0.0/16,10.10.0.10"
            timeOfAccess = {
                dateSchedule = {
                    fromDate = "2022-10-29 10:30:00"
                    toDate = "2022-11-05 18:30:00"
                    timezone = "Asia/Calcutta"
                }
                daysSchedule = {
                    fromTime = "16:30:00"
                    toTime = "19:30:00"
                    timezone = "Asia/Calcutta"
                    days = ["SATURDAY", "SUNDAY"]
                }
            }
        }
    )
    access_type  = "Allow"
    consumer     = "resourcemanager"
    is_active    = true
    is_draft     = false
    is_read_only = false
    resource_labels {
        label_key = "Environment"
        values    = ["QA Subscription"]
    }
    resource_labels {
        label_key = "EnvironmentGroup"
        values    = ["Development"]
    }
}
```

## Argument Reference

The following arguments are supported:

* `policy_name` - (Required) The name of the profile policy.
* `description` - (Optional) A description of the profile policy.
* `members` - (Optional) Set of members under this policy. This is a JSON formatted string. Includes the usernames of `serviceIdentities`, `tags`, and `users`.
* `condition` - (Optional) Set of conditions applied to this policy. This is a JSON formatted string.  
  * The `condition` block can include:
    * `ipAddress` - Comma separated IP addresses in CIDR, dotted decimal format or `null`.
    * `timeOfAccess` - Can be scheduled based on date, days, both or `null`.
      * `dateSchedule` - Should contain `fromDate`, `toDate` in format "YYYY-MM-DD HH:MM:SS" and `timezone` as a string from https://en.wikipedia.org/wiki/List_of_tz_database_time_zones. If not required, set to `null`.
      * `daysSchedule` - Should contain `fromTime`, `toTime` in format "HH:MM:SS", `timezone` as a string from https://en.wikipedia.org/wiki/List_of_tz_database_time_zones and `days` as a list of strings. If not required, set to `null`.
* `access_type` - (Optional) Type of access the policy provides. This can have two values `"Allow"`/`"Deny"`. Default: `"Allow"`.
* `access_level` - (Optional) Level of access the policy provides. This can have value as `"manage"`.
* `consumer` - (Optional) The consumer service. Default: `"resourcemanager"`.
* `is_active` - (Optional) Indicates if a policy is active. Default: `true`.
* `is_draft` - (Optional) Indicates if a policy is a draft. Default: `false`.
* `is_read_only` - (Optional) Indicates if a policy is read only. Default: `false`.
* `resource_labels` - (Optional) List of resource labels for the policy. Each block supports:
  * `label_key` - (Required) Name of the resource label.
  * `values` - (Required) List of values for the resource label.

## Attribute Reference

In addition to the above arguments, the following attribute is exported:

* `id` - An identifier of the policy for the profile with format `resource-manager/policies/{{policy_id}}`

## Import

You can import a policy for the profile using any of these accepted formats:

```sh
terraform import britive_resource_manager_resource_policy.example resource-manager/policies/{{policy_id}}
terraform import britive_resource_manager_resource_policy.example {{policy_name}}

terraform import britive_resource_manager_resource_policy.example resource-manager/policies/New_Policy
terraform import britive_resource_manager_resource_policy.example New_Policy
```