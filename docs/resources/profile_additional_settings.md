# britive_profile_additional_settings Resource

This resource allows you to create and configure the additional settings associated to a profile.

-> This resource is only supported for GCP, GCP Standalone and Azure applications.

## Example Usage

```hcl
data "britive_application" "my_app" {
    name = "My Application"
}

resource "britive_profile" "new" {
    app_container_id                 = data.britive_application.my_app.id
    name                             = "My Profile"
    description                      = "My Profile Description"
    disabled            = false
    expiration_duration = "1h0m0s"
    extendable          = false
    associations {
        type  = "EnvironmentGroup"
        value = "Banking"
    }
}

resource "britive_profile_additional_settings" "new" {
    profile_id                     = britive_profile.new.id
    console_access                 = false
    programmatic_access            = true
    project_id_for_service_account = "my-project-id-123"
    use_app_credential_type        = false
}

resource "britive_profile_additional_settings" "new1" {
    profile_id                     = "q4fxxxxxxxa2qvny9qz4"
    console_access                 = true
    programmatic_access            = true
    project_id_for_service_account = ""
    use_app_credential_type        = true
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required, ForceNew) The identifier of the profile.

* `use_app_credential_type` - (Optional) Boolean attribute, to choose whether to inherit the credential type from the application.

* `console_access` - (Optional) Boolean attribute, to provide the console access to the profile or not. Overriden if `use_app_credential_type` is set to true.

* `programmatic_access` - (Optional) Boolean attribute, to provide the programmatic access to the profile or not. Overriden if `use_app_credential_type` is set to true.

* `project_id_for_service_account` - (Optional) The project id for creating service accounts. If set to null or empty string, picks up the app default project id. Supported only for GCP and GCP Standalone applications.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with the format `paps/{{profile_id}}/additional-settings`

## Import

You can import Britive profile additional settings using the accepted format:

```sh
terraform import britive_profile_additional_settings.new paps/{{profile_id}}/additional-settings
```
