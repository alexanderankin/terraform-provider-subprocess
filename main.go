package main

import (
	"context"
	"flag"
	"log"
	"os"

	. "github.com/alexanderankin/terraform-provider-subprocess/internal"
	"github.com/alexanderankin/terraform-provider-subprocess/internal/supervisor"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--subprocess-supervisor" {
		os.Exit(supervisor.BecomeSupervisor())
	}
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
