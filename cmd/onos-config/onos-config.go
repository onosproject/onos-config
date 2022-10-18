// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
	"os"
	"os/signal"
	"syscall"

	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/spf13/cobra"
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
	cmd.Flags().StringSlice("plugin", []string{}, "configuration model plugin (name:port)")
	return cmd
}

func runRootCommand(cmd *cobra.Command, args []string) error {
	caPath, _ := cmd.Flags().GetString("caPath")
	keyPath, _ := cmd.Flags().GetString("keyPath")
	certPath, _ := cmd.Flags().GetString("certPath")
	topoEndpoint, _ := cmd.Flags().GetString("topoEndpoint")
	plugins, _ := cmd.Flags().GetStringSlice("plugin")

	log.Infow("Starting onos-config",
		"CAPath", caPath,
		"KeyPath", keyPath,
		"CertPath", certPath,
		"GRPCPort", 5150,
		"TopoAddress", topoEndpoint,
		"Plugins", plugins,
	)

	cfg := manager.Config{
		CAPath:      caPath,
		KeyPath:     keyPath,
		CertPath:    certPath,
		GRPCPort:    5150,
		TopoAddress: topoEndpoint,
		Plugins:     plugins,
	}

	mgr := manager.NewManager(cfg)

	mgr.Run()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	mgr.Close()
	return nil
}
