---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_application_scan_schedule Resource - britive"
description: |-
  Manages a scheduled scan for an application for the Britive provider.
---

# britive_application_scan_schedule Resource

The `britive_application_scan_schedule` resource allows you to manage scheduled scans for a
Britive application. Each instance of this resource represents one scheduled scan ("task");
an application can have any number of them.

The underlying scan task service for an application is registered automatically by the API
when the application itself is created - there's nothing to configure separately for that.
Whether scanning is actually turned on for the application as a whole is managed
independently via `scan_enabled` on [`britive_application`](application.md), since it's an
application-wide toggle, not scoped to any one schedule - see that resource's `scan_enabled`
Semantics section for details.

This resource's schema deliberately mirrors
[`britive_resource_manager_resource_type_schedule_scan`](resource_manager_resource_type_schedule_scan.md)
(`day_of_week`, `day_of_month`, `start_time`), since both configure tasks on the same
underlying scan task-service API. The one addition is `hour_interval`, for the `Hourly`
frequency type, which has no resource-manager equivalent.

## Application Type Support

Scheduled scanning itself, `scope`, and `org_scan` are each supported only for certain
`application_type` values (see [`britive_application`](application.md)). The provider
validates this on `terraform apply` against the application's actual type, resolved live via
the API rather than trusted from local config:

| Application Type       | Scheduled Scanning | `scope` | `org_scan` |
|-------------------------|:---:|:---:|:---:|
| AWS                     | Yes | Yes | Yes |
| AWS Standalone          | Yes | Yes | No  |
| Azure                   | Yes | No  | No  |
| Azure WIF               | Yes | No  | No  |
| GCP                     | Yes | No  | No  |
| GCP Standalone          | Yes | No  | No  |
| GCP WIF                 | Yes | No  | No  |
| Google Workspace        | Yes | No  | No  |
| Kubernetes              | No  | No  | No  |
| Britive                 | Yes | Yes | No  |
| Oracle WIF              | Yes | Yes | Yes |
| Okta                    | Yes | Yes | No  |
| Snowflake               | Yes | Yes | Yes |
| Snowflake Standalone    | Yes | Yes | No  |

Using this resource at all against a `Kubernetes` application, or setting `scope`/`org_scan`
for an application type whose column above is `No`, fails `terraform apply` with an
"Unsupported Application Scan Schedule Configuration" error before any API call that would
create or modify the schedule.

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

  scope {
    type  = "EnvironmentGroup"
    value = "ou-example-group-1"
  }
  scope {
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

  scope {
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

  scope {
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
* `hour_interval` - (Optional) Number of hours between scans (e.g. `5` = every 5 hours). Required when `frequency_type = "Hourly"`; must be unset otherwise. Has no equivalent on `britive_resource_manager_resource_type_schedule_scan`, since `Hourly` is an application-only frequency type.
* `start_time` - (Optional) The time of day the scan runs, in 24-hour `"HH:MM"` format. Required for `Daily`/`Weekly`/`Monthly`; must be unset for `Hourly`, which runs on its own interval instead.
* `org_scan` - (Optional) Whether to scan the entire organization, ignoring `scope`. Left **unmanaged** when omitted from config - the provider only sends this field to the API when it's explicitly present in config; the exported value otherwise just reflects whatever the task's actual `orgScan` status already is. Exported as `null` (not `false`) for application types that don't support `org_scan` at all, since the API omits the field entirely for those rather than returning an explicit `false`.
* `scope` - (Optional) Environments/environment groups the scan is restricted to. Omit entirely (and set `org_scan = true`) to scan the whole organization. Each block supports:
  * `type` - (Required) One of `Environment`, `EnvironmentGroup` (case-insensitive).
  * `value` - (Required) The environment or environment group, by name or ID (mirrors [`britive_profile`](profile.md)'s `associations` block). Two `scope` blocks of the same `type` may reference the same environment/environment group via different forms (one by name, one by ID) without conflict - each is tracked and refreshed independently.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The composite identifier of the scan schedule.
* `application_type` - The associated application's type. Resolved automatically from `application_id` and cached, since it never changes for a given application - not independently managed by this provider. Used internally to validate `scope`/`org_scan`/scheduled-scan support (see Application Type Support above).
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
