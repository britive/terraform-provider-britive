---
layout: "Britive"
page_title: "Provider: Britive"
description: |-
  The Britive provider is used to interact with the resources supported by Britive API. The provider needs to be configured with the proper credentials before it can be used.
---
# Britive Provider

Britive is a cloud-native security solution built for the most demanding cloud-forward enterprises.

## Example Usage
```hcl
provider "britive" {
}
```
## Authentication
There are generally two ways to configure the Britive provider.

### From environment variables

Detection of host and token from environment variables BRITIVE_HOST and BRITIVE_TOKEN on the host machine

```hcl
provider "britive" {
}
```

### Statically defined credentials
TLS certificate credentials:
Another way is **statically** define host name and token

```hcl
provider "britive" {
  host = "https://britive.api.local"
  token = "${file("~/britive-token.config")}"
}
```