package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/synology-community/terraform-provider-synology/synology/provider"
)

//go:generate terraform fmt -recursive ./examples/

//go:generate go tool github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name synology
func main() {
	var debug bool

	flag.BoolVar(
		&debug,
		"debug",
		false,
		"set to true to run the provider with support for debuggers like delve",
	)
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/synology-community/synology",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
