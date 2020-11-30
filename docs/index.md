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

In this configuration, the environment variables `BRITIVE_HOST` and `BRITIVE_TOKEN` are used to detect the host and token on the host machine.

```hcl
provider "britive" {
}
```

### Statically-defined variables

In this configuration, it is required to **statically** define host name and token as variables.

```hcl
provider "britive" {
  host = "https://britive.api.local/api"
  token = "${file("~/britive-token.config")}"
}
```

## Argument Reference

The following arguments are supported:

* `host`  (Required): The API URL for the Britive API.  

  For example, https://britive.local/api

* `token`  (Required): The API token required to authenticate the Britive API. 

  For example, `iw8ECAdxhF/T/fyX/O3bCBV60TkOopdu5JEE0UY1mSw=`

* `config_path` (Optional): The Britive configuration (holding host and token as JSON attributes) file path. The default configuration path is `~/.britive/config`. 

  A sample config file content is shown here. 
  ```
  {
    "host": "https://britive.local/api",
    "token": "iw8ECAdxhF/T/fyX/O3bCBV60TkOopdu5JEE0UY1mSw="
  }
  ```
**Note:** If host and token are passed either statically or through environment variables, they will be overwritten on the config file.