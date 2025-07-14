# britive_policy Resource

-> When using this version for the first time, you may encounter noisy diffs caused by the reordering of resource argument values. 

This resource allows you to create and configure a policy.

## Example Usage

```hcl
resource "britive_policy" "new" {
    name         = "New Policy"
    description  = "New Policy Description"
    access_type  = "Allow"
    members      = jsonencode(
        {
            serviceIdentities = [
                {
                    name = "service-identity-45B"
                },
                {
                    name = "service-identity-99"
                },
            ]
            tags              = [
                {
                    name = "tag_004"
                },
                {
                    name = "tag_005"
                },
            ]
            tokens            = [
                {
                    name = "token_01"
                },
                {
                    name = "token_12"
                },
            ]
            users             = [
                {
                    name = "apennyworth"
                },
                {
                    name = "skyle"
                },
            ]
        }
    )
    permissions  = jsonencode(
        [
            {
                name = "Permission_20"
            },
            {
                name = britive_permission.new.name
            },
        ]
    )
    roles        = jsonencode(
        [
            {
                name = "Role_21"
            },
            {
                name = britive_role.new.name
            },
        ]
    )
    condition    = jsonencode(
        {
            approval     = {
                approvers          = {
                    tags    = [
                        "tag_08",
                        "tag_11",
                    ]
                    userIds = [
                        "hdent",
                        "jblake",
                    ]
                }
                isValidForInDays   = false
                notificationMedium = [
                    "Teams",
                    "Email"
                ]
                timeToApprove      = 30
                validFor           = 120
            }
            ipAddress    = "192.162.0.0/16,10.10.0.10"
            timeOfAccess = {
                "dateSchedule": {
                    "fromDate": "2022-10-29 10:30:00",
                    "toDate": "2022-11-05 18:30:00",
                    "timezone": "Asia/Calcutta"
                },
                "daysSchedule": {
                    "fromTime": "16:30:00",
                    "toTime": "19:30:00",
                    "timezone": "Asia/Calcutta",
                    "days": [
                        "SATURDAY",
                        "SUNDAY"
                    ]
                }
            }
        }
    )
    is_active    = true
    is_draft     = false
    is_read_only = false
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy.

* `description` - (Optional) A description of the policy.

* `access_type` - (Optional) Type of access the policy provides. This can have two values "Allow"/"Deny". Default:`"Allow"`.

* `members` - (Optional) Set of members under this policy. This is a JSON formatted string. Includes the usernames of `serviceIdentities`, `tags`, `tokens` and `users`

* `permissions` - (Optional) Permissions associated to the policy. Either a role/permission is to be assigned to a policy.

* `roles` - (Optional) Roles associated to the policy. Either a role/permission is to be assigned to a policy.

* `condition` - (Optional) Set of conditions applied to this policy. This is a JSON formatted string.
 * The `approvers` block under `approval` includes the username for `tags` and `userIds`, and/or slack channel Ids for `channelIds` and `slackAppChannels` as a list of strings. It also includes the `teamsAppChannels`, as a list of maps. Each map containing the keys `team` as the name of the team and `channels` as names of the channels to the corresponding team.
 * The `approval` block also includes:
   * `notificationMedium` as a list of strings.
   * `timeToApprove` is provided in minutes.
   * `validFor` can be provided in days or minutes, depending on `isValidForInDays` boolean value being set to true or false respectively.
   * The `managerApproval` block, which contains:
     * `condition` - Specifies the approval condition, e.g., ["All", "Any", "Manager"].
     * `required` - Boolean indicating if manager approval is required.
 * The condition based on `ipAddress` should be specified as comma separated IP addresses in CIDR, dotted decimal format or `null`.
 * The `timeOfAccess` can be scheduled based on date, days, both or `null`.
 * The `dateSchedule` should contain the `fromDate`, `toDate` in format of "YYYY-MM-DD HH:MM:SS" and `timezone` as a string from https://en.wikipedia.org/wiki/List_of_tz_database_time_zones. If `dateSchedule` is not required, it has to be set to `null`.
 * The `daysSchedule` should contain the `fromTime`, `toTime` in format of "HH:MM:SS", `timezone` as a string from https://en.wikipedia.org/wiki/List_of_tz_database_time_zones and `days` as a list of strings. If `daysSchedule` is not required, it has to be set to `null`.

* `is_active` - (Optional) Indicates if a policy is active. Boolean value accepts true/false. Default: `true`. 

* `is_draft` - (Optional) Indicates if a policy is a draft. Boolean value accepts true/false. Default: `false`.

* `is_read_only` - (Optional) Indicates if a policy is read only. Boolean value accepts true/false. Default: `false`.


## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the policy with format `policies/{{name}}`

## Import

You can import a policy using any of these accepted formats:

```SH
terraform import britive_policy.new policies/{{name}}
terraform import britive_policy.new {{name}}
```
