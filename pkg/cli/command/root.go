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

// Package command holds ONOS command-line command implementations.
package command

import (
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// GetRootCommand returns the root CLI command.
func GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onos",
		Short: "ONOS command line client",
	}

	viper.SetDefault("address", ":5150")
	viper.SetDefault("keyPath", certs.Client1Key)
	viper.SetDefault("certPath", certs.Client1Crt)

	cmd.PersistentFlags().StringP("address", "a", viper.GetString("address"), "the controller address")
	cmd.PersistentFlags().StringP("keyPath", "k", viper.GetString("keyPath"), "path to client private key")
	cmd.PersistentFlags().StringP("certPath", "c", viper.GetString("certPath"), "path to client certificate")
	cmd.PersistentFlags().String("config", "", "config file (default: $HOME/.onos/config.yaml)")

	viper.BindPFlag("address", cmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("keyPath", cmd.PersistentFlags().Lookup("keyPath"))
	viper.BindPFlag("certPath", cmd.PersistentFlags().Lookup("certPath"))

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newCompletionCommand())
	cmd.AddCommand(newDevicesCommand())
	cmd.AddCommand(newNetChangesCommand())
	cmd.AddCommand(newRollbackCommand())
	cmd.AddCommand(newChangesCommand())
	cmd.AddCommand(newConfigsCommand())
	cmd.AddCommand(newDeviceTreeCommand())
	return cmd
}

func getConnection(cmd *cobra.Command) *grpc.ClientConn {
	keyPath := getConfig("keyPath")
	certPath := getConfig("certPath")
	address := getConfig("address")
	opts, err := certs.HandleCertArgs(&keyPath, &certPath)
	if err != nil {
		ExitWithError(ExitError, err)
	}
	return northbound.Connect(address, opts...)
}
