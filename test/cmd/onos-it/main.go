package main

import (
	"fmt"
	"github.com/onosproject/onos-config/test/integration"
	"github.com/onosproject/onos-config/test/runner"
	"os"
)

func main() {
	cmd := runner.GetCommand(integration.Registry)
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
