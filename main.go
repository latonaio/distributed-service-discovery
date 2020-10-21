package main

import (
	"os"

	"bitbucket.org/latonaio/distributed-service-discovery/discovery"
)

func main() {
	command := discovery.NewServiceDiscoveryCommand()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
