# Britive Provider

Britive is a cloud-native security solution that provides centralized Privileged Access Security for cloud-forward enterprises. 

This is an overview document for the Britive Terraform Provider hosted by the Terraform registry.  

The Britive provider is used to configure your Britive infrastructure using Terraform. The Britive provider is jointly maintained by:

* The Britive Team

## Example usage

```hcl
provider "britive" {
}
```
## Authentication

There are two ways to configure the Britive Provider:

1. Using environment variables
2. Using statically-defined variables

### Configuring using environment variables

In this configuration, the environment variables `BRITIVE_TENANT` and `BRITIVE_TOKEN` are used to detect the tenant and token on the host machine.

```hcl
provider "britive" {
}
```

### Statically-defined variables

In this configuration, it is required to **statically** define tenant name and token as variables.

```hcl
provider "britive" {
  tenant = "https://britive.api.local"
  token = "iw8ECAdxhF/T/fyX/O3bCBV60TkOopdu5JEE0UY1mSw="
}
```

## Argument Reference

The following arguments are supported:

* `tenant`  (Required): The API URL for the Britive API.  

  For example, https://britive.local

* `token`  (Required): The API token required to authenticate the Britive API. 

  For example, `iw8ECAdxhF/T/fyX/O3bCBV60TkOopdu5JEE0UY1mSw=`

* `config_path` (Optional): The Britive configuration (holding tenant and token as JSON attributes) file path. The default configuration path is `~/.britive/config`. 

  A sample config file content is shown here. 
  ```
  {
    "tenant": "https://britive.local",
    "token": "iw8ECAdxhF/T/fyX/O3bCBV60TkOopdu5JEE0UY1mSw="
  }
  ```
**Note:** If tenant and token are passed either statically or through environment variables, they will be overwritten on the config file.