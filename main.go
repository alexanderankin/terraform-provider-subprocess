package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "enable provider debugging")
	flag.Parse()

	err := providerserver.Serve(context.Background(), NewProvider(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/alexanderankin/subprocess",
		Debug:   *debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
