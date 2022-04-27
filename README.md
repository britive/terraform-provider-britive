Terraform Provider for Britive
==================

- Website: https://www.terraform.io
- Documentation: https://registry.terraform.io/providers/britive/britive/latest/docs
<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Maintainers
-----------

This provider plugin is maintained by:

* The Britive Team at [Britive](https://www.britive.com/)

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13.7+
- [Go](https://golang.org/doc/install) >= 1.15


Using the provider
----------------------

See the [Britive Provider documentation](https://registry.terraform.io/providers/britive/britive/latest/docs) to get started using the
Britive provider.


Upgrading the provider
----------------------

The Britive provider doesn't upgrade automatically once you've started using it. After a new release you can run

```bash
terraform init -upgrade
```

to upgrade to the latest stable version of the Britive provider. See the [Terraform website](https://www.terraform.io/docs/configuration/providers.html#provider-versions)
for more information on provider upgrades, and how to set version constraints on your provider.

Building the provider
---------------------

Clone repository to: `$GOPATH/src/github.com/britive/terraform-provider-britive`

```sh
$ mkdir -p $GOPATH/src/github.com/britive; cd $GOPATH/src/github.com/britive
$ git clone git@github.com:britive/terraform-provider-britive
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/britive/terraform-provider-britive
$ make build
```

Developing the provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org)
installed on your machine (version 1.15.0+ is *required*). You can use [goenv](https://github.com/syndbg/goenv)
to manage your Go version. You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH),
as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`.
This will build the provider and put the provider binary in the `$GOPATH/bin`
directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-britive
...
```
