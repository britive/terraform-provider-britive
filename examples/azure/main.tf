terraform {
  required_providers {
    britive = {
      versions = ["0.1"]
      source   = "github.com/britive/britive"
    }
  }
}

provider "britive" {
}

variable "name" {
  default = "Azure-ValueLabs"
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
  status      = "Active"
  user_tag_identity_providers {
    identity_provider {
      id = data.britive_identity_provider.idp.id
    }
  }
}

output "britive_tag_new" {
  value = britive_tag.new
}

resource "britive_tag_member" "new" {
  tag_id   = britive_tag.new.id
  username = "terraformexample1@britive.com"
}

output "britive_tag_member_new" {
  value = britive_tag_member.new
}
