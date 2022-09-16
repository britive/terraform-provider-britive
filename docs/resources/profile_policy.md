# britive_profile_policy Resource

-> This resource is compatible only with enhanced Britive profiles feature.
   Resources britive_profile_identity and britive_profile_tag are replaced by britive_profile_policy. 

!> Please update the approval block, under the condition argument, to include `validFor` and  `isValidForInDays` variable. Existing profile policies should be updated with `validFor`, else any action on the resource will fail with error "Error: PP-0005: Please provide validation time for approval: validFor"   

This resource allows you to create and configure the policy associated to a profile.

## Example Usage

```hcl
resource "britive_profile_policy" "new" {
    # ...
}

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
                        "tag_007",
                    ]
                    userIds = [
                        "bwayne",
                        "rdawes",
                    ]
                }
                isValidForInDays   = true
                notificationMedium = "Email"
                timeToApprove      = 630
                validFor           = 2
            }
            ipAddress    = "192.162.0.0/16,10.10.0.10"
            timeOfAccess = {
                from = "2022-04-29 14:30:00"
                to   = "2022-04-29 20:30:00"
            }
        }
    )
    access_type  = "Allow"
    consumer     = "papservice"   
    is_active    = true
    is_draft     = false
    is_read_only = false
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required, Forces new resource) The identifier of the profile.

* `policy_name` - (Required) The name of the profile-policy.

* `description` - (Optional) A description of the profile-policy.

* `members` - (Optional) Set of members under this policy. This is a JSON formatted string. Includes the usernames of `serviceIdentities`, `tags` and `users`

* `condition` - (Optional) Set of conditions applied to this policy. This is a JSON formatted string. Includes the username for `tags` and `userIds` under `approvers`. The `approval` block also includes the `notificationMedium` and `timeToApprove` in minutes, `validFor` can be provided in days or minutes, depending on `isValidForInDays` boolean value being set to true or false respectively. The condition based on `ipAddress` should be specified as comma separated IP addresses in CIDR or dotted decimal format. The `timeOfAccess` can be a range in format of "YYYY-MM-DD HH:MM:SS" or scheduled daily by passing the range in "HH:MM:SS". 

* `access_type` - (Optional) Type of access the policy provides. This can have two values "Allow"/"Deny". Defaults to "Allow".

* `consumer` - (Optional) A component/entity that will use the policy engine for access decisions. Defaults to "papservice". Do not provide any other value.

* `is_active` - (Optional) Indicates if a policy is active. Boolean value accepts true/false. Defaults to true. 

* `is_draft` - (Optional) Indicates if a policy is a draft. Boolean value accepts true/false. Defaults to false.

* `is_read_only` - (Optional) Indicates if a policy is read only. Boolean value accepts true/false. Defaults to false.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the policy for the profile with format `paps/{{profile_id}}/policies/{{policy_name}}`

## Import

You can import a policy for the profile using any of these accepted formats:

```SH
terraform import britive_profile_policy.new paps/{{profile_id}}/policies/{{policy_name}}
terraform import britive_profile_policy.new {{profile_id}}/{{policy_name}}
```
