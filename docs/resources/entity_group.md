# britive_entity_group Resource

This resource allows you to create and configure an application entity of the type "Environment Group".

-> This resource is only supported for Snowflake Standalone applications.

-> For applications created from the Britive console, the first entity must be created through the console so that this resource has a parent under which it can be created. This step is not needed for applications created via the Britive Terraform provider plugin.

## Example Usage

```hcl
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
    entity_name        = "My Entity Group"
    entity_description = "My Entity Group Description"
    parent_id          = britive_application.new_snowflake_standalone.entity_root_environment_group_id
}

resource "britive_entity_group" "new1" {
    application_id     = britive_application.new_snowflake_standalone.id
    entity_name        = "My Group"
    entity_description = "My Group Description"
    parent_id          = britive_entity_group.new.entity_id
}

resource "britive_entity_group" "new2" {
    application_id     = "h59ih6p1537xxxxxxxxx"
    entity_name        = "My New Group"
    entity_description = "My New Group Description"
    parent_id          = "qawerxxxx1efr43xxx"
}
```

## Argument Reference

The following arguments are supported:

* `application_id` - (Required, ForceNew) The identity of the Britive application.

* `entity_name` - (Required) The name of the environment group entity to be created.

* `entity_description` - (Required) Description of the environment group entity.

* `parent_id` - (Required, ForceNew)  The identity of the parent under which the environment group entity will be created.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `entity_id` - An identifier of the environment group entity.

* `id` - An identifier of the resource with format `apps/{{application_id}}/root-environment-group/groups/{{entity_id}}`

## Import

You can import an environment group entity using any of these accepted formats:

```SH
terraform import britive_entity_group.new apps/{{application_id}}/root-environment-group/groups/{{entity_id}}
terraform import britive_entity_group.new {{application_id}}/groups/{{entity_id}}
```
