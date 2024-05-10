package main

import (
	"context"
	"flag"

	log "github.com/sirupsen/logrus"

	"github.com/appkins/terraform-provider-synology/synology/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name synology

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ./data/api.yaml

//go:generate terraform fmt -recursive ./examples/

//go:generate go run github.com/rjeczalik/interfaces/cmd/structer
func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/appkins/synology",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
