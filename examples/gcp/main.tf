terraform {
  required_providers {
    britive = {
      source  = "britive/britive"
      version = ">= 1.0"
    }
  }
}
source  = "britive/britive"
provider "britive" {
}

variable "name" {
  default = "GCP-ValueLabs"
}

locals {
  resource_name_prefix = "${var.name}-2020-11-02"
}

data "britive_identity_provider" "idp" {
  name = "Britive"
}

resource "britive_tag" "new" {
  name        = "${local.resource_name_prefix}-Tag"
  description = "${local.resource_name_prefix}-Tag"
  identity_provider_id = data.britive_identity_provider.idp.id
}

resource "britive_tag_member" "new" {
  tag_id   = britive_tag.new.id
  username = "terraformexample1@britive.com"
}

data "britive_application" "app" {
  name = var.name
}

resource "britive_profile" "new" {
  app_container_id                 = data.britive_application.app.id
  name                             = "${local.resource_name_prefix}-Profile"
  description                      = "${local.resource_name_prefix}-Profile"
  expiration_duration              = "25m0s"
  extendable                       = false
  notification_prior_to_expiration = "10m0s"
  extension_duration               = "12m30s"
  extension_limit                  = 2
  associations {
    type  = "EnvironmentGroup"
    value = "Test Folder"
  }
}

resource "britive_profile_permission" "new" {
  profile_id      = britive_profile.new.id
  permission_name = "Access Approval Viewer"
  permission_type = "role"
}

resource "britive_profile_tag" "new" {
  profile_id = britive_profile.new.id
  tag_name   = "${local.resource_name_prefix}-Tag"
  access_period {
    start = "2020-11-01T06:00:00Z"
    end   = "2020-11-05T06:00:00Z"
  }
  depends_on = [britive_tag.new, britive_tag_member.new]
}

resource "britive_profile_identity" "new" {
  profile_id = britive_profile.new.id
  username   = "terraformexample2@britive.com"
  access_period {
    start = "2020-11-02T06:00:00Z"
    end   = "2020-11-06T06:00:00Z"
  }
}
