---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_application_scan_schedule Resource - britive"
description: |-
  Manages a scheduled scan for an application for the Britive provider.
---

# britive_application_scan_schedule Resource

The `britive_application_scan_schedule` resource allows you to manage scheduled scans for a
Britive application. Each instance of this resource represents one scheduled scan.

The underlying scan task service for an application is registered automatically by the API
when the application itself is created - there's nothing to configure separately for that.
Whether scanning is actually turned on for the application as a whole is managed
independently via `scan_enabled` on [`britive_application`](application.md), since it's an
application-wide toggle, not scoped to any one schedule - see that resource's `scan_enabled`
Semantics section for details.

Exactly one of `hour_interval`, `day_of_week`, or `day_of_month` applies, depending on
`frequency_type` (`Hourly`, `Weekly`, or `Monthly` respectively) - the others must be left
unset. See the Argument Reference below for details.

-> Not every application supports `associations`/`org_scan` to the same extent. Support is
derived per-application from two live catalog flags (`supportsEnvironmentScanning` and
`requiresHierarchicalModel`), not from `application_type` - so this stays accurate for
application types added after this provider version. The provider validates this on
`terraform apply` and fails fast with a clear error before it would otherwise hit a less
specific error from the backend.

## Example Usage

### Hourly

```hcl
resource "britive_application_scan_schedule" "hourly_example" {
  application_id = britive_application.example.id
  name           = "hourly-scan"
  frequency_type = "Hourly"
  hour_interval  = 5
  org_scan       = true
}
```

### Daily, scoped to specific environments and environment groups

```hcl
resource "britive_application_scan_schedule" "daily_example" {
  application_id = britive_application.example.id
  name           = "daily-scan"
  frequency_type = "Daily"
  start_time     = "11:00"

  associations {
    type  = "EnvironmentGroup"
    value = "ou-example-group-1"
  }
  associations {
    type  = "Environment"
    value = "111111111111"
  }
}
```

### Weekly

```hcl
resource "britive_application_scan_schedule" "weekly_example" {
  application_id = britive_application.example.id
  name           = "weekly-scan"
  frequency_type = "Weekly"
  day_of_week    = "Monday"
  start_time     = "21:45"

  associations {
    type  = "Environment"
    value = "222222222222"
  }
}
```

### Monthly

```hcl
resource "britive_application_scan_schedule" "monthly_example" {
  application_id = britive_application.example.id
  name           = "monthly-scan"
  frequency_type = "Monthly"
  day_of_month   = 7
  start_time     = "13:30"

  associations {
    type  = "EnvironmentGroup"
    value = "ou-example-group-2"
  }
}
```

## Argument Reference

* `application_id` - (Required) The ID of the associated application. Forces replacement if changed.
* `name` - (Required) The name of the scheduled scan.
* `frequency_type` - (Required) How often the scan runs. One of `Hourly`, `Daily`, `Weekly`, `Monthly` (case-insensitive).
* `day_of_week` - (Optional) The day of the week the scan runs. Required when `frequency_type = "Weekly"`; must be unset otherwise. One of `Sunday`/`Sun`, `Monday`/`Mon`, `Tuesday`/`Tue`, `Wednesday`/`Wed`, `Thursday`/`Thu`, `Friday`/`Fri`, `Saturday`/`Sat` (case-insensitive).
* `day_of_month` - (Optional) The day of the month (`1`-`31`) the scan runs. Required when `frequency_type = "Monthly"`; must be unset otherwise.
* `hour_interval` - (Optional) Number of hours between scans (e.g. `5` = every 5 hours). Required when `frequency_type = "Hourly"`; must be unset otherwise.
* `start_time` - (Optional) The time of day the scan runs, in 24-hour `"HH:MM"` format. Required for `Daily`/`Weekly`/`Monthly`; must be unset for `Hourly`, which runs on its own interval instead.
* `org_scan` - (Optional) Whether to scan the entire organization, ignoring `associations`. Left **unmanaged** when omitted from config - the provider only sends this field to the API when it's explicitly present in config; the exported value otherwise just reflects whatever the task's actual `orgScan` status already is. Exported as `null` (not `false`) for applications that don't support `org_scan` at all, since the API omits the field entirely for those rather than returning an explicit `false`. Not every application supports `org_scan` - see the note above.
* `associations` - (Optional) Environments/environment groups the scan is restricted to. Omit entirely (and set `org_scan = true`) to scan the whole organization. Each block supports:
  * `type` - (Required) One of `Environment`, `EnvironmentGroup` (case-insensitive).
  * `value` - (Required) The environment or environment group, by name or ID. Two `associations` blocks of the same `type` may reference the same environment/environment group via different forms (one by name, one by ID) without conflict - each is tracked and refreshed independently.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the scan schedule.
* `task_id` - The unique identifier of the scheduled scan task.
* `next_run` - The next scheduled run timestamp (epoch milliseconds).

## Import

Scan schedules can be imported using their composite ID:

```sh
terraform import britive_application_scan_schedule.example apps/<application_id>/scan-schedules/<task_id>
```

## Delete Behavior

Destroying this resource deletes the individual scan schedule task only. It does not touch
the application's scan task service, the `scan_enabled` toggle, or any sibling scan schedules
for the same application. Destroying the parent [`britive_application`](application.md)
itself, however, deletes its scan task service (and therefore every scan schedule under it)
as a cascade on the backend - there's no need to destroy scan schedules separately first.
