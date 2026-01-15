package main

import (
	"context"
	"log"

	"github.com/brad-jones/terraform-provider-denobridge/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "example.registry.local/brad-jones/denobridge",
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
