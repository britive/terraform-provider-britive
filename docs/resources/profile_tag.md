| subcategory  | layout    | page_title                    | description                                            |
| ------------ | --------- | ----------------------------- | ------------------------------------------------------ |
| Profile Tag   | Britive   | Britive: britive_profile_tag   | The Britive Profile Tag adds a tag to the Britive profile. |

# britive\_profile\_tag

Adds a tag to the Britive profile.

## Example Usage

```hcl
resource "britive_profile" "new" {
    # ...
}

resource "britive_profile_tag" "new" {
    profile_id = britive_profile.new.id
    tag = "My Tag"
    access_period {
        start = "2020-11-01T06:00:00Z"
        end   = "2020-11-05T06:00:00Z"
    }
}
```

## Argument Reference

The following arguments are supported:

* `profile_id` (Required): The identifier of the profile.

  For example: `britive_profile.new.id`

* `tag` (Required): The name of the tag.

  For example: `My Tag`

* `access_period` (Optional): The access period of tag in the profile. 

  The format of access_period is given below.


### `access_period` block supports

* `start` - The start of the access period for the associated tag.

* `end` - The end of the access period for the associated tag.

## Attribute Reference

In addition to the above argument, the following attribute is exported.

* `id` - An identifier of the resource with format `paps/{{profileID}}/user-tags/{{tagID}}`

## Import

You can import a profile tag using any of these accepted formats:

```
$ terraform import britive_profile_tag.new paps/{{profile_name}}/user-tags/{{tag_name}}
$ terraform import britive_profile_tag.new {{profile_name}}/{{tag_name}}
```