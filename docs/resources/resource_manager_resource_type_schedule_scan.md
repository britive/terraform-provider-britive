---
subcategory: "Resource Manager"
layout: "britive"
page_title: "britive_resource_manager_resource_type_schedule_scan Resource - britive"
description: |-
  Manages a scheduled scan for a resource type for the Britive provider.
---

# britive_resource_manager_resource_type_schedule_scan Resource

The `britive_resource_manager_resource_type_schedule_scan` resource allows you to manage
scheduled scans for a Britive resource manager resource type. Each instance of this resource
represents one scheduled scan ("task"); a resource type can have any number of them.

The underlying scan task service for a resource type is created automatically by the API the
first time a schedule scan is created for it - there's nothing to configure separately for
that. Whether scanning is actually turned on for the resource type as a whole is managed
independently via `scan_enabled` on
[`britive_resource_manager_resource_type`](resource_manager_resource_type.md), since it's a
resource-type-wide toggle, not scoped to any one schedule. Note that `scan_enabled` cannot be
set to `true` in the same apply that creates the resource type's first schedule scan - see
["Enabling Scanning" in that resource's docs](resource_manager_resource_type.md#enabling-scanning-scan_enabled-requires-two-applies)
for why and the required two-step workflow.

## Example Usage

### Weekly

```hcl
resource "britive_resource_manager_resource_type_schedule_scan" "weekly_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name              = "weekly-scan"
  frequency_type    = "Weekly"
  day_of_week       = "Monday"
  start_time        = "11:00"

  resource_labels {
    label_key = "Environment"
    values    = ["Production"]
  }
  resource_labels {
    label_key = "Team"
    values    = ["Platform", "Security"]
  }
}
```

### Monthly

```hcl
resource "britive_resource_manager_resource_type_schedule_scan" "monthly_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name              = "monthly-scan"
  frequency_type    = "Monthly"
  day_of_month      = 6
  start_time        = "03:30"
}
```

### Daily

```hcl
resource "britive_resource_manager_resource_type_schedule_scan" "daily_example" {
  resource_type_id = britive_resource_manager_resource_type.example.id
  name              = "daily-scan"
  frequency_type    = "Daily"
  start_time        = "06:30"
}
```

## Argument Reference

* `resource_type_id` - (Required) The ID of the associated resource type. Forces replacement if changed.
* `name` - (Required) The name of the scheduled scan.
* `description` - (Optional) The description of the scheduled scan.
* `frequency_type` - (Required) How often the scan runs. One of `Daily`, `Weekly`, `Monthly` (case-insensitive).
* `day_of_week` - (Optional) The day of the week the scan runs. Required when `frequency_type = "Weekly"`; must be unset otherwise. One of `Sunday`/`Sun`, `Monday`/`Mon`, `Tuesday`/`Tue`, `Wednesday`/`Wed`, `Thursday`/`Thu`, `Friday`/`Fri`, `Saturday`/`Sat` (case-insensitive).
* `day_of_month` - (Optional) The day of the month (`1`-`31`) the scan runs. Required when `frequency_type = "Monthly"`; must be unset otherwise.
* `start_time` - (Required) The time of day the scan runs, in 24-hour `"HH:MM"` format.
* `resource_labels` - (Optional) Resource labels to filter the scan to. Omit entirely to scan every resource of this type. Each block supports:
  * `label_key` - (Required) The name of the resource label. Each `label_key` may appear in at most one `resource_labels` block.
  * `values` - (Required) The label's selected values to filter the scan to.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the schedule scan.
* `task_id` - The unique identifier of the scheduled scan task.
* `created_by` - The user who created the scheduled scan.
* `created` - The creation timestamp of the scheduled scan (epoch milliseconds).
* `modified` - The last-modified timestamp of the scheduled scan (epoch milliseconds). Null until the first update.
* `modified_by` - The user who last modified the scheduled scan.
* `next_run` - The next scheduled run timestamp (epoch milliseconds).

## Import

Schedule scans can be imported using their composite ID:

```sh
terraform import britive_resource_manager_resource_type_schedule_scan.example resource-manager/resource-types/<resource_type_id>/schedule-scans/<task_id>
```

## Delete Behavior

Destroying this resource deletes the individual schedule scan task only. It does not touch
the resource type's scan task service, the `scan_enabled` toggle, or any sibling schedule
scans for the same resource type.
