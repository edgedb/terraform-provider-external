package main

import (
	"github.com/edgedb/terraform-provider-external/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.New})
}
