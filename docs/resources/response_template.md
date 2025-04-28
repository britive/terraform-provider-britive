# britive_response_template Resource

The `britive_response_template` resource allows you to create, update, and manage response templates in Britive.

## Example Usage

```hcl
resource "britive_response_template" "example" {
    name                      = "example_response_template"
    description               = "This is an example response template."
    is_console_access_enabled = true
    show_on_ui                = false
    template_data             = "The user {{name}} has the role {{role}}."
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the response template. Must be unique.
* `description` - (Optional) A description of the response template. Default is an empty string.
* `is_console_access_enabled` - (Optional) A boolean flag to enable console access for the response template. Default is `false`.
* `show_on_ui` - (Optional) A boolean flag to determine if the template is visible in the UI. Default is `false`.
* `template_data` - (Required) The content of the response template. It can include placeholders such as `{{name}}` and `{{role}}`.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `template_id` - The unique identifier of the response template.

## Import

Response templates can be imported using their `template_id`:

```sh
terraform import britive_response_template.example {{template_id}}
```
