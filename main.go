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
	debug := flag.Bool("debug", false, "enable provider debugging")
	supervisorMode := flag.Bool("subprocess-supervisor", false, "run the process supervisor")
	flag.Parse()

	if *supervisorMode {
		os.Exit(supervisor.BecomeSupervisor())
	}

	err := providerserver.Serve(context.Background(), NewProvider(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/alexanderankin/subprocess",
		Debug:   *debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
