---
subcategory: "Application and Access Profile Management"
layout: "britive"
page_title: "britive_entity_group Resource - britive"
description: |-
  Manages entity group for the Britive provider.
---

# britive_entity_group Resource

This resource allows you to create and configure an application entity of the type "Environment Group".

-> This resource is only supported for Snowflake Standalone, AWS Standalone and Okta applications.

-> For applications created from the Britive console, the first entity must be created through the console so that this resource has a parent under which it can be created. This step is not needed for applications created via the Britive Terraform provider plugin.

## Example Usage

```hcl
# Example: Snowflake Standalone EnvironmentGroup
resource "britive_application" "new_snowflake_standalone" {
    application_type = "Snowflake Standalone"
    user_account_mappings {
      name = "Mobile"
      description = "Mobile"
    }
    properties {
      name = "displayName"
      value = "New Snowflake Standalone"
    }
    properties {
      name = "description"
      value = "New Snowflake Standalone Description"
    }
    properties {
      name = "maxSessionDurationForProfiles"
      value = "1000"
    }
}

resource "britive_entity_group" "new" {
    application_id     = britive_application.new_snowflake_standalone.id
    entity_name        = "My Snowflake Entity Group"
    entity_description = "My Snowflake Entity Group Description"
    parent_id          = britive_application.new_snowflake_standalone.entity_root_environment_group_id
}

# In the same way, entity group can be created for AWS Standalone and Okta applications, as shown in the following example.

# Example: AWS Standalone EnvironmentGroup
resource "britive_entity_group" "AWS_Env_Group" {
    application_id     = "hsgfuysxxausdiuasd"
    entity_name        = "My AWS Entity Group"
    entity_description = "My AWS Entity Group Description"
    parent_id          = "jiashdsdkjxxahdaud"
}

# Example: Okta EnvironmentGroup
resource "britive_entity_group" "AWS_Env_Group" {
    application_id     = "ahgduasdxxaudiau"
    entity_name        = "My Okta Entity Group"
    entity_description = "My Okta Entity Group Description"
    parent_id          = "asjdhuhxxdccudhd"
}

```

## Argument Reference (Snowflake Sandalone, AWS Standalone and Okta)

The following arguments are supported:

* `application_id` - (Required, ForceNew) The identity of the Britive application.

* `entity_name` - (Required) The name of the environment group entity to be created.

* `entity_description` - (Required) Description of the environment group entity.

* `parent_id` - (Required, ForceNew)  The identity of the parent under which the environment group entity will be created.

-> Refer to the `environment_group_ids_names` attribute of the `britive_application` data source to get the set of group IDs and names for an application.

## Attribute Reference

In addition to the above arguments, the following attributes are exported.

* `entity_id` - An identifier of the environment group entity.

* `id` - An identifier of the resource with format `apps/{{application_id}}/root-environment-group/groups/{{entity_id}}`

## Import

You can import an environment group entity using any of these accepted formats:

```sh
terraform import britive_entity_group.new apps/{{application_id}}/root-environment-group/groups/{{entity_id}}
terraform import britive_entity_group.new {{application_id}}/groups/{{entity_id}}
```
