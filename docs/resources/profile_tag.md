# britive_profile_tag Resource

This resource allows you to add or remove a tag from a Britive profile.

## Example Usage

```hcl
resource "britive_profile" "new" {
    # ...
}

resource "britive_profile_tag" "new" {
    profile_id = britive_profile.new.id
    tag_name = "My Tag"
    access_period {
        start = "2020-11-01T06:00:00Z"
        end   = "2020-11-05T06:00:00Z"
    }
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` - (Required) The identifier of the profile.

* `tag_name` - (Required) The name of the tag.

* `access_period` - (Optional) The access period of the tag in the Britive profile.

  The format of `access_period` is documented below.

### `access_period` block supports

* `start` - (Required) The start of the access period for the associated tag.

* `end` - (Required) The end of the access period for the associated tag.

## Attribute Reference

In addition to the above arguments, the following attribute is exported.

* `id` - An identifier of the resource with format `paps/{{profileID}}/tags/{{tagID}}`

## Import

You can import a Britive profile using any of these accepted formats:

```SH
$ terraform import britive_profile_tag.new apps/{{app_name}}/paps/{{profile_name}}/tags/{{tag_name}}
$ terraform import britive_profile_tag.new {{app_name}}/{{profile_name}}/{{tag_name}}
```