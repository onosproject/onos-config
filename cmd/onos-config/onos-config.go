// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package onos-config is the main entry point to the ONOS configuration subsystem.

It connects to devices through a Southbound gNMI interface and
gives a gNMI interface northbound for other systems to connect to, and an
Admin service through gRPC

Arguments
-allowUnvalidatedConfig <allow configuration for devices without a corresponding model plugin>

-modelPlugin (repeated) <the location of a shared object library that implements the Model Plugin interface>

-caPath <the location of a CA certificate>

-keyPath <the location of a client private key>

-certPath <the location of a client certificate>


See ../../docs/run.md for how to run the application.
*/
package main

import (
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var log = logging.GetLogger("main")

// The main entry point
func main() {
	if err := getRootCommand().Execute(); err != nil {
		println(err)
		os.Exit(1)
	}
}

func getRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onos-config",
		Short: "ONOS configuration subsystem",
		RunE:  runRootCommand,
	}
	cmd.Flags().Int("port", 5150, "gRPC port")
	cmd.Flags().Bool("allowUnvalidatedConfig", false, "allow configuration for devices without a corresponding model plugin")
	cmd.Flags().String("caPath", "", "path to CA certificate")
	cmd.Flags().String("keyPath", "", "path to client private key")
	cmd.Flags().String("certPath", "", "ppath to client certificate")
	cmd.Flags().String("topoEndpoint", "onos-topo:5150", "topology service endpoint")
	cmd.Flags().UintSlice("plugin-port", []uint{}, "configuration model plugin ports")
	return cmd
}

func runRootCommand(cmd *cobra.Command, args []string) error {
	allowUnvalidatedConfig, _ := cmd.Flags().GetBool("allowUnvalidatedConfig")
	caPath, _ := cmd.Flags().GetString("caPath")
	keyPath, _ := cmd.Flags().GetString("keyPath")
	certPath, _ := cmd.Flags().GetString("certPath")
	topoEndpoint, _ := cmd.Flags().GetString("topoEndpoint")
	pluginPorts, _ := cmd.Flags().GetUintSlice("plugin-port")

	log.Info("Starting onos-config")

	cfg := manager.Config{
		CAPath:                 caPath,
		KeyPath:                keyPath,
		CertPath:               certPath,
		GRPCPort:               5150,
		TopoAddress:            topoEndpoint,
		AllowUnvalidatedConfig: allowUnvalidatedConfig,
		PluginPorts:            pluginPorts,
	}

	mgr := manager.NewManager(cfg)

	mgr.Run()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	mgr.Close()
	return nil
}
