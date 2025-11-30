package main

import (
	"context"
	"flag"

	"github.com/britive/terraform-provider-britive/britive"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {

	var debug bool
	flag.BoolVar(&debug, "debug", false, "set this to true if you want to debug the code using delve")
	flag.Parse()

	ctx := context.Background()

	providerserver.Serve(ctx, britive.New, providerserver.ServeOpts{
		Debug:   debug,
		Address: "terraform.example.com/local/britive",
	})

}
