# Britive Provider

Britive is a cloud-native security solution that provides centralized Privileged Access Security for cloud-forward enterprises.

This is an overview document for the Britive Terraform Provider hosted by the Terraform registry.  

The Britive provider is used to configure your Britive infrastructure using Terraform. The Britive provider is jointly maintained by:

* The Britive Team

## Example usage

Terraform 0.13 and later:

```hcl
terraform {
  required_providers {
    britive = {
      source = "britive/britive"
      version = "~> 1.0"
    }
  }
}

# Configure the Britive Provider
provider "britive" {
  tenant = "https://company.britive.com"
  token  = "xxxx"
}
```

Terraform 0.12:

```hcl

# Configure the Britive Provider
provider "britive" {
  tenant = "https://company.britive.com"
  token  = "xxxx"
}
```

## Authentication

The Britive provider offers a flexible means of providing credentials for authentication. The following methods are supported, in this order, and explained below:

1. Environment variables
2. Provider Config

### Environment variables

You can provide your credentials via the `BRITIVE_TENANT` and `BRITIVE_TOKEN`, environment variables, representing your Britive Tenant URL (ie. `"https://company.britive.com"`) and Britive API Token, respectively.

```hcl
provider "britive" {}
```

Usage:

```sh
$ export BRITIVE_TENANT="https://company.britive.com"
$ export BRITIVE_TOKEN="xxxx"
$ terraform plan
```

## Argument Reference

In addition to [generic `provider` arguments](https://www.terraform.io/docs/configuration/providers.html)
(e.g. `alias` and `version`), the following arguments are supported in the Britive
 `provider` block:

* `tenant` - (Optional): This is the Britive Tenant URL, for example `https://company.britive.com`. It must be provided, but it can also be sourced from the `BRITIVE_TENANT` environment variable.  

* `token` - (Optional): This is the API Token to interact with your Britive. It must be provided, but it can also be sourced from the `BRITIVE_TOKEN` environment variable.

* `config_path` (Optional): This is the file path for Britive provider configuration. The default configuration path is `~/.britive/tf.config`. It can also be sourced from the `BRITIVE_CONFIG` environment variable.

  A sample Britive configuration file content as below.
  
  ```json
  {
    "tenant": "https://company.britive.com",
    "token": "xxxx"
  }
  ```

**Note:** If you have **both** valid configurations in a config file and provider config, the provider config is used as an override. ie. any provider config will override its counterpart loaded from config file.
