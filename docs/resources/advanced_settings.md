# britive_advanced_settings Resource

Manages advanced settings for Britive resources.

## Overview

The `britive_advanced_settings` resource allows you to configure and manage advanced settings for supported Britive resources. This includes settings such as justification requirements and ITSM (IT Service Management) integration. Use this resource to enforce additional controls and workflows on your Britive-managed entities.

## Supported Resource Types

This resource supports advanced settings for the following resource types:

- `APPLICATION` – Britive application resource
- `PROFILE` – Britive application profile resource
- `PROFILE_POLICY` – Britive application profile policy resource

## Example Usage

```hcl
resource "britive_advanced_settings" "example" {
  resource_id   = "8s7fw93frt09gng8sy98r3sjdw83"
  resource_type = "APPLICATION" # (PROFILE or PROFILE_POLICY)

  justification_settings {
    is_justification_required = true
    justification_regex        = "test_advanced_settings"
  }

  itsm {
    connection_id       = "729xj-8e2e-22938-2293nx"
    connection_type     = "jira"
    is_itsm_enabled     = true

    itsm_filter_criteria {
      supported_ticket_type = "issue"
      filter                = jsonencode({
        jql = "advanced_settings_test"
      })
    }
  }
}
```

-> The format of `resource_id` must correspond to the specific `resource_type` you are configuring. Ensure that the `resource_id` and `resource_type` are associated with the same Britive resource.

## Argument Reference

The following arguments are supported:

- `resource_id` (Required) – The unique identifier of the resource for which advanced settings are being managed.
- `resource_type` (Required) – The type of resource. Must be one of: `APPLICATION`, `PROFILE`, or `PROFILE_POLICY`.
- `justification_settings` (Optional):
  - `justification_id` (Computed) – The ID of the justification setting.
  - `is_justification_required` (Required) – Whether justification is required for actions on the resource.
  - `justification_regex` (Required) – A regular expression to validate justification input.
- `itsm` (Optional):
  - `itsm_id` (Computed) – The ID of the ITSM setting.
  - `connection_id` (Required) – The ID of the ITSM connection.
  - `connection_type` (Required) – The type of ITSM connection (e.g., Jira, ServiceNow).
  - `is_itsm_enabled` (Required) – Whether ITSM integration is enabled.
  - `itsm_filter_criteria` (Required):
      - `filter` (Required) – The filter definition (e.g., JQL for Jira).
      - `supported_ticket_type` (Required) – The supported ticket type for the filter criteria. Example: `"issue"`, `"request"`.

## Resource Type Examples

Below are configuration examples for each supported resource type. The structure is the same, except for the `resource_id` and `resource_type` values.

### APPLICATION

```hcl
resource "britive_advanced_settings" "application" {
  resource_id   = "{{ApplicationID}}"
  resource_type = "APPLICATION"
  # ...advanced settings configuration...
}
```

### PROFILE

```hcl
resource "britive_advanced_settings" "profile" {
  resource_id   = "{{ProfileID}}"
  resource_type = "PROFILE"
  # ...advanced settings configuration...
}
```

### PROFILE_POLICY

```hcl
resource "britive_advanced_settings" "profile_policy" {
  resource_id   = "paps/{{ProfileID}}/policies/{{PolicyID}}"
  resource_type = "PROFILE_POLICY"
  # ...advanced settings configuration...
}
```

-> Replace the `resource_id` and `resource_type` values according to the resource for which you are managing advanced settings. The rest of the configuration remains the same.

## Import
 
Advanced settings can be imported using one of the following formats:
 
- For `APPLICATION` or `PROFILE`:
 
```SH
terraform import britive_advanced_settings.new {{resource_id}}/{{resource_type}}
terraform import britive_advanced_settings.new 8kjchct9fdxunt1ntjp98gx/application
terraform import britive_advanced_settings.new 89susd3hdy83dhd8h87euhd8/profile
```
 
- For `PROFILE_POLICY`:
 
```SH
terraform import britive_advanced_settings.new paps/{{profileId}}/policies/{{policyId}}/profile_policy
terraform import britive_advanced_settings.new paps/9asduahsd83h3e8/policies/89sus-d3hdy-83dhd8-h87euhd8/profile_policy
```

-> During the import process, only advanced settings that are not inherited will be imported.