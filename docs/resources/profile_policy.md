# britive_profile_policy Resource

-> When using this version for the first time, you may encounter noisy diffs caused by the reordering of resource argument values. 

This resource allows you to create and configure the policy associated to a profile.

## Example Usage

```hcl
resource "britive_profile_policy" "new" {
    profile_id   = "kbcnp7zk3gp2ddlj232"
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
            approval     = {
                approvers          = {
                    tags    = [
                        "tag_006",
                        "tag_007"
                    ],
                    userIds = [
                        "bwayne",
                        "rdawes"
                    ],
                    channelIds = [
                        "channel_id_01",
                        "channel_id_02"
                    ],
                    slackAppChannels = [
                        "slack_app_channel_id_01",
                        "slack_app_channel_id_02"
                    ],
                    teamsAppChannels = [
                        {
                            team = "team_name_1",
                            channels = [
                                "team_1_channel_name_1",
                                "team_1_channel_name_2"
                            ]
                        },
                        {
                            team = "team_name_2",
                            channels = [
                                "team_2_channel_name_1",
                                "team_2_channel_name_2"
                            ]
                        }
                    ]
                }
                isValidForInDays   = true
                notificationMedium = [
                    "Teams",
                    "Email"
                ]
                managerApproval = {
                    condition = "All"
                    required = true
                }
                timeToApprove      = 630
                validFor           = 2
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
    access_type  = "Allow"
    consumer     = "papservice"   
    is_active    = true
    is_draft     = false
    is_read_only = false
    associations {
      type  = "Environment"
      value = "QA Subscription"
    }
    associations {
      type  = "EnvironmentGroup"
      value = "Development"
    }
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required, ForceNew) The identifier of the profile.

* `policy_name` - (Required) The name of the profile-policy.

* `description` - (Optional) A description of the profile-policy.

* `members` - (Optional) Set of members under this policy. This is a JSON formatted string. Includes the usernames of `serviceIdentities`, `tags` and `users`

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

* `access_type` - (Optional) Type of access the policy provides. This can have two values "Allow"/"Deny". Default: `"Allow"`.

* `consumer` - (Optional) A component/entity that will use the policy engine for access decisions. Default: `"papservice"`. Do not provide any other value.

* `is_active` - (Optional) Indicates if a policy is active. Boolean value accepts true/false. Default: `true`. 

* `is_draft` - (Optional) Indicates if a policy is a draft. Boolean value accepts true/false. Default: `false`.

* `is_read_only` - (Optional) Indicates if a policy is read only. Boolean value accepts true/false. Default: `false`.

* `associations` - (Optional) The set of associations for the Britive profile policy.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the policy for the profile with format `paps/{{profile_id}}/policies/{{policy_name}}`

## Import

You can import a policy for the profile using any of these accepted formats:

```SH
terraform import britive_profile_policy.new paps/{{profile_id}}/policies/{{policy_name}}
terraform import britive_profile_policy.new {{profile_id}}/{{policy_name}}
```
